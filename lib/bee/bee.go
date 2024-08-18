// Package bee provides a simple HTTP handler with functionality that is inconvenient to implement in middleware.
//
// It provides two primary features. First, is easier error handling. Handlers can return errors which will be handled
// by a list of error handlers that will be called when an error occurs. Second, it automatically sets the ETag header
// based on the digest of the response body.
//
// These features may seem entirely unrelated but they are both related because the response body must be buffered in
// its entirety. For error handling an error may occur after some of the response has been written and the response
// needs to be replaced. For ETag the response body must be buffered so that the digest can be calculated and set in the
// headers.
package bee

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

type bufferedResponseWriter struct {
	w          http.ResponseWriter
	b          *bytes.Buffer
	statusCode int
}

func (brw *bufferedResponseWriter) Header() http.Header {
	return brw.w.Header()
}

func (brw *bufferedResponseWriter) Write(p []byte) (int, error) {
	return brw.b.Write(p)
}

func (brw *bufferedResponseWriter) WriteHeader(statusCode int) {
	brw.statusCode = statusCode
}

func (brw *bufferedResponseWriter) Reset() {
	brw.b.Reset()
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error) (bool, error)

// HandlerBuilder is used to build Handlers with shared functionality. HandlerBuilder must not be mutated after any
// methods have been called.
type HandlerBuilder[T any] struct {
	// Key used to get the env from the request context. If nil then a zero value T is passed as the env to the handler.
	CtxKeyEnv any

	// ParseParams builds the parameters to pass to the handler function. If nil then the package ParseParams function is
	// used.
	ParseParams func(*http.Request) (map[string]any, error)

	// ErrorHandlers are called one at a time until one returns true. If none return true or one returns an error then a
	// generic HTTP 500 error is returned.
	ErrorHandlers []ErrorHandler

	// ETagDigestFilter is used to filter out parts of the response body that should not be included in the automatic ETag
	// digest. This is useful for filtering out dynamic content such as CSRF tokens. If nil then the entire response body
	// is used.
	ETagDigestFilter *regexp.Regexp
}

// New returns a new http.Handler that calls fn. If fn returns an error then the error is passed to the ErrorHandlers.
func (hb *HandlerBuilder[T]) New(fn func(ctx context.Context, w http.ResponseWriter, r *http.Request, env T, params map[string]any) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		b := bufPool.Get().(*bytes.Buffer)
		defer func() {
			b.Reset()
			bufPool.Put(b)
		}()

		brw := &bufferedResponseWriter{
			w: w,
			b: b,
		}

		env, _ := ctx.Value(hb.CtxKeyEnv).(T)

		var parseParams func(*http.Request) (map[string]any, error)
		if hb.ParseParams != nil {
			parseParams = hb.ParseParams
		} else {
			parseParams = ParseParams
		}

		params, err := parseParams(r)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		err = fn(ctx, brw, r, env, params)
		if err != nil {
			brw.Reset()
			for _, eh := range hb.ErrorHandlers {
				handled, err := eh(brw, r, err)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				if handled {
					return
				}
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		// Even though the net/http package will set the Content-Type header if it is not set, we do it here so that
		// Content-Type is available for middleware such as chi/middleware/Compress.
		if brw.Header().Get("Content-Type") == "" {
			brw.Header().Set("Content-Type", http.DetectContentType(brw.b.Bytes()))
		}

		if r.Method == http.MethodGet && brw.Header().Get("ETag") == "" {
			digest := sha256.New()
			if hb.ETagDigestFilter == nil {
				digest.Write(brw.b.Bytes())
			} else {
				buf := brw.b.Bytes()
				for len(buf) > 0 {
					loc := hb.ETagDigestFilter.FindIndex(buf)
					if loc == nil {
						digest.Write(buf)
						buf = buf[len(buf):]
					} else {
						digest.Write(buf[:loc[0]])
						buf = buf[loc[1]:]
					}
				}
			}

			bodyDigest := digest.Sum(nil)
			etag := `W/"` + base64.URLEncoding.EncodeToString(bodyDigest[:]) + `"`

			if r.Header.Get("If-None-Match") == etag {
				brw.w.WriteHeader(http.StatusNotModified)
				return
			}

			brw.w.Header().Set("ETag", etag)
		}

		if brw.statusCode != 0 {
			brw.w.WriteHeader(brw.statusCode)
		}
		brw.b.WriteTo(brw.w)
	})
}

// ParseParams parses the request parameters from the Chi route parameters, the URL query string, and the request
// body. The request body can be parsed for application/json, application/x-www-form-urlencoded, and
// multipart/form-data.
//
// When the request is URL encoded, the parameters are parsed as follows:
//   - foo=bar -> map[string]any{"foo": "bar"}
//   - foo[]=bar -> map[string]any{"foo": []string{"bar"}}
//   - foo[]=bar&foo[]baz -> map[string]any{"foo": []string{"bar", "baz"}}
//   - foo[bar]=baz -> {"foo": {"bar": "baz"}}
//   - foo[bar][]=baz&foo[bar][]=qux -> {"foo": {"bar": []string{"baz", "qux"}}}
func ParseParams(r *http.Request) (map[string]any, error) {
	params := make(map[string]any)

	if chiContext := chi.RouteContext(r.Context()); chiContext != nil {
		routeParams := chi.RouteContext(r.Context()).URLParams
		for i := 0; i < len(routeParams.Keys); i++ {
			params[routeParams.Keys[i]] = routeParams.Values[i]
		}
	}

	addValuesToParams := func(m map[string][]string) error {
		for key, values := range m {
			keyParts := splitParamName(key)
			setNested(params, keyParts, values)
		}

		return nil
	}

	addValuesToParams(r.URL.Query())

	contentType := r.Header.Get("Content-Type")
	switch {
	case contentType == "application/json":
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		err := decoder.Decode(&params)
		if err != nil {
			return nil, err
		}
	case contentType == "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}
		addValuesToParams(r.PostForm)
	case strings.HasPrefix(contentType, "multipart/form-data"):
		err := r.ParseMultipartForm(5 * 1024 * 1024)
		if err != nil {
			return nil, err
		}
		addValuesToParams(r.MultipartForm.Value)
	}

	return params, nil
}

const paramNameArrayPart = "[]"

var splitParamNameInitialRegexp = regexp.MustCompile(`\A\w+`)
var splitParamNameNestedRegexp = regexp.MustCompile(`\A\[\w*\]`)

func splitParamName(paramName string) []string {
	loc := splitParamNameInitialRegexp.FindStringIndex(paramName)
	if loc == nil {
		return []string{paramName}
	}

	if loc[1] == len(paramName) {
		return []string{paramName}
	}

	parts := make([]string, 0, 4)
	parts = append(parts, paramName[:loc[1]])
	originalParamName := paramName
	paramName = paramName[loc[1]:]

	for {
		loc = splitParamNameNestedRegexp.FindStringIndex(paramName)
		if loc == nil {
			return []string{originalParamName}
		}

		if loc[1] == 2 { // [] -> []
			if len(paramName) > loc[1] {
				return []string{originalParamName}
			}
			parts = append(parts, paramName[:loc[1]])
		} else { // [foo] -> foo
			parts = append(parts, paramName[loc[0]+1:loc[1]-1])
		}

		if loc[1] == len(paramName) {
			return parts
		}

		paramName = paramName[loc[1]:]
	}

}

func setNested(params map[string]any, keyParts []string, values []string) {
	if len(keyParts) == 1 {
		params[keyParts[0]] = values[len(values)-1]
		return
	}

	// Since len(keyParts) > 1, keyParts[1] is always valid. Check if it is an array part.
	if keyParts[1] == paramNameArrayPart {
		params[keyParts[0]] = values
		return
	}

	if nestedMap, ok := params[keyParts[0]].(map[string]any); ok {
		setNested(nestedMap, keyParts[1:], values)
	} else {
		nestedMap := make(map[string]any)
		params[keyParts[0]] = nestedMap
		setNested(nestedMap, keyParts[1:], values)
	}
}
