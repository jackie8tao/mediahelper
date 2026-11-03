package downloader

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// DownloaderFactory creates a Downloader instance from configuration.
type DownloaderFactory func(cfg json.RawMessage) (Downloader, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]DownloaderFactory)
)

// RegisterDownloaderFactory registers a downloader factory for a given type name (e.g. "qbittorrent").
func RegisterDownloaderFactory(downloaderType string, f DownloaderFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[strings.ToLower(downloaderType)] = f
}

// CreateDownloader instantiates a Downloader client for the configured instance.
func CreateDownloader(cfg config.DownloaderComponentConfig) (Downloader, error) {
	dlType := strings.ToLower(cfg.Type)
	if dlType == "" {
		return nil, fmt.Errorf("downloader type cannot be empty")
	}

	factoriesMu.RLock()
	factory, exists := factories[dlType]
	factoriesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported downloader type '%s' (registered types: %v)", dlType, AvailableDownloaderTypes())
	}

	return factory(cfg.Detail)
}

// AvailableDownloaderTypes returns a list of all registered downloader types.
func AvailableDownloaderTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	types := make([]string, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

// RegisterComponents instantiates downloader components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*DownloaderComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetDownloaderConfigs()
	if len(configs) == 0 {
		slog.Debug("No downloader instances configured")
		return nil, nil
	}

	var registered []*DownloaderComponent

	for _, dlCfg := range configs {
		if !dlCfg.Enabled {
			slog.Debug("Downloader instance is disabled, skipping registration", "id", dlCfg.ID)
			continue
		}

		dl, err := CreateDownloader(dlCfg)
		if err != nil {
			return nil, fmt.Errorf("create downloader component '%s' error: %w", dlCfg.ID, err)
		}
		comp := &DownloaderComponent{
			dl:  dl,
			cfg: dlCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register downloader component '%s' error: %w", dlCfg.ID, err)
		}

		slog.Info("Registered downloader pipeline component", "id", dlCfg.ID, "type", dlCfg.Type, "name", dlCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
