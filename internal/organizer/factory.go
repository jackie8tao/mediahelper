package organizer

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// OrganizerFactory creates an Organizer instance from configuration.
type OrganizerFactory func(cfg json.RawMessage) (Organizer, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]OrganizerFactory)
)

// RegisterFactory registers an organizer factory for a given type name.
func RegisterFactory(organizerType string, f OrganizerFactory) {
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
		return nil, fmt.Errorf("unsupported organizer type '%s' (registered types: %v)", orgType, AvailableTypes())
	}

	return factory(cfg.Detail)
}

// AvailableTypes returns a list of all registered organizer types.
func AvailableTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

