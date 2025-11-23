package anthropic

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type Provider struct {
	client anthropic.Client
}

// New creates a new Anthropic provider
func New(apiKey string, opts ...option.RequestOption) *Provider {
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	clientOpts = append(clientOpts, opts...)

	return &Provider{
		client: anthropic.NewClient(clientOpts...),
	}
}

func (p *Provider) Name() string {
	return "anthropic"
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	var systemPrompt []anthropic.TextBlockParam
	var messages []anthropic.MessageParam

	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		case "system":
			systemPrompt = append(systemPrompt, anthropic.TextBlockParam{
				Text: m.Content,
			})
		default:
			return nil, fmt.Errorf("unsupported role: %s", m.Role)
		}
	}

	params := anthropic.MessageNewParams{
		Messages:  messages,
		Model:     anthropic.Model(req.Model),
		MaxTokens: int64(req.MaxTokens),
	}

	if len(systemPrompt) > 0 {
		params.System = systemPrompt
	}

	if req.MaxTokens == 0 {
		params.MaxTokens = 1024 // Default max tokens
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &domain.CanonicalResponse{
		ID:      resp.ID,
		Object:  "chat.completion", // Map to OpenAI-compatible object
		Created: 0,                 // Anthropic doesn't return created timestamp in the same way?
		Model:   string(resp.Model),
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    string(resp.Role),
					Content: content,
				},
				FinishReason: string(resp.StopReason),
			},
		},
		Usage: domain.Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	pager := p.client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})

	var models []domain.Model
	for pager.Next() {
		model := pager.Current()
		models = append(models, domain.Model{
			ID:      model.ID,
			Object:  string(model.Type),
			Created: model.CreatedAt.Unix(),
		})
	}

	if err := pager.Err(); err != nil {
		return nil, err
	}

	return &domain.ModelList{Object: "list", Data: models}, nil
}
