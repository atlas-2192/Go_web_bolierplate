package bee_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/web-starter-app/lib/bee"
	"github.com/stretchr/testify/require"
)

func TestParseParamsQueryParameters(t *testing.T) {
	queryArgs := url.Values{}
	queryArgs.Add("foo", "1")
	queryArgs.Add("bar", "2")
	r := httptest.NewRequest("GET", fmt.Sprintf("/somewhere?%s", queryArgs.Encode()), nil)

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t, map[string]any{"foo": "1", "bar": "2"}, params)
}

func TestParseParamsFormURLEncoded(t *testing.T) {
	queryArgs := url.Values{}
	queryArgs.Add("foo", "1")
	queryArgs.Add("bar", "2")

	r := httptest.NewRequest("POST", "/somewhere", strings.NewReader(queryArgs.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t, map[string]any{"foo": "1", "bar": "2"}, params)
}

func TestParseParamsApplicationJSON(t *testing.T) {
	postData := map[string]any{"foo": "1", "bar": "2"}
	postBody, err := json.Marshal(postData)
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/somewhere", bytes.NewReader(postBody))
	r.Header.Set("Content-Type", "application/json")

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t, postData, params)
}

// bee.Must(err, 500) ?

func TestHandlerBuilderHandlerSetsEtag(t *testing.T) {
	hb := &bee.HandlerBuilder[struct{}]{}
	handler := hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, _ struct{}, params map[string]any) error {
		w.Write([]byte("Hello, world"))
		return nil
	})

	r := httptest.NewRequest("GET", "/", nil)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, r)

	require.Equal(t, "Hello, world", responseRecorder.Body.String())
	require.Equal(t, `W/"SufDtqwL7_Zx76jPVzhhUcBuWMpTp42D82EHMWzsEl8="`, responseRecorder.Header().Get("ETag"))
}
