package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
)

type Provider struct {
	client openai.Client
}

// New creates a new OpenAI provider
func New(apiKey string, opts ...option.RequestOption) *Provider {
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	clientOpts = append(clientOpts, opts...)

	return &Provider{
		client: openai.NewClient(clientOpts...),
	}
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, m := range req.Messages {
		switch m.Role {
		case "user":
			messages[i] = openai.UserMessage(m.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(m.Content)
		case "system":
			messages[i] = openai.SystemMessage(m.Content)
		default:
			return nil, fmt.Errorf("unsupported role: %s", m.Role)
		}
	}

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModel(req.Model),
	}

	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}

	// TODO: Handle other parameters like Temperature, Tools, etc.

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	choices := make([]domain.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = domain.Choice{
			Index: int(c.Index),
			Message: domain.Message{
				Role:    string(c.Message.Role),
				Content: c.Message.Content,
			},
			FinishReason: string(c.FinishReason),
		}
	}

	return &domain.CanonicalResponse{
		ID:      resp.ID,
		Object:  string(resp.Object),
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: domain.Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, m := range req.Messages {
		switch m.Role {
		case "user":
			messages[i] = openai.UserMessage(m.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(m.Content)
		case "system":
			messages[i] = openai.SystemMessage(m.Content)
		default:
			return nil, fmt.Errorf("unsupported role: %s", m.Role)
		}
	}

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModel(req.Model),
	}

	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	out := make(chan domain.CanonicalEvent)

	go func() {
		defer close(out)
		defer stream.Close()

		for stream.Next() {
			chunk := stream.Current()

			// If usage is present (last chunk with stream_options), we can send it.
			// For now, we focus on content deltas.

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				delta := choice.Delta

				event := domain.CanonicalEvent{
					Role:         delta.Role,
					ContentDelta: delta.Content,
				}

				// TODO: Handle ToolCalls

				out <- event
			}
		}

		if err := stream.Err(); err != nil {
			out <- domain.CanonicalEvent{Error: err}
		}
	}()

	return out, nil
}
