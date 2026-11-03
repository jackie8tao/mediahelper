package parser

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// ParserFactory creates a Parser instance from configuration.
type ParserFactory func(cfg json.RawMessage) (Parser, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]ParserFactory)
)

// RegisterParserFactory registers a parser factory for a given type name (e.g. "openai", "gemini").
func RegisterParserFactory(parserType string, f ParserFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(parserType)] = f
}

// CreateParser instantiates a Parser client for the configured instance.
func CreateParser(cfg config.ParserComponentConfig) (Parser, error) {
	pType := strings.ToLower(cfg.Type)
	if pType == "" {
		return nil, fmt.Errorf("parser type cannot be empty")
	}

	factoriesMu.RLock()
	factory, exists := factories[pType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported parser type '%s' (registered types: %v)", pType, AvailableParserTypes())
	}

	return factory(cfg.Detail)
}

// AvailableParserTypes returns a list of all registered parser types.
func AvailableParserTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
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
