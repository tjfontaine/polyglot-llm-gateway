package graph

import (
	"context"
	"encoding/json"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/tenant"
)

// Helper functions for resolvers

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	return &i
}

func toMapAny(m map[string]string) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func jsonRawToMap(raw json.RawMessage) map[string]any {
	if raw == nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func tenantIDFromContext(ctx context.Context) string {
	if tenantVal := ctx.Value("tenant"); tenantVal != nil {
		if t, ok := tenantVal.(*tenant.Tenant); ok && t != nil && t.ID != "" {
			return t.ID
		}
	}
	return "default"
}

func domainShadowToGraphQL(s *domain.ShadowResult) ShadowResult {
	result := ShadowResult{
		ID:                      s.ID,
		InteractionID:           s.InteractionID,
		ProviderName:            s.ProviderName,
		ProviderModel:           strPtr(s.ProviderModel),
		DurationNs:              int64(s.Duration),
		TokensIn:                intPtr(s.TokensIn),
		TokensOut:               intPtr(s.TokensOut),
		HasStructuralDivergence: s.HasStructuralDivergence,
		CreatedAt:               s.CreatedAt.Unix(),
	}

	if s.Request != nil {
		result.Request = &ShadowRequest{
			Canonical:       jsonRawToMap(s.Request.Canonical),
			ProviderRequest: jsonRawToMap(s.Request.ProviderRequest),
		}
	}

	if s.Response != nil {
		result.Response = &ShadowResponse{
			Raw:            jsonRawToMap(s.Response.Raw),
			Canonical:      jsonRawToMap(s.Response.Canonical),
			ClientResponse: jsonRawToMap(s.Response.ClientResponse),
			FinishReason:   strPtr(s.Response.FinishReason),
		}
		if s.Response.Usage != nil {
			result.Response.Usage = &ShadowUsage{
				PromptTokens:     intPtr(s.Response.Usage.PromptTokens),
				CompletionTokens: intPtr(s.Response.Usage.CompletionTokens),
				TotalTokens:      intPtr(s.Response.Usage.TotalTokens),
			}
		}
	}

	if s.Error != nil {
		result.Error = &ShadowError{
			Type:    s.Error.Type,
			Code:    strPtr(s.Error.Code),
			Message: s.Error.Message,
		}
	}

	for _, d := range s.Divergences {
		result.Divergences = append(result.Divergences, Divergence{
			Type:        domainDivergenceTypeToGraphQL(d.Type),
			Path:        d.Path,
			Description: d.Description,
			Primary:     anyToMap(d.Primary),
			Shadow:      anyToMap(d.Shadow),
		})
	}

	return result
}

func domainDivergenceTypeToGraphQL(dt domain.DivergenceType) DivergenceType {
	switch dt {
	case domain.DivergenceMissingField:
		return DivergenceTypeMissingField
	case domain.DivergenceExtraField:
		return DivergenceTypeExtraField
	case domain.DivergenceTypeMismatch:
		return DivergenceTypeTypeMismatch
	case domain.DivergenceArrayLength:
		return DivergenceTypeArrayLength
	case domain.DivergenceNullMismatch:
		return DivergenceTypeNullMismatch
	default:
		return DivergenceTypeMissingField
	}
}

func anyToMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	// Try to marshal/unmarshal for complex types
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		// If it's not an object, wrap it
		return map[string]any{"value": v}
	}
	return result
}
