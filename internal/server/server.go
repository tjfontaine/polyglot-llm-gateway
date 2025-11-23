package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/auth"
)

type Server struct {
	Router *chi.Mux
	Port   int
	logger *slog.Logger
}

func New(port int, logger *slog.Logger, authenticator *auth.Authenticator) *Server {
	r := chi.NewRouter()

	// Apply middleware in order
	r.Use(RequestIDMiddleware)
	r.Use(LoggingMiddleware(logger))

	// Add auth middleware if authenticator is provided
	if authenticator != nil {
		r.Use(AuthMiddleware(authenticator))
	}

	r.Use(TimeoutMiddleware(30 * time.Second))
	r.Use(middleware.Recoverer)

	// Wrap with OpenTelemetry HTTP instrumentation
	r.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "poly-gateway")
	})

	return &Server{
		Router: r,
		Port:   port,
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("starting server", slog.Int("port", s.Port))
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Router)
}
