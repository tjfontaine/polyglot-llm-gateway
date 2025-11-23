package conversation

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
)

// Record stores a canonical request/response pair in the conversation store.
// It best-effort logs on failure without failing the request path.
func Record(ctx context.Context, store storage.ConversationStore, convID string, req *domain.CanonicalRequest, resp *domain.CanonicalResponse, metadata map[string]string) string {
	if store == nil {
		return convID
	}

	logger := slog.Default()

	if convID == "" {
		if resp != nil && resp.ID != "" {
			convID = resp.ID
		} else {
			convID = "conv_" + uuid.New().String()
		}
	}

	meta := make(map[string]string)
	for k, v := range metadata {
		meta[k] = v
	}

	if req != nil {
		meta["model"] = req.Model
		meta["requested_model"] = req.Model
		for k, v := range req.Metadata {
			meta["req."+k] = v
		}
	}

	if resp != nil && resp.Model != "" {
		meta["resp.model"] = resp.Model
		meta["served_model"] = resp.Model
	}

	if reqID, ok := ctx.Value(server.RequestIDKey).(string); ok && reqID != "" {
		meta["request_id"] = reqID
	}

	tenantID := tenantIDFromContext(ctx)

	conv := &storage.Conversation{
		ID:       convID,
		TenantID: tenantID,
		Metadata: meta,
	}

	if err := store.CreateConversation(ctx, conv); err != nil {
		logger.Error("failed to create conversation",
			slog.String("conversation_id", convID),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
	}

	addMessage := func(role, content string) {
		if content == "" {
			return
		}
		if err := store.AddMessage(ctx, convID, &storage.Message{
			ID:      "msg_" + uuid.New().String(),
			Role:    role,
			Content: content,
		}); err != nil {
			logger.Error("failed to store message",
				slog.String("conversation_id", convID),
				slog.String("tenant_id", tenantID),
				slog.String("role", role),
				slog.String("error", err.Error()),
			)
		}
	}

	if req != nil {
		for _, msg := range req.Messages {
			addMessage(msg.Role, msg.Content)
		}
	}

	if resp != nil && len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		addMessage(msg.Role, msg.Content)
	}

	return convID
}

func tenantIDFromContext(ctx context.Context) string {
	if tenantVal := ctx.Value("tenant"); tenantVal != nil {
		if t, ok := tenantVal.(*tenant.Tenant); ok && t != nil && t.ID != "" {
			return t.ID
		}
	}
	return "default"
}
