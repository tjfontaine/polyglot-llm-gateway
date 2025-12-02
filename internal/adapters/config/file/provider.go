// Package file provides file-based configuration with hot-reload.
package file

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Provider implements ports.ConfigProvider using file-based configuration.
// It watches the config file for changes and triggers reload callbacks.
type Provider struct {
	path    string
	watcher *fsnotify.Watcher
	logger  *slog.Logger
	mu      sync.RWMutex
	current *config.Config
}

// NewProvider creates a new file-based config provider.
func NewProvider(path string) (*Provider, error) {
	if path == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	return &Provider{
		path:   path,
		logger: slog.Default(),
	}, nil
}

// Load loads the configuration from the file.
func (p *Provider) Load(ctx context.Context) (*config.Config, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config from %s: %w", p.path, err)
	}

	p.current = cfg
	p.logger.Info("config loaded", slog.String("path", p.path))

	return cfg, nil
}

// Watch watches the config file for changes and calls onChange when the file is modified.
func (p *Provider) Watch(ctx context.Context, onChange func(*config.Config)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	p.mu.Lock()
	p.watcher = watcher
	p.mu.Unlock()

	if err := watcher.Add(p.path); err != nil {
		watcher.Close()
		return fmt.Errorf("watch %s: %w", p.path, err)
	}

	p.logger.Info("watching config file for changes", slog.String("path", p.path))

	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				p.logger.Debug("config watch stopped")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only reload on write events
				if event.Op&fsnotify.Write == fsnotify.Write {
					p.logger.Info("config file changed, reloading", slog.String("path", event.Name))

					cfg, err := config.Load()
					if err != nil {
						p.logger.Error("failed to reload config",
							slog.String("error", err.Error()),
							slog.String("path", p.path))
						continue
					}

					p.mu.Lock()
					p.current = cfg
					p.mu.Unlock()

					// Call the onChange callback
					onChange(cfg)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				p.logger.Error("config watch error", slog.String("error", err.Error()))
			}
		}
	}()

	return nil
}

// Close stops watching the config file.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.watcher != nil {
		return p.watcher.Close()
	}

	return nil
}
