package parser

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// ParserComponent adapts a Parser implementation to the pipeline.Component interface.
type ParserComponent struct {
	parser Parser
	cfg    config.ParserComponentConfig
}

func (c *ParserComponent) ID() string                          { return c.cfg.ID }
func (c *ParserComponent) Name() string                        { return c.cfg.Name }
func (c *ParserComponent) Description() string                 { return c.cfg.Description }
func (c *ParserComponent) Parser() Parser                      { return c.parser }
func (c *ParserComponent) Config() config.ParserComponentConfig { return c.cfg }

// Execute handles parser operations within a pipeline step.
func (c *ParserComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	return nil, nil
}

// RegisterComponents instantiates parser components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*ParserComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetParserConfigs()
	if len(configs) == 0 {
		slog.Debug("No parser instances configured")
		return nil, nil
	}

	var registered []*ParserComponent

	for _, pCfg := range configs {
		if !pCfg.Enabled {
			slog.Debug("Parser instance is disabled, skipping registration", "id", pCfg.ID)
			continue
		}

		p, err := CreateParser(pCfg)
		if err != nil {
			return nil, fmt.Errorf("create parser component '%s' error: %w", pCfg.ID, err)
		}
		comp := &ParserComponent{
			parser: p,
			cfg:    pCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register parser component '%s' error: %w", pCfg.ID, err)
		}

		slog.Info("Registered parser pipeline component", "id", pCfg.ID, "type", pCfg.Type, "name", pCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
