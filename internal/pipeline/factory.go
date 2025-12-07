package pipeline

import (
	"fmt"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// NewExecutorFromConfig creates a pipeline executor from app configuration.
// Returns nil if no stages are configured.
func NewExecutorFromConfig(cfg config.PipelineConfig) (*Executor, error) {
	if len(cfg.Stages) == 0 {
		return nil, nil
	}

	stages := make([]StageConfig, 0, len(cfg.Stages))

	for _, stageCfg := range cfg.Stages {
		stage, err := newStageFromConfig(stageCfg)
		if err != nil {
			return nil, fmt.Errorf("stage %s: %w", stageCfg.Name, err)
		}

		stageType := ports.StageType(stageCfg.Type)
		if stageType != ports.StagePre && stageType != ports.StagePost {
			return nil, fmt.Errorf("stage %s: invalid type %q (must be 'pre' or 'post')", stageCfg.Name, stageCfg.Type)
		}

		stages = append(stages, StageConfig{
			Name:  stageCfg.Name,
			Type:  stageType,
			Order: stageCfg.Order,
			Stage: stage,
		})
	}

	return NewExecutor(ExecutorConfig{Stages: stages}), nil
}

func newStageFromConfig(cfg config.PipelineStageConfig) (ports.Stage, error) {
	// Parse timeout
	timeout := 5 * time.Second // Default
	if cfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", cfg.Timeout, err)
		}
	}

	// Parse onError action
	var onError ports.StageAction
	switch cfg.OnError {
	case "", "deny":
		onError = ports.ActionDeny
	case "allow":
		onError = ports.ActionAllow
	default:
		return nil, fmt.Errorf("invalid on_error %q (must be 'allow' or 'deny')", cfg.OnError)
	}

	return NewWebhookStage(WebhookStageConfig{
		Name:    cfg.Name,
		Type:    ports.StageType(cfg.Type),
		URL:     cfg.URL,
		Timeout: timeout,
		OnError: onError,
		Retries: cfg.Retries,
		Squelch: cfg.Squelch,
		Headers: cfg.Headers,
	}), nil
}
