package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// MappingProvider rewrites model names and routes requests to specific providers.
// It is a drop-in replacement for the old provider.ModelMappingProvider.
type MappingProvider struct {
	defaultProvider ports.Provider
	providers       map[string]ports.Provider
	prefixMap       map[string]string
	rewrites        []config.ModelRewriteRule
	fallback        *config.ModelRewriteRule
}

// NewMappingProvider creates a provider that can rewrite and remap models before routing.
func NewMappingProvider(defaultProvider ports.Provider, providers map[string]ports.Provider, cfg config.ModelRoutingConfig) (*MappingProvider, error) {
	// Validate that all configured providers exist
	for _, providerName := range cfg.PrefixProviders {
		if _, ok := providers[providerName]; !ok {
			return nil, fmt.Errorf("unknown provider in prefix map: %s", providerName)
		}
	}
	for _, rewrite := range cfg.Rewrites {
		if rewrite.Provider != "" {
			if _, ok := providers[rewrite.Provider]; !ok {
				matchLabel := rewrite.ModelExact
				if matchLabel == "" {
					matchLabel = rewrite.ModelPrefix
				}
				if matchLabel == "" {
					matchLabel = rewrite.Match
				}
				return nil, fmt.Errorf("unknown provider in rewrite for model %s: %s", matchLabel, rewrite.Provider)
			}
		}
	}
	if cfg.Fallback != nil && cfg.Fallback.Provider != "" {
		if _, ok := providers[cfg.Fallback.Provider]; !ok {
			return nil, fmt.Errorf("unknown provider in fallback rewrite: %s", cfg.Fallback.Provider)
		}
	}

	return &MappingProvider{
		defaultProvider: defaultProvider,
		providers:       providers,
		prefixMap:       cfg.PrefixProviders,
		rewrites:        cfg.Rewrites,
		fallback:        cfg.Fallback,
	}, nil
}

func (p *MappingProvider) Name() string {
	return "model-mapper"
}

func (p *MappingProvider) APIType() domain.APIType {
	return p.defaultProvider.APIType()
}

func (p *MappingProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	provider, mappedReq, responseModel, err := p.selectProvider(req)
	if err != nil {
		return nil, err
	}

	resp, err := provider.Complete(ctx, mappedReq)
	if err != nil {
		return nil, err
	}

	resp.ProviderModel = resp.Model
	if responseModel != "" {
		resp.Model = responseModel
	}

	return resp, nil
}

func (p *MappingProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	provider, mappedReq, responseModel, err := p.selectProvider(req)
	if err != nil {
		return nil, err
	}

	upstream, err := provider.Stream(ctx, mappedReq)
	if err != nil {
		return nil, err
	}

	if responseModel == "" {
		return upstream, nil
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)
		for event := range upstream {
			if event.Model != "" {
				event.ProviderModel = event.Model
				event.Model = responseModel
			}
			out <- event
		}
	}()
	return out, nil
}

func (p *MappingProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	list, err := p.defaultProvider.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	// Apply response model rewrite for listings when configured.
	for i, m := range list.Data {
		for _, rewrite := range p.rewrites {
			if !rewrite.RewriteResponseModel {
				continue
			}
			if rewrite.Model != "" && m.ID == rewrite.Model {
				if rewrite.ModelExact != "" {
					list.Data[i].ID = rewrite.ModelExact
				} else if rewrite.ModelPrefix != "" {
					list.Data[i].ID = rewrite.ModelPrefix
				} else if rewrite.Match != "" {
					list.Data[i].ID = rewrite.Match
				}
			}
		}
	}

	return list, nil
}

// CountTokens delegates to the routed provider when supported.
func (p *MappingProvider) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	type countTokensProvider interface {
		CountTokens(ctx context.Context, body []byte) ([]byte, error)
	}

	model := extractModelFromBody(body)
	target, _, _, err := p.selectProvider(&domain.CanonicalRequest{Model: model})
	if err != nil {
		return nil, err
	}

	if ctp, ok := target.(countTokensProvider); ok {
		return ctp.CountTokens(ctx, body)
	}

	return nil, fmt.Errorf("count_tokens not supported by provider for request")
}

func extractModelFromBody(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &req)
	return req.Model
}

// selectProvider applies mapping rules and returns the target provider, the mapped request, and optional response model rewrite.
func (p *MappingProvider) selectProvider(req *domain.CanonicalRequest) (ports.Provider, *domain.CanonicalRequest, string, error) {
	if req == nil {
		return nil, nil, "", fmt.Errorf("request cannot be nil")
	}

	if req.Model == "" {
		return nil, nil, "", fmt.Errorf("model cannot be empty")
	}

	mappedReq := *req

	if providerName, ok := p.prefixMap[p.getProviderName(req.Model)]; ok {
		if provider, ok := p.providers[providerName]; ok {
			mappedReq.Model = p.stripProviderPrefix(req.Model)
			return provider, &mappedReq, "", nil
		}
	}

	for _, rewrite := range p.rewrites {
		if rewrite.ModelExact != "" && req.Model == rewrite.ModelExact {
			return p.applyRewrite(rewrite, &mappedReq)
		}
		if rewrite.ModelPrefix != "" && strings.HasPrefix(req.Model, rewrite.ModelPrefix) {
			return p.applyRewrite(rewrite, &mappedReq)
		}
		if rewrite.Match != "" && strings.Contains(req.Model, rewrite.Match) {
			return p.applyRewrite(rewrite, &mappedReq)
		}
	}

	if p.fallback != nil {
		return p.applyRewrite(*p.fallback, &mappedReq)
	}

	return p.defaultProvider, &mappedReq, "", nil
}

func (p *MappingProvider) applyRewrite(rewrite config.ModelRewriteRule, req *domain.CanonicalRequest) (ports.Provider, *domain.CanonicalRequest, string, error) {
	cloned := *req

	provider := p.defaultProvider
	if rewrite.Provider != "" {
		var ok bool
		provider, ok = p.providers[rewrite.Provider]
		if !ok {
			return nil, nil, "", fmt.Errorf("unknown provider: %s", rewrite.Provider)
		}
	}

	responseModel := ""
	if rewrite.RewriteResponseModel {
		responseModel = req.Model
	}

	if rewrite.Model != "" {
		cloned.Model = rewrite.Model
	}

	return provider, &cloned, responseModel, nil
}

func (p *MappingProvider) getProviderName(model string) string {
	if idx := strings.Index(model, "/"); idx != -1 {
		return model[:idx]
	}
	return ""
}

func (p *MappingProvider) stripProviderPrefix(model string) string {
	if idx := strings.Index(model, "/"); idx != -1 {
		return model[idx+1:]
	}
	return model
}
