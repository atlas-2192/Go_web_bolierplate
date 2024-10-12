package httpz

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/securecookie"
	"github.com/jackc/pgx/v5"
)

type RequestUser struct {
	ID       uuid.UUID
	Username string
	System   bool
}

type RequestLoginSession struct {
	ID   uuid.UUID
	User *RequestUser
}

// loginSessionHandler returns a middleware handler that loads the login session from the request cookie.
func loginSessionHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			env := ctx.Value(ctxKeyEnvironment).(*environment)
			loginSession := &RequestLoginSession{}
			ctx = context.WithValue(ctx, ctxKeySession, loginSession)

			cookie, err := r.Cookie(env.sessionCookieTemplate.Name)
			if err != nil {
				// Only expected error is http.ErrNoCookie.
				if !errors.Is(err, http.ErrNoCookie) {
					env.logger.Warn().Err(err).Msg("unexpected error getting session cookie")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			var loginSessionID uuid.UUID
			err = env.secureCookie.Decode(env.sessionCookieTemplate.Name, cookie.Value, &loginSessionID)
			if err != nil {
				var secureCookieError securecookie.Error
				if errors.As(err, &secureCookieError) && secureCookieError.IsDecode() {
					env.logger.Warn().Err(err).Msg("error decoding session cookie")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			user := &RequestUser{}
			err = env.dbpool.QueryRow(ctx,
				`select login_sessions.id, users.id, users.username, users.system
from login_sessions
	join users on login_sessions.user_id=users.id
where login_sessions.id=$1`,
				loginSessionID,
			).Scan(&loginSession.ID, &user.ID, &user.Username, &user.System)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// invalid session ID
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
			loginSession.User = user

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

// setLoginSessionCookie sets the login session cookie in the response.
func setLoginSessionCookie(w http.ResponseWriter, r *http.Request, loginSessionID uuid.UUID) error {
	ctx := r.Context()
	env := ctx.Value(ctxKeyEnvironment).(*environment)
	cookie := &(*env.sessionCookieTemplate)

	var err error
	cookie.Value, err = env.secureCookie.Encode(cookie.Name, loginSessionID)
	if err != nil {
		return err
	}

	http.SetCookie(w, cookie)

	return nil
}

// clearLoginSessionCookie clears the login session cookie in the response.
func clearLoginSessionCookie(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := ctx.Value(ctxKeyEnvironment).(*environment)
	cookie := &(*env.sessionCookieTemplate)
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)
}

// getLoginSession returns the login session from the request context.
func getLoginSession(ctx context.Context) *RequestLoginSession {
	return ctx.Value(ctxKeySession).(*RequestLoginSession)
}

// requireCurrentUserHandler returns a middleware handler that redirects to redirectURL if there is no current user.
func requireCurrentUserHandler(redirectURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			loginSession := getLoginSession(ctx)
			if loginSession.User == nil {
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// requireSystemUserHandler returns a middleware handler that redirects to redirectURL if there is no current user or the
// user is not a system user.
func requireSystemUserHandler(redirectURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			loginSession := getLoginSession(ctx)
			if loginSession.User == nil || !loginSession.User.System {
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
