package shadow

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

// DetectDivergences compares a primary response with a shadow result
// and returns a list of structural divergences found.
func DetectDivergences(primary *domain.CanonicalResponse, shadow *domain.ShadowResult) []domain.Divergence {
	if primary == nil || shadow == nil || shadow.Response == nil {
		return nil
	}

	// Parse the shadow's canonical response
	var shadowResp domain.CanonicalResponse
	if err := json.Unmarshal(shadow.Response.Canonical, &shadowResp); err != nil {
		return []domain.Divergence{{
			Type:        domain.DivergenceTypeMismatch,
			Path:        "response",
			Description: fmt.Sprintf("failed to parse shadow response: %v", err),
		}}
	}

	var divergences []domain.Divergence

	// Compare choices count
	if len(primary.Choices) != len(shadowResp.Choices) {
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceArrayLength,
			Path:        "choices",
			Primary:     fmt.Sprintf("%d", len(primary.Choices)),
			Shadow:      fmt.Sprintf("%d", len(shadowResp.Choices)),
			Description: fmt.Sprintf("choices count differs: primary=%d, shadow=%d", len(primary.Choices), len(shadowResp.Choices)),
		})
	}

	// Compare each choice
	minChoices := min(len(primary.Choices), len(shadowResp.Choices))
	for i := 0; i < minChoices; i++ {
		choiceDivergences := compareChoices(fmt.Sprintf("choices[%d]", i), &primary.Choices[i], &shadowResp.Choices[i])
		divergences = append(divergences, choiceDivergences...)
	}

	// Compare finish reasons (if any choices exist)
	if len(primary.Choices) > 0 && len(shadowResp.Choices) > 0 {
		if primary.Choices[0].FinishReason != shadowResp.Choices[0].FinishReason {
			divergences = append(divergences, domain.Divergence{
				Type:        domain.DivergenceTypeMismatch,
				Path:        "choices[0].finish_reason",
				Primary:     primary.Choices[0].FinishReason,
				Shadow:      shadowResp.Choices[0].FinishReason,
				Description: fmt.Sprintf("finish reason differs: primary=%s, shadow=%s", primary.Choices[0].FinishReason, shadowResp.Choices[0].FinishReason),
			})
		}
	}

	// Compare tool calls presence and structure
	toolCallDivergences := compareToolCalls(primary, &shadowResp)
	divergences = append(divergences, toolCallDivergences...)

	return divergences
}

// compareChoices compares two Choice objects for structural divergences
func compareChoices(path string, primary, shadow *domain.Choice) []domain.Divergence {
	var divergences []domain.Divergence

	// Compare message role
	if primary.Message.Role != shadow.Message.Role {
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceTypeMismatch,
			Path:        path + ".message.role",
			Primary:     primary.Message.Role,
			Shadow:      shadow.Message.Role,
			Description: fmt.Sprintf("message role differs: primary=%s, shadow=%s", primary.Message.Role, shadow.Message.Role),
		})
	}

	// Compare content presence (not content itself - that's expected to differ)
	primaryHasContent := hasMessageContent(&primary.Message)
	shadowHasContent := hasMessageContent(&shadow.Message)
	if primaryHasContent != shadowHasContent {
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceNullMismatch,
			Path:        path + ".message.content",
			Primary:     fmt.Sprintf("has_content=%v", primaryHasContent),
			Shadow:      fmt.Sprintf("has_content=%v", shadowHasContent),
			Description: "content presence differs between primary and shadow",
		})
	}

	// Compare content structure if both have rich content
	if primary.Message.RichContent != nil && shadow.Message.RichContent != nil {
		contentDivergences := compareMessageContent(path+".message.content", primary.Message.RichContent, shadow.Message.RichContent)
		divergences = append(divergences, contentDivergences...)
	} else if (primary.Message.RichContent != nil) != (shadow.Message.RichContent != nil) {
		// One has rich content, the other doesn't
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceTypeMismatch,
			Path:        path + ".message.content_type",
			Primary:     fmt.Sprintf("rich_content=%v", primary.Message.RichContent != nil),
			Shadow:      fmt.Sprintf("rich_content=%v", shadow.Message.RichContent != nil),
			Description: "content structure differs (simple vs rich content)",
		})
	}

	return divergences
}

// compareMessageContent compares the structure of message content
func compareMessageContent(path string, primary, shadow *domain.MessageContent) []domain.Divergence {
	var divergences []domain.Divergence

	// Compare text vs parts content
	if (len(primary.Parts) > 0) != (len(shadow.Parts) > 0) {
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceTypeMismatch,
			Path:        path,
			Primary:     fmt.Sprintf("parts_count=%d", len(primary.Parts)),
			Shadow:      fmt.Sprintf("parts_count=%d", len(shadow.Parts)),
			Description: "content structure differs (text vs multipart content)",
		})
		return divergences
	}

	// Compare parts structure if both have it
	if len(primary.Parts) > 0 {
		if len(primary.Parts) != len(shadow.Parts) {
			divergences = append(divergences, domain.Divergence{
				Type:        domain.DivergenceArrayLength,
				Path:        path,
				Primary:     fmt.Sprintf("%d", len(primary.Parts)),
				Shadow:      fmt.Sprintf("%d", len(shadow.Parts)),
				Description: fmt.Sprintf("parts count differs: primary=%d, shadow=%d", len(primary.Parts), len(shadow.Parts)),
			})
		}

		// Compare types of each content block
		minBlocks := min(len(primary.Parts), len(shadow.Parts))
		for i := 0; i < minBlocks; i++ {
			if primary.Parts[i].Type != shadow.Parts[i].Type {
				divergences = append(divergences, domain.Divergence{
					Type:        domain.DivergenceTypeMismatch,
					Path:        fmt.Sprintf("%s[%d].type", path, i),
					Primary:     string(primary.Parts[i].Type),
					Shadow:      string(shadow.Parts[i].Type),
					Description: fmt.Sprintf("content block type differs at index %d: primary=%s, shadow=%s", i, primary.Parts[i].Type, shadow.Parts[i].Type),
				})
			}
		}
	}

	return divergences
}

// compareToolCalls compares tool call structures between responses
func compareToolCalls(primary, shadow *domain.CanonicalResponse) []domain.Divergence {
	var divergences []domain.Divergence

	primaryToolCalls := extractToolCalls(primary)
	shadowToolCalls := extractToolCalls(shadow)

	// Compare presence
	if len(primaryToolCalls) == 0 && len(shadowToolCalls) == 0 {
		return nil
	}

	if len(primaryToolCalls) != len(shadowToolCalls) {
		divergences = append(divergences, domain.Divergence{
			Type:        domain.DivergenceArrayLength,
			Path:        "tool_calls",
			Primary:     fmt.Sprintf("%d", len(primaryToolCalls)),
			Shadow:      fmt.Sprintf("%d", len(shadowToolCalls)),
			Description: fmt.Sprintf("tool call count differs: primary=%d, shadow=%d", len(primaryToolCalls), len(shadowToolCalls)),
		})
	}

	// Compare tool call structure (names, not arguments which can vary)
	minCalls := min(len(primaryToolCalls), len(shadowToolCalls))
	for i := 0; i < minCalls; i++ {
		if primaryToolCalls[i].Name != shadowToolCalls[i].Name {
			divergences = append(divergences, domain.Divergence{
				Type:        domain.DivergenceTypeMismatch,
				Path:        fmt.Sprintf("tool_calls[%d].name", i),
				Primary:     primaryToolCalls[i].Name,
				Shadow:      shadowToolCalls[i].Name,
				Description: fmt.Sprintf("tool call name differs at index %d: primary=%s, shadow=%s", i, primaryToolCalls[i].Name, shadowToolCalls[i].Name),
			})
		}

		// Compare argument structure (keys present, not values)
		argDivergences := compareToolCallArguments(fmt.Sprintf("tool_calls[%d].arguments", i), primaryToolCalls[i].Arguments, shadowToolCalls[i].Arguments)
		divergences = append(divergences, argDivergences...)
	}

	return divergences
}

// toolCall represents a simplified tool call for comparison
type toolCall struct {
	Name      string
	Arguments string
}

// extractToolCalls extracts tool calls from a response
func extractToolCalls(resp *domain.CanonicalResponse) []toolCall {
	if resp == nil || len(resp.Choices) == 0 {
		return nil
	}

	var calls []toolCall
	for _, choice := range resp.Choices {
		for _, tc := range choice.Message.ToolCalls {
			calls = append(calls, toolCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}
	return calls
}

// compareToolCallArguments compares the structure of tool call arguments
func compareToolCallArguments(path string, primary, shadow string) []domain.Divergence {
	var divergences []domain.Divergence

	var primaryArgs, shadowArgs map[string]any
	if err := json.Unmarshal([]byte(primary), &primaryArgs); err != nil {
		// Not JSON, can't compare structure
		return nil
	}
	if err := json.Unmarshal([]byte(shadow), &shadowArgs); err != nil {
		// Not JSON, can't compare structure
		return nil
	}

	// Compare keys present
	primaryKeys := getMapKeys(primaryArgs)
	shadowKeys := getMapKeys(shadowArgs)

	for key := range primaryKeys {
		if !shadowKeys[key] {
			divergences = append(divergences, domain.Divergence{
				Type:        domain.DivergenceMissingField,
				Path:        path + "." + key,
				Primary:     "present",
				Shadow:      "missing",
				Description: fmt.Sprintf("argument key '%s' missing in shadow response", key),
			})
		}
	}

	for key := range shadowKeys {
		if !primaryKeys[key] {
			divergences = append(divergences, domain.Divergence{
				Type:        domain.DivergenceExtraField,
				Path:        path + "." + key,
				Primary:     "missing",
				Shadow:      "present",
				Description: fmt.Sprintf("argument key '%s' extra in shadow response", key),
			})
		}
	}

	// Compare value types for common keys
	for key := range primaryKeys {
		if shadowKeys[key] {
			primaryType := getJSONType(primaryArgs[key])
			shadowType := getJSONType(shadowArgs[key])
			if primaryType != shadowType {
				divergences = append(divergences, domain.Divergence{
					Type:        domain.DivergenceTypeMismatch,
					Path:        path + "." + key,
					Primary:     primaryType,
					Shadow:      shadowType,
					Description: fmt.Sprintf("argument '%s' type differs: primary=%s, shadow=%s", key, primaryType, shadowType),
				})
			}
		}
	}

	return divergences
}

// hasMessageContent checks if a Message has any content
func hasMessageContent(m *domain.Message) bool {
	if m.Content != "" {
		return true
	}
	if m.RichContent != nil && (m.RichContent.Text != "" || len(m.RichContent.Parts) > 0) {
		return true
	}
	return false
}

// getMapKeys returns a set of keys from a map
func getMapKeys(m map[string]any) map[string]bool {
	keys := make(map[string]bool)
	for k := range m {
		keys[k] = true
	}
	return keys
}

// getJSONType returns a string representation of a JSON value's type
func getJSONType(v any) string {
	if v == nil {
		return "null"
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "unknown"
	}
}
