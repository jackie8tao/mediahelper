package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// ParserFactory creates a Parser instance from configuration.
type ParserFactory func(cfg json.RawMessage) (Parser, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]ParserFactory)
)

// RegisterFactory registers a parser factory for a given type name (e.g. "openai", "gemini").
func RegisterFactory(parserType string, f ParserFactory) {
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
		return nil, fmt.Errorf("unsupported parser type '%s' (registered types: %v)", pType, AvailableTypes())
	}

	return factory(cfg.Detail)
}

// AvailableTypes returns a list of all registered parser types.
func AvailableTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

