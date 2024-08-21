package httpz

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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxutil"
	"github.com/jackc/web-starter-app/db"
	"github.com/jackc/web-starter-app/lib/bee"
	"github.com/jackc/web-starter-app/lib/formdata"
	"github.com/jackc/web-starter-app/view"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// NewHandler returns an http.Handler that serves the web application.
func NewHandler(
	dbpool *pgxpool.Pool,
	logger *zerolog.Logger,
	csrfKey []byte,
	secureCookies bool,
	cookieAuthenticationKey []byte,
	cookieEncryptionKey []byte,
) (http.Handler, error) {

	router := chi.NewRouter()

	env := &environment{
		dbpool:       dbpool,
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

	hb := bee.HandlerBuilder[*environment]{
		CtxKeyEnv: ctxKeyEnvironment,
		ErrorHandlers: []bee.ErrorHandler{
			func(w http.ResponseWriter, r *http.Request, err error) (bool, error) {
				zerolog.Ctx(r.Context()).Error().Err(err).Msg("error handling request")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return true, nil
			},
		},
	}

	router.Method("GET", "/login", hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		return view.LoginPage(csrf.Token(r)).Render(ctx, w)
	}))

	router.Method("POST", "/login/submit", hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
		user, err := db.GetUserByUsername(ctx, env.dbpool, r.FormValue("username"))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// TODO - rerender form
				http.Error(w, "user name not found", http.StatusNotFound)
				return nil
			}
			return err
		}

		now := time.Now()
		loginSessionID, err := pgxutil.InsertRowReturning(ctx, env.dbpool, "login_sessions", map[string]any{
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
			return err
		}

		setLoginSessionCookie(w, r, loginSessionID)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	}))

	router.Method("POST", "/logout", hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
		loginSession := getLoginSession(ctx)
		if loginSession != nil {
			_, err := env.dbpool.Exec(ctx, "delete from login_sessions where id=$1", loginSession.ID)
			if err != nil {
				return err
			}
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}))

	router.Method("GET", "/", hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
		now, err := db.GetCurrentTime(ctx, env.dbpool)
		if err != nil {
			return err
		}

		var name string
		loginSession := getLoginSession(ctx)
		if loginSession.User != nil {
			name = loginSession.User.Username
		} else {
			name = "world"
		}

		return view.ApplicationLayout(view.Hello(csrf.Token(r), name, now)).Render(r.Context(), w)
	}))

	newWalkForm := &formdata.Form{
		Fields: []*formdata.Field{
			{
				Label:    "Duration",
				Name:     "duration",
				Type:     "duration",
				Required: true,
			},
			{
				Label:    "Distance in miles",
				Name:     "distance_in_miles",
				Type:     "number",
				Required: true,
			},
		},
	}

	router.Method("GET", "/walks/new", hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
		loginSession := getLoginSession(ctx)
		if loginSession.User == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}

		formData := newWalkForm.New()

		return view.ApplicationLayout(view.WalksNew(formData)).Render(r.Context(), w)
	}))

	router.Method("POST", "/walks", func() http.Handler {
		return hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, env *environment, params map[string]any) error {
			loginSession := getLoginSession(ctx)
			if loginSession.User == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}

			formData := newWalkForm.Parse(params)

			_, err := env.dbpool.Exec(
				ctx,
				"insert into walks (id, user_id, duration, distance_in_miles) values ($1, $2, $3, $4)",
				uuid.Must(uuid.NewV7()),
				loginSession.User.ID,
				formData.FieldValues["duration"].Value,
				formData.FieldValues["distance_in_miles"].Value,
			)
			if err != nil {
				return err
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return nil
		})
	}())

	return router, nil
}
