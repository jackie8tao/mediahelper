package scraper

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// ScraperComponent adapts a Scraper implementation to the pipeline.Component interface.
type ScraperComponent struct {
	scraper Scraper
	cfg     config.ScraperComponentConfig
}

func (c *ScraperComponent) ID() string                            { return c.cfg.ID }
func (c *ScraperComponent) Name() string                          { return c.cfg.Name }
func (c *ScraperComponent) Description() string                   { return c.cfg.Description }
func (c *ScraperComponent) Scraper() Scraper                      { return c.scraper }
func (c *ScraperComponent) Config() config.ScraperComponentConfig { return c.cfg }

// Execute handles scraper operations within a pipeline step.
func (c *ScraperComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	return nil, nil
}

// RegisterComponents instantiates scraper components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*ScraperComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetScraperConfigs()
	if len(configs) == 0 {
		slog.Debug("No scraper instances configured")
		return nil, nil
	}

	var registered []*ScraperComponent

	for _, scCfg := range configs {
		if !scCfg.Enabled {
			slog.Debug("Scraper instance is disabled, skipping registration", "id", scCfg.ID)
			continue
		}

		sc, err := CreateScraper(scCfg)
		if err != nil {
			return nil, fmt.Errorf("create scraper component '%s' error: %w", scCfg.ID, err)
		}
		comp := &ScraperComponent{
			scraper: sc,
			cfg:     scCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register scraper component '%s' error: %w", scCfg.ID, err)
		}

		slog.Info("Registered scraper pipeline component", "id", scCfg.ID, "type", scCfg.Type, "name", scCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
