package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Router *chi.Mux
	Port   int
}

func New(port int) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	return &Server{
		Router: r,
		Port:   port,
	}
}

func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Router)
}
