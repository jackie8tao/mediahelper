package organizer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// OrganizerFactory creates an Organizer instance from configuration.
type OrganizerFactory func(cfg json.RawMessage) (Organizer, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]OrganizerFactory)
)

// RegisterOrganizerFactory registers an organizer factory for a given type name.
func RegisterOrganizerFactory(organizerType string, f OrganizerFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(organizerType)] = f
}

// CreateOrganizer instantiates an Organizer client for the configured instance.
func CreateOrganizer(cfg config.OrganizerComponentConfig) (Organizer, error) {
	orgType := strings.ToLower(cfg.Type)
	if orgType == "" {
		return nil, fmt.Errorf("organizer type cannot be empty")
	}

	factoriesMu.RLock()
	factory, exists := factories[orgType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported organizer type '%s' (registered types: %v)", orgType, AvailableOrganizerTypes())
	}

	return factory(cfg.Detail)
}

// AvailableOrganizerTypes returns a list of all registered organizer types.
func AvailableOrganizerTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
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

