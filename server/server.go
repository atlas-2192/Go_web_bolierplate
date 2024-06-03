package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
)

type Server struct {
	handler       http.Handler
	listenAddress string
	server        *http.Server

	dbsession *db.Session
	logger    *zerolog.Logger
}

func NewServer(
	listenAddress string,
	dbsession *db.Session,
	logger *zerolog.Logger,
) (*Server, error) {

	router := chi.NewRouter()

	server := &Server{
		handler:       router,
		listenAddress: listenAddress,
		dbsession:     dbsession,
		logger:        logger,
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

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		server := ctx.Value(ctxKeyServer).(*Server)
		now, err := db.GetCurrentTime(ctx, server.dbsession)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		view.Hello("world", now).Render(r.Context(), w)
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
