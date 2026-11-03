package organizer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// OrganizerComponent adapts an Organizer implementation to the pipeline.Component interface.
type OrganizerComponent struct {
	organizer Organizer
	cfg       config.OrganizerComponentConfig
}

func (c *OrganizerComponent) ID() string                             { return c.cfg.ID }
func (c *OrganizerComponent) Name() string                           { return c.cfg.Name }
func (c *OrganizerComponent) Description() string                    { return c.cfg.Description }
func (c *OrganizerComponent) Organizer() Organizer                   { return c.organizer }
func (c *OrganizerComponent) Config() config.OrganizerComponentConfig { return c.cfg }

// Execute handles organizer operations within a pipeline step.
func (c *OrganizerComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	return nil, nil
}

// RegisterComponents instantiates organizer components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*OrganizerComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetOrganizerConfigs()
	if len(configs) == 0 {
		slog.Debug("No organizer instances configured")
		return nil, nil
	}

	var registered []*OrganizerComponent

	for _, orgCfg := range configs {
		if !orgCfg.Enabled {
			slog.Debug("Organizer instance is disabled, skipping registration", "id", orgCfg.ID)
			continue
		}

		org, err := CreateOrganizer(orgCfg)
		if err != nil {
			return nil, fmt.Errorf("create organizer component '%s' error: %w", orgCfg.ID, err)
		}
		comp := &OrganizerComponent{
			organizer: org,
			cfg:       orgCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register organizer component '%s' error: %w", orgCfg.ID, err)
		}

		slog.Info("Registered organizer pipeline component", "id", orgCfg.ID, "type", orgCfg.Type, "name", orgCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
