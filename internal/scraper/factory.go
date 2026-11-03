package scraper

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// ScraperFactory creates a Scraper instance from configuration.
type ScraperFactory func(cfg json.RawMessage) (Scraper, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]ScraperFactory)
)

// RegisterFactory registers a scraper factory for a given type name (e.g. "tmdb").
func RegisterFactory(scraperType string, f ScraperFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(scraperType)] = f
}

// CreateScraper instantiates a Scraper client for the configured instance.
func CreateScraper(cfg config.ScraperComponentConfig) (Scraper, error) {
	scType := strings.ToLower(cfg.Type)
	if scType == "" {
		return nil, fmt.Errorf("scraper type cannot be empty")
	}

	factoriesMu.RLock()
	factory, exists := factories[scType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported scraper type '%s' (registered types: %v)", scType, AvailableTypes())
	}

	return factory(cfg.Detail)
}

// AvailableTypes returns a list of all registered scraper types.
func AvailableTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}
