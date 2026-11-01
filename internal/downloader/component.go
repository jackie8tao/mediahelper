package downloader

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// DownloaderComponent adapts a Downloader implementation to the pipeline.Component interface.
type DownloaderComponent struct {
	dl  Downloader
	cfg config.DownloaderComponentConfig
}

func (c *DownloaderComponent) ID() string                               { return c.cfg.ID }
func (c *DownloaderComponent) Name() string                             { return c.cfg.Name }
func (c *DownloaderComponent) Description() string                      { return c.cfg.Description }
func (c *DownloaderComponent) Downloader() Downloader                   { return c.dl }
func (c *DownloaderComponent) Config() config.DownloaderComponentConfig { return c.cfg }

// Execute handles downloader operations within a pipeline step.
func (c *DownloaderComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	return nil, nil
}

// RegisterComponents instantiates downloader components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*DownloaderComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetDownloaderConfigs()
	if len(configs) == 0 {
		slog.Debug("No downloader instances configured")
		return nil, nil
	}

	var registered []*DownloaderComponent

	for _, dlCfg := range configs {
		if !dlCfg.Enabled {
			slog.Debug("Downloader instance is disabled, skipping registration", "id", dlCfg.ID)
			continue
		}

		dl, err := CreateDownloader(dlCfg)
		if err != nil {
			return nil, fmt.Errorf("create downloader component '%s' error: %w", dlCfg.ID, err)
		}
		comp := &DownloaderComponent{
			dl:  dl,
			cfg: dlCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register downloader component '%s' error: %w", dlCfg.ID, err)
		}

		slog.Info("Registered downloader pipeline component", "id", dlCfg.ID, "type", dlCfg.Type, "name", dlCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
