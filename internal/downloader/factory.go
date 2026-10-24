package downloader

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// ClientFactory creates a Downloader instance from configuration.
type ClientFactory func(cfg config.DownloaderInstanceConfig) (Downloader, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]ClientFactory)
)

// RegisterFactory registers a downloader factory for a given type name (e.g. "qbittorrent").
func RegisterFactory(downloaderType string, f ClientFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(downloaderType)] = f
}

// CreateDownloader instantiates a Downloader client for the configured instance.
func CreateDownloader(cfg config.DownloaderInstanceConfig) (Downloader, error) {
	dlType := strings.ToLower(cfg.Type)
	if dlType == "" {
		dlType = "qbittorrent"
	}

	factoriesMu.RLock()
	factory, exists := factories[dlType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported downloader type '%s' (registered types: %v)", dlType, AvailableTypes())
	}

	return factory(cfg)
}

// AvailableTypes returns a list of all registered downloader types.
func AvailableTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}
