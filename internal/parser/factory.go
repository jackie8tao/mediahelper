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

// MatcherFactory creates a Matcher instance from configuration.
type MatcherFactory func(cfg json.RawMessage) (Matcher, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]ParserFactory)

	matchersMu sync.RWMutex
	matchers   = make(map[string]MatcherFactory)
)

// RegisterParserFactory registers a parser factory for a given type name (e.g. "deepseek", "openai").
func RegisterParserFactory(parserType string, f ParserFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(parserType)] = f
}

// RegisterMatcherFactory registers a matcher factory for a given provider name (e.g. "tmdb", "douban").
func RegisterMatcherFactory(matcherType string, f MatcherFactory) {
	matchersMu.Lock()
	defer matchersMu.Unlock()
	matchers[strings.ToLower(matcherType)] = f
}

// CreateMatcher instantiates a Matcher client from configuration.
func CreateMatcher(matcherType string, cfg json.RawMessage) (Matcher, error) {
	matchersMu.RLock()
	factory, exists := matchers[strings.ToLower(matcherType)]
	matchersMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported matcher type '%s'", matcherType)
	}

	return factory(cfg)
}

// CreateParser instantiates a Parser client for the configured instance.
// If the configuration includes a nested matcher, it automatically wraps the recognizer in a StandardParser.
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

	baseParser, err := factory(cfg.Detail)
	if err != nil {
		return nil, err
	}

	// Check if a matcher is configured inside detail
	if len(cfg.Detail) > 0 {
		var wrapper struct {
			Matcher *struct {
				Type   string          `json:"type"`
				Detail json.RawMessage `json:"detail"`
			} `json:"matcher"`
		}
		if json.Unmarshal(cfg.Detail, &wrapper) == nil && wrapper.Matcher != nil && wrapper.Matcher.Type != "" {
			matcherCfgBytes := wrapper.Matcher.Detail
			if len(matcherCfgBytes) == 0 {
				matcherCfgBytes = cfg.Detail
			}
			m, mErr := CreateMatcher(wrapper.Matcher.Type, matcherCfgBytes)
			if mErr != nil {
				return nil, fmt.Errorf("initialize parser matcher '%s' error: %w", wrapper.Matcher.Type, mErr)
			}
			if rec, ok := baseParser.(Recognizer); ok {
				return NewStandardParser(rec, m), nil
			}
		}
	}

	return baseParser, nil
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

// AvailableMatcherTypes returns a list of all registered matcher types.
func AvailableMatcherTypes() []string {
	matchersMu.RLock()
	defer matchersMu.RUnlock()
	types := make([]string, 0, len(matchers))
	for t := range matchers {
		types = append(types, t)
	}
	return types
}
