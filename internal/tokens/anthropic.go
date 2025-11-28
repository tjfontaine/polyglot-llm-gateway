package tokens

import (
	"context"
	"encoding/json"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// AnthropicCounter uses Anthropic's native count_tokens API.
type AnthropicCounter struct {
	client  *anthropic.Client
	matcher *ModelMatcher
}

// NewAnthropicCounter creates a new Anthropic token counter.
func NewAnthropicCounter(apiKey string, opts ...anthropic.ClientOption) *AnthropicCounter {
	return &AnthropicCounter{
		client: anthropic.NewClient(apiKey, opts...),
		matcher: NewModelMatcher(
			[]string{"claude-"},
			nil,
		),
	}
}

// NewAnthropicCounterWithClient creates an Anthropic counter with an existing client.
func NewAnthropicCounterWithClient(client *anthropic.Client) *AnthropicCounter {
	return &AnthropicCounter{
		client: client,
		matcher: NewModelMatcher(
			[]string{"claude-"},
			nil,
		),
	}
}

// CountTokens counts tokens using Anthropic's API.
func (c *AnthropicCounter) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	// Convert domain request to Anthropic API request
	apiReq := &anthropic.CountTokensRequest{
		Model: req.Model,
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiMsg := anthropic.Message{
			Role: msg.Role,
		}

		// Handle content - check for rich content first
		if msg.RichContent != nil && len(msg.RichContent.Parts) > 0 {
			var parts []anthropic.ContentPart
			for _, part := range msg.RichContent.Parts {
				switch part.Type {
				case domain.ContentTypeText:
					parts = append(parts, anthropic.ContentPart{
						Type: "text",
						Text: part.Text,
					})
				case domain.ContentTypeToolUse:
					parts = append(parts, anthropic.ContentPart{
						Type:  "tool_use",
						ID:    part.ID,
						Name:  part.Name,
						Input: part.Input,
					})
				case domain.ContentTypeToolResult:
					parts = append(parts, anthropic.ContentPart{
						Type:      "tool_result",
						ToolUseID: part.ToolUseID,
						Content:   part.Text,
					})
				}
			}
			apiMsg.Content = parts
		} else {
			// Simple text content
			apiMsg.Content = []anthropic.ContentPart{{Type: "text", Text: msg.Content}}
		}

		apiReq.Messages = append(apiReq.Messages, apiMsg)
	}

	// Add system message
	if req.System != "" {
		apiReq.System = []anthropic.SystemBlock{{Type: "text", Text: req.System}}
	}

	// Convert tools
	for _, tool := range req.Tools {
		apiTool := anthropic.Tool{
			Name:        tool.Name,
			Description: tool.Description,
		}
		if tool.Parameters != nil {
			apiTool.InputSchema = tool.Parameters
		}
		apiReq.Tools = append(apiReq.Tools, apiTool)
	}

	// Call API
	resp, err := c.client.CountTokens(ctx, apiReq, nil)
	if err != nil {
		return nil, err
	}

	return &domain.TokenCountResponse{
		InputTokens: resp.InputTokens,
		Model:       req.Model,
		Estimated:   false, // Anthropic provides exact counts
	}, nil
}

// SupportsModel returns true for Claude models.
func (c *AnthropicCounter) SupportsModel(model string) bool {
	return c.matcher.Matches(model)
}

// CountTokensRaw counts tokens from raw JSON request body.
// This is useful for pass-through scenarios.
func (c *AnthropicCounter) CountTokensRaw(ctx context.Context, body []byte) ([]byte, error) {
	var req anthropic.CountTokensRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	resp, err := c.client.CountTokens(ctx, &req, nil)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}
