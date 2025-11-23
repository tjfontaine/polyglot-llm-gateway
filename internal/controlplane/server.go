package controlplane

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
)

//go:embed dist/*
var distFS embed.FS

type Server struct {
	router    *chi.Mux
	startTime time.Time
	assets    fs.FS
	cfg       *config.Config
	store     storage.ConversationStore
	tenants   []*tenant.Tenant
}

func NewServer(cfg *config.Config, store storage.ConversationStore, tenants []*tenant.Tenant) *Server {
	assets, _ := fs.Sub(distFS, "dist")

	s := &Server{
		router:    chi.NewRouter(),
		startTime: time.Now(),
		assets:    assets,
		cfg:       cfg,
		store:     store,
		tenants:   tenants,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/api/stats", s.handleStats)
	s.router.Get("/api/overview", s.handleOverview)
	s.router.Get("/api/threads", s.handleListThreads)
	s.router.Get("/api/threads/{thread_id}", s.handleThreadDetail)

	s.router.Get("/*", s.handleApp)
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

	writeJSON(w, stats)
}

type OverviewResponse struct {
	Mode       string             `json:"mode"`
	Storage    StorageSummary     `json:"storage"`
	Apps       []AppSummary       `json:"apps"`
	Frontdoors []FrontdoorSummary `json:"frontdoors"`
	Providers  []ProviderSummary  `json:"providers"`
	Routing    RoutingSummary     `json:"routing"`
	Tenants    []TenantSummary    `json:"tenants"`
}

type StorageSummary struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
}

type FrontdoorSummary struct {
	Type         string `json:"type"`
	Path         string `json:"path"`
	Provider     string `json:"provider,omitempty"`
	DefaultModel string `json:"default_model,omitempty"`
}

type AppSummary struct {
	Name         string              `json:"name"`
	Frontdoor    string              `json:"frontdoor"`
	Path         string              `json:"path"`
	Provider     string              `json:"provider,omitempty"`
	DefaultModel string              `json:"default_model,omitempty"`
	ModelRouting ModelRoutingSummary `json:"model_routing,omitempty"`
}

type ModelRoutingSummary struct {
	PrefixProviders map[string]string     `json:"prefix_providers,omitempty"`
	Rewrites        []ModelRewriteSummary `json:"rewrites,omitempty"`
}

type ModelRewriteSummary struct {
	ModelExact  string `json:"model_exact,omitempty"`
	ModelPrefix string `json:"model_prefix,omitempty"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
}

type ProviderSummary struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	BaseURL           string `json:"base_url,omitempty"`
	SupportsResponses bool   `json:"supports_responses"`
}

type RoutingSummary struct {
	DefaultProvider string            `json:"default_provider"`
	Rules           []RoutingRuleView `json:"rules"`
}

type RoutingRuleView struct {
	ModelPrefix string `json:"model_prefix,omitempty"`
	ModelExact  string `json:"model_exact,omitempty"`
	Provider    string `json:"provider"`
}

type TenantSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ProviderCount  int    `json:"provider_count"`
	RoutingRules   int    `json:"routing_rules"`
	SupportsTenant bool   `json:"supports_tenant"`
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	resp := OverviewResponse{
		Mode: "single-tenant",
	}

	if s.cfg != nil {
		resp.Storage = StorageSummary{
			Enabled: s.store != nil && s.cfg.Storage.Type != "" && s.cfg.Storage.Type != "none",
			Type:    s.cfg.Storage.Type,
			Path:    s.cfg.Storage.SQLite.Path,
		}

		for _, app := range s.cfg.Apps {
			summary := AppSummary{
				Name:         app.Name,
				Frontdoor:    app.Frontdoor,
				Path:         app.Path,
				Provider:     app.Provider,
				DefaultModel: app.DefaultModel,
			}

			if len(app.ModelRouting.PrefixProviders) > 0 || len(app.ModelRouting.Rewrites) > 0 {
				summary.ModelRouting = ModelRoutingSummary{
					PrefixProviders: app.ModelRouting.PrefixProviders,
				}
				for _, rewrite := range app.ModelRouting.Rewrites {
					matchValue := rewrite.ModelExact
					if matchValue == "" {
						matchValue = rewrite.Match
					}

					summary.ModelRouting.Rewrites = append(summary.ModelRouting.Rewrites, ModelRewriteSummary{
						ModelExact:  matchValue,
						ModelPrefix: rewrite.ModelPrefix,
						Provider:    rewrite.Provider,
						Model:       rewrite.Model,
					})
				}
			}

			resp.Apps = append(resp.Apps, summary)
		}

		if len(resp.Apps) == 0 && len(s.cfg.Frontdoors) > 0 {
			for _, fd := range s.cfg.Frontdoors {
				resp.Apps = append(resp.Apps, AppSummary{
					Name:         fd.Type,
					Frontdoor:    fd.Type,
					Path:         fd.Path,
					Provider:     fd.Provider,
					DefaultModel: fd.DefaultModel,
				})
			}
		}
		summary := RoutingSummary{
			DefaultProvider: s.cfg.Routing.DefaultProvider,
		}
		for _, rule := range s.cfg.Routing.Rules {
			summary.Rules = append(summary.Rules, RoutingRuleView{
				ModelPrefix: rule.ModelPrefix,
				ModelExact:  rule.ModelExact,
				Provider:    rule.Provider,
			})
		}
		resp.Routing = summary

		if len(s.cfg.Tenants) > 0 {
			resp.Mode = "multi-tenant"
		}

		for _, fd := range s.cfg.Frontdoors {
			resp.Frontdoors = append(resp.Frontdoors, FrontdoorSummary{
				Type:         fd.Type,
				Path:         fd.Path,
				Provider:     fd.Provider,
				DefaultModel: fd.DefaultModel,
			})
		}

		for _, p := range s.cfg.Providers {
			resp.Providers = append(resp.Providers, ProviderSummary{
				Name:              p.Name,
				Type:              p.Type,
				BaseURL:           p.BaseURL,
				SupportsResponses: p.SupportsResponses,
			})
		}
	}

	for _, t := range s.tenants {
		resp.Tenants = append(resp.Tenants, TenantSummary{
			ID:             t.ID,
			Name:           t.Name,
			ProviderCount:  len(t.Providers),
			RoutingRules:   len(t.Routing.Rules),
			SupportsTenant: true,
		})
	}

	writeJSON(w, resp)
}

type ThreadSummary struct {
	ID           string            `json:"id"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	MessageCount int               `json:"message_count"`
}

type ThreadListResponse struct {
	Threads []ThreadSummary `json:"threads"`
}

func (s *Server) handleListThreads(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "conversation storage not configured", http.StatusServiceUnavailable)
		return
	}

	limit := 50
	offset := 0

	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	if q := r.URL.Query().Get("offset"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 0 {
			offset = v
		}
	}

	tenantID := tenantIDFromContext(r.Context())
	conversations, err := s.store.ListConversations(r.Context(), storage.ListOptions{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		http.Error(w, "failed to list conversations", http.StatusInternalServerError)
		return
	}

	resp := ThreadListResponse{Threads: make([]ThreadSummary, 0, len(conversations))}
	for _, conv := range conversations {
		updatedAt := conv.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = conv.CreatedAt
		}
		resp.Threads = append(resp.Threads, ThreadSummary{
			ID:           conv.ID,
			CreatedAt:    conv.CreatedAt.Unix(),
			UpdatedAt:    updatedAt.Unix(),
			Metadata:     conv.Metadata,
			MessageCount: len(conv.Messages),
		})
	}

	writeJSON(w, resp)
}

type ThreadDetail struct {
	ID        string            `json:"id"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Messages  []MessageView     `json:"messages"`
}

type MessageView struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Server) handleThreadDetail(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "conversation storage not configured", http.StatusServiceUnavailable)
		return
	}

	threadID := chi.URLParam(r, "thread_id")
	conv, err := s.store.GetConversation(r.Context(), threadID)
	if err != nil {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	tenantID := tenantIDFromContext(r.Context())
	if conv.TenantID != "" && tenantID != "" && conv.TenantID != tenantID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	updatedAt := conv.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = conv.CreatedAt
	}

	resp := ThreadDetail{
		ID:        conv.ID,
		CreatedAt: conv.CreatedAt.Unix(),
		UpdatedAt: updatedAt.Unix(),
		Metadata:  conv.Metadata,
		Messages:  make([]MessageView, 0, len(conv.Messages)),
	}

	for _, msg := range conv.Messages {
		resp.Messages = append(resp.Messages, MessageView{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Unix(),
		})
	}

	writeJSON(w, resp)
}

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	path = strings.TrimPrefix(path, "admin/")

	if path == "" || strings.HasSuffix(r.URL.Path, "/") {
		path = "index.html"
	}

	if !strings.Contains(path, "..") && s.serveAsset(w, r, path) {
		return
	}

	if s.serveAsset(w, r, "index.html") {
		return
	}

	http.NotFound(w, r)
}

func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, path string) bool {
	f, err := s.assets.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false
	}

	data, err := fs.ReadFile(s.assets, path)
	if err != nil {
		return false
	}

	http.ServeContent(w, r, path, info.ModTime(), bytes.NewReader(data))
	return true
}

func tenantIDFromContext(ctx context.Context) string {
	if tenantVal := ctx.Value("tenant"); tenantVal != nil {
		if t, ok := tenantVal.(*tenant.Tenant); ok && t != nil && t.ID != "" {
			return t.ID
		}
	}
	return "default"
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
