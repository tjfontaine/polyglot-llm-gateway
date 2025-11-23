package controlplane

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed dist/*
var distFS embed.FS

type Server struct {
	router    *chi.Mux
	startTime time.Time
	assets    fs.FS
}

func NewServer() *Server {
	// Sub-filesystem for the 'dist' directory
	assets, _ := fs.Sub(distFS, "dist")

	s := &Server{
		router:    chi.NewRouter(),
		startTime: time.Now(),
		assets:    assets,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/api/stats", s.handleStats)

	// Serve static files
	fileServer := http.FileServer(http.FS(s.assets))
	s.router.Handle("/*", http.StripPrefix("/admin", fileServer))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

type StatsResponse struct {
	Uptime       string      `json:"uptime"`
	GoVersion    string      `json:"go_version"`
	NumGoroutine int         `json:"num_goroutine"`
	Memory       MemoryStats `json:"memory"`
}

type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := StatsResponse{
		Uptime:       time.Since(s.startTime).String(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		Memory: MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
