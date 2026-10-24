package scraper

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// Factory instantiates a Scraper from configuration.
type Factory func(cfg config.ScraperInstanceConfig) (Scraper, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]Factory)
)

// RegisterFactory registers a scraper provider factory (e.g. "tmdb").
func RegisterFactory(scraperType string, f Factory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(scraperType)] = f
}

// CreateScraper instantiates a Scraper based on the configured instance type.
func CreateScraper(cfg config.ScraperInstanceConfig) (Scraper, error) {
	scType := strings.ToLower(cfg.Type)
	if scType == "" {
		scType = "tmdb"
	}

	factoriesMu.RLock()
	factory, exists := factories[scType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported scraper type '%s' (registered types: %v)", scType, AvailableTypes())
	}

	return factory(cfg)
}

// AvailableTypes returns a list of registered scraper provider types.
func AvailableTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}
