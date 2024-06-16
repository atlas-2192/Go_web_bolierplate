package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	"github.com/jackc/pgxutil"
	"github.com/jackc/web-starter-app/db"
	"github.com/jackc/web-starter-app/view"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Use when setting something though the request context.
type ctxRequestKey int

const (
	_ ctxRequestKey = iota
	ctxKeyServer
	ctxKeySession
)

type Server struct {
	handler       http.Handler
	listenAddress string
	server        *http.Server

	dbsession *db.Session
	logger    *zerolog.Logger

	secureCookie          *securecookie.SecureCookie
	sessionCookieTemplate *http.Cookie
}

func NewServer(
	listenAddress string,
	dbsession *db.Session,
	logger *zerolog.Logger,
	csrfKey []byte,
	secureCookies bool,
	cookieAuthenticationKey []byte,
	cookieEncryptionKey []byte,
) (*Server, error) {

	router := chi.NewRouter()

	server := &Server{
		handler:       router,
		listenAddress: listenAddress,
		dbsession:     dbsession,
		logger:        logger,
		secureCookie:  securecookie.New(cookieAuthenticationKey, cookieEncryptionKey),
		sessionCookieTemplate: &http.Cookie{
			Name:     "web-starter-app-session",
			Path:     "/",
			Secure:   secureCookies,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	}

	router.Use(middleware.Compress(5))
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)

	router.Use(hlog.NewHandler(*logger))
	router.Use(hlog.RequestIDHandler("request_id", "x-request-id"))
	router.Use(hlog.MethodHandler("method"))
	router.Use(hlog.URLHandler("url"))
	router.Use(hlog.RemoteAddrHandler("remote_ip"))
	router.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("HTTP request")
	}))

	router.Use(middleware.Recoverer)

	router.Use(setContextValue(ctxKeyServer, server))

	CSRF := csrf.Protect(csrfKey, csrf.Path("/"), csrf.Secure(secureCookies))
	router.Use(CSRF)

	router.Use(loginSessionHandler())

	router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		view.LoginPage(csrf.Token(r)).Render(ctx, w)
	})

	router.Post("/login/submit", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		server := ctx.Value(ctxKeyServer).(*Server)
		user, err := db.GetUserByUsername(ctx, server.dbsession, r.FormValue("username"))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// TODO - rerender form
				http.Error(w, "user name not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		dbpool := db.DBPool(server.dbsession)
		now := time.Now()
		loginSessionID, err := pgxutil.InsertRowReturning(ctx, dbpool, "login_sessions", map[string]any{
			"id":                            uuid.Must(uuid.NewV7()),
			"user_id":                       user.ID,
			"user_agent":                    zeronull.Text(r.UserAgent()),
			"login_time":                    now,
			"login_request_id":              middleware.GetReqID(ctx),
			"approximate_last_request_time": now,
		},
			"id",
			pgx.RowTo[uuid.UUID],
		)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		setLoginSessionCookie(w, r, loginSessionID)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		server := ctx.Value(ctxKeyServer).(*Server)
		now, err := db.GetCurrentTime(ctx, server.dbsession)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var name string
		loginSession := ctx.Value(ctxKeySession).(*RequestLoginSession)
		if loginSession.User != nil {
			name = loginSession.User.Username
		} else {
			name = "world"
		}

		view.Hello(name, now).Render(r.Context(), w)
	})

	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) Serve() error {
	s.server = &http.Server{
		Addr:    s.listenAddress,
		Handler: s.handler,
	}

	s.logger.Info().Str("listen_address", s.listenAddress).Msg("Starting HTTP server")

	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Stopping HTTP server")
	s.server.SetKeepAlivesEnabled(false)
	err := s.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// setContextValue returns a middleware handler that sets a value in the request context.
func setContextValue(key ctxRequestKey, value any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, key, value)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

type RequestUser struct {
	ID       uuid.UUID
	Username string
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
			server := ctx.Value(ctxKeyServer).(*Server)
			loginSession := &RequestLoginSession{}
			ctx = context.WithValue(ctx, ctxKeySession, loginSession)

			cookie, err := r.Cookie(server.sessionCookieTemplate.Name)
			if err != nil {
				// Only expected error is http.ErrNoCookie.
				if !errors.Is(err, http.ErrNoCookie) {
					server.logger.Warn().Err(err).Msg("unexpected error getting session cookie")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			var loginSessionID uuid.UUID
			err = server.secureCookie.Decode(server.sessionCookieTemplate.Name, cookie.Value, &loginSessionID)
			if err != nil {
				var secureCookieError securecookie.Error
				if errors.As(err, &secureCookieError) && secureCookieError.IsDecode() {
					server.logger.Warn().Err(err).Msg("error decoding session cookie")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			dbpool := db.DBPool(server.dbsession)

			user := &RequestUser{}
			err = dbpool.QueryRow(ctx,
				`select login_sessions.id, users.id, users.username
from login_sessions
	join users on login_sessions.user_id=users.id
where login_sessions.id=$1`,
				loginSessionID,
			).Scan(&loginSession.ID, &user.ID, &user.Username)
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
	server := ctx.Value(ctxKeyServer).(*Server)
	cookie := &(*server.sessionCookieTemplate)

	var err error
	cookie.Value, err = server.secureCookie.Encode(cookie.Name, loginSessionID)
	if err != nil {
		return err
	}

	http.SetCookie(w, cookie)

	return nil
}

// clearLoginSessionCookie clears the login session cookie in the response.
func clearLoginSessionCookie(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	server := ctx.Value(ctxKeyServer).(*Server)
	cookie := &(*server.sessionCookieTemplate)
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)
}
