// Package httpz provides the application's HTTP handler and other functionality.
//
// It is named httpz to avoid a name conflict with the standard library's http package.
package httpz

import (
	"context"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Use when setting something though the request context.
type ctxRequestKey int

const (
	_ ctxRequestKey = iota
	ctxKeyEnvironment
	ctxKeySession
)

type environment struct {
	dbpool *pgxpool.Pool
	logger *zerolog.Logger

	secureCookie          *securecookie.SecureCookie
	sessionCookieTemplate *http.Cookie
}

// setContextValue returns a middleware handler that sets a value in the request context.
func setContextValue(key any, value any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, key, value)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}
