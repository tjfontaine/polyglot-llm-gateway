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
	s.router.Get("/api/responses", s.handleListResponses)
	s.router.Get("/api/responses/{response_id}", s.handleResponseDetail)
	s.router.Get("/api/interactions", s.handleListInteractions)
	s.router.Get("/api/interactions/{interaction_id}", s.handleInteractionDetail)

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
	Name            string              `json:"name"`
	Frontdoor       string              `json:"frontdoor"`
	Path            string              `json:"path"`
	Provider        string              `json:"provider,omitempty"`
	DefaultModel    string              `json:"default_model,omitempty"`
	EnableResponses bool                `json:"enable_responses"`
	ModelRouting    ModelRoutingSummary `json:"model_routing,omitempty"`
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
	EnablePassthrough bool   `json:"enable_passthrough"`
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
				Name:            app.Name,
				Frontdoor:       app.Frontdoor,
				Path:            app.Path,
				Provider:        app.Provider,
				DefaultModel:    app.DefaultModel,
				EnableResponses: app.EnableResponses,
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
				EnablePassthrough: p.EnablePassthrough,
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

// ResponseSummary is a summary view of a response for list display
type ResponseSummary struct {
	ID                 string            `json:"id"`
	Status             string            `json:"status"`
	Model              string            `json:"model"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          int64             `json:"created_at"`
	UpdatedAt          int64             `json:"updated_at"`
}

// ResponseListResponse is the response for listing responses
type ResponseListResponse struct {
	Responses []ResponseSummary `json:"responses"`
}

// ResponseDetailView is the full detail view of a response
type ResponseDetailView struct {
	ID                 string            `json:"id"`
	Status             string            `json:"status"`
	Model              string            `json:"model"`
	Request            json.RawMessage   `json:"request,omitempty"`
	Response           json.RawMessage   `json:"response,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          int64             `json:"created_at"`
	UpdatedAt          int64             `json:"updated_at"`
}

func (s *Server) handleListResponses(w http.ResponseWriter, r *http.Request) {
	respStore, ok := s.store.(storage.ResponseStore)
	if !ok || s.store == nil {
		http.Error(w, "response storage not configured", http.StatusServiceUnavailable)
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
	records, err := respStore.ListResponses(r.Context(), storage.ListOptions{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		http.Error(w, "failed to list responses", http.StatusInternalServerError)
		return
	}

	resp := ResponseListResponse{Responses: make([]ResponseSummary, 0, len(records))}
	for _, record := range records {
		updatedAt := record.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = record.CreatedAt
		}
		resp.Responses = append(resp.Responses, ResponseSummary{
			ID:                 record.ID,
			Status:             record.Status,
			Model:              record.Model,
			PreviousResponseID: record.PreviousResponseID,
			Metadata:           record.Metadata,
			CreatedAt:          record.CreatedAt.Unix(),
			UpdatedAt:          updatedAt.Unix(),
		})
	}

	writeJSON(w, resp)
}

func (s *Server) handleResponseDetail(w http.ResponseWriter, r *http.Request) {
	respStore, ok := s.store.(storage.ResponseStore)
	if !ok || s.store == nil {
		http.Error(w, "response storage not configured", http.StatusServiceUnavailable)
		return
	}

	responseID := chi.URLParam(r, "response_id")
	record, err := respStore.GetResponse(r.Context(), responseID)
	if err != nil {
		http.Error(w, "response not found", http.StatusNotFound)
		return
	}

	tenantID := tenantIDFromContext(r.Context())
	if record.TenantID != "" && tenantID != "" && record.TenantID != tenantID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	updatedAt := record.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = record.CreatedAt
	}

	resp := ResponseDetailView{
		ID:                 record.ID,
		Status:             record.Status,
		Model:              record.Model,
		Request:            record.Request,
		Response:           record.Response,
		PreviousResponseID: record.PreviousResponseID,
		Metadata:           record.Metadata,
		CreatedAt:          record.CreatedAt.Unix(),
		UpdatedAt:          updatedAt.Unix(),
	}

	writeJSON(w, resp)
}

// InteractionSummary is a unified view of either a conversation or response
type InteractionSummary struct {
	ID                 string            `json:"id"`
	Type               string            `json:"type"` // "conversation" or "response"
	Status             string            `json:"status,omitempty"`
	Model              string            `json:"model,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	MessageCount       int               `json:"message_count,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	CreatedAt          int64             `json:"created_at"`
	UpdatedAt          int64             `json:"updated_at"`
}

// InteractionListResponse is the response for listing all interactions
type InteractionListResponse struct {
	Interactions []InteractionSummary `json:"interactions"`
	Total        int                  `json:"total"`
}

// InteractionDetailView is the full detail view of an interaction
type InteractionDetailView struct {
	ID                 string            `json:"id"`
	Type               string            `json:"type"`
	Status             string            `json:"status,omitempty"`
	Model              string            `json:"model,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	CreatedAt          int64             `json:"created_at"`
	UpdatedAt          int64             `json:"updated_at"`
	// For conversations
	Messages []MessageView `json:"messages,omitempty"`
	// For responses
	Request  json.RawMessage `json:"request,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

func (s *Server) handleListInteractions(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	limit := 50
	offset := 0
	filterType := r.URL.Query().Get("type") // "conversation", "response", or "" for all

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
	opts := storage.ListOptions{
		TenantID: tenantID,
		Limit:    limit * 2, // Get more to allow merging and sorting
		Offset:   0,
	}

	var interactions []InteractionSummary

	// Get conversations if not filtered to responses only
	if filterType == "" || filterType == "conversation" {
		conversations, err := s.store.ListConversations(r.Context(), opts)
		if err == nil {
			for _, conv := range conversations {
				updatedAt := conv.UpdatedAt
				if updatedAt.IsZero() {
					updatedAt = conv.CreatedAt
				}
				interactions = append(interactions, InteractionSummary{
					ID:           conv.ID,
					Type:         "conversation",
					Metadata:     conv.Metadata,
					MessageCount: len(conv.Messages),
					Model:        conv.Metadata["served_model"],
					CreatedAt:    conv.CreatedAt.Unix(),
					UpdatedAt:    updatedAt.Unix(),
				})
			}
		}
	}

	// Get responses if not filtered to conversations only
	if filterType == "" || filterType == "response" {
		if respStore, ok := s.store.(storage.ResponseStore); ok {
			records, err := respStore.ListResponses(r.Context(), opts)
			if err == nil {
				for _, record := range records {
					updatedAt := record.UpdatedAt
					if updatedAt.IsZero() {
						updatedAt = record.CreatedAt
					}
					interactions = append(interactions, InteractionSummary{
						ID:                 record.ID,
						Type:               "response",
						Status:             record.Status,
						Model:              record.Model,
						Metadata:           record.Metadata,
						PreviousResponseID: record.PreviousResponseID,
						CreatedAt:          record.CreatedAt.Unix(),
						UpdatedAt:          updatedAt.Unix(),
					})
				}
			}
		}
	}

	// Sort by updated_at descending
	for i := 0; i < len(interactions); i++ {
		for j := i + 1; j < len(interactions); j++ {
			if interactions[j].UpdatedAt > interactions[i].UpdatedAt {
				interactions[i], interactions[j] = interactions[j], interactions[i]
			}
		}
	}

	// Apply pagination
	total := len(interactions)
	if offset >= len(interactions) {
		interactions = []InteractionSummary{}
	} else {
		end := offset + limit
		if end > len(interactions) {
			end = len(interactions)
		}
		interactions = interactions[offset:end]
	}

	resp := InteractionListResponse{
		Interactions: interactions,
		Total:        total,
	}

	writeJSON(w, resp)
}

func (s *Server) handleInteractionDetail(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	interactionID := chi.URLParam(r, "interaction_id")
	tenantID := tenantIDFromContext(r.Context())

	// Try to get as conversation first
	conv, err := s.store.GetConversation(r.Context(), interactionID)
	if err == nil {
		if conv.TenantID != "" && tenantID != "" && conv.TenantID != tenantID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		updatedAt := conv.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = conv.CreatedAt
		}

		resp := InteractionDetailView{
			ID:        conv.ID,
			Type:      "conversation",
			Metadata:  conv.Metadata,
			Model:     conv.Metadata["served_model"],
			CreatedAt: conv.CreatedAt.Unix(),
			UpdatedAt: updatedAt.Unix(),
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
		return
	}

	// Try to get as response
	if respStore, ok := s.store.(storage.ResponseStore); ok {
		record, err := respStore.GetResponse(r.Context(), interactionID)
		if err == nil {
			if record.TenantID != "" && tenantID != "" && record.TenantID != tenantID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			updatedAt := record.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = record.CreatedAt
			}

			resp := InteractionDetailView{
				ID:                 record.ID,
				Type:               "response",
				Status:             record.Status,
				Model:              record.Model,
				Metadata:           record.Metadata,
				PreviousResponseID: record.PreviousResponseID,
				CreatedAt:          record.CreatedAt.Unix(),
				UpdatedAt:          updatedAt.Unix(),
				Request:            record.Request,
				Response:           record.Response,
			}

			writeJSON(w, resp)
			return
		}
	}

	http.Error(w, "interaction not found", http.StatusNotFound)
}
