package httpz

import (
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

// NewHandler returns an http.Handler that serves the web application.
func NewHandler(
	dbsession *db.Session,
	logger *zerolog.Logger,
	csrfKey []byte,
	secureCookies bool,
	cookieAuthenticationKey []byte,
	cookieEncryptionKey []byte,
) (http.Handler, error) {

	router := chi.NewRouter()

	env := &environment{
		dbsession:    dbsession,
		logger:       logger,
		secureCookie: securecookie.New(cookieAuthenticationKey, cookieEncryptionKey),
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

	router.Use(setContextValue(ctxKeyEnvironment, env))

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
		env := ctx.Value(ctxKeyEnvironment).(*environment)
		user, err := db.GetUserByUsername(ctx, env.dbsession, r.FormValue("username"))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// TODO - rerender form
				http.Error(w, "user name not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		dbpool := db.DBPool(env.dbsession)
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
		env := ctx.Value(ctxKeyEnvironment).(*environment)
		now, err := db.GetCurrentTime(ctx, env.dbsession)
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

	return router, nil
}
