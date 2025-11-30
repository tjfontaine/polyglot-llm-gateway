package conversation

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// LogEvent appends an interaction event to the store (best-effort).
func LogEvent(ctx context.Context, store storage.InteractionStore, evt *domain.InteractionEvent) {
	if store == nil || evt == nil {
		return
	}
	if evt.ID == "" {
		evt.ID = "evt_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now()
	}
	_ = store.AppendInteractionEvent(ctx, evt)
}
