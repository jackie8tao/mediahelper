package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// DownloaderComponent adapts a Downloader implementation to the pipeline.Component interface.
type DownloaderComponent struct {
	id          string
	name        string
	description string
	dl          Downloader
	cfg         config.DownloaderInstanceConfig
}

// NewDownloaderComponent creates a new DownloaderComponent instance.
func NewDownloaderComponent(
	id string,
	name string,
	dl Downloader,
	cfg config.DownloaderInstanceConfig,
) *DownloaderComponent {
	if id == "" {
		id = "downloader"
	}
	if name == "" {
		if dl != nil {
			name = dl.Name()
		} else {
			name = "Downloader"
		}
	}

	desc := fmt.Sprintf("Downloader component [%s] managing torrents and tags", name)
	return &DownloaderComponent{
		id:          id,
		name:        name,
		description: desc,
		dl:          dl,
		cfg:         cfg,
	}
}

func (c *DownloaderComponent) ID() string                              { return c.id }
func (c *DownloaderComponent) Name() string                            { return c.name }
func (c *DownloaderComponent) Description() string                     { return c.description }
func (c *DownloaderComponent) Downloader() Downloader                  { return c.dl }
func (c *DownloaderComponent) Config() config.DownloaderInstanceConfig { return c.cfg }

// Execute handles downloader operations within a pipeline step.
func (c *DownloaderComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if c.dl == nil {
		return nil, fmt.Errorf("downloader client is not initialized for component '%s'", c.id)
	}

	action := strings.ToLower(sCtx.GetStringParam("action", ""))
	if action == "" {
		action = strings.ToLower(sCtx.GetStringParam("operation", ""))
	}

	// Auto-deduce action if not explicitly given
	if action == "" {
		if _, ok := sCtx.GetParam("tags"); ok {
			action = "add_tags"
		} else if _, ok := sCtx.GetParam("tag"); ok {
			action = "add_tags"
		} else if _, ok := sCtx.GetParam("remove_tags"); ok {
			action = "remove_tags"
		} else if _, ok := sCtx.GetParam("remove_tag"); ok {
			action = "remove_tags"
		} else if _, ok := sCtx.GetParam("files"); ok {
			action = "get_files"
		} else if _, ok := sCtx.GetParam("categories"); ok {
			action = "get_completed"
		} else if _, ok := sCtx.GetParam("category"); ok {
			action = "get_completed"
		} else if _, ok := sCtx.GetParam("filter"); ok {
			action = "get_torrents"
		} else {
			action = "get_completed"
		}
	}

	switch action {
	case "add_tags", "tag", "add_tag":
		return c.handleAddTags(ctx, sCtx)
	case "remove_tags", "untag", "remove_tag":
		return c.handleRemoveTags(ctx, sCtx)
	case "get_completed", "query_completed":
		return c.handleGetCompleted(ctx, sCtx)
	case "get_torrents", "list":
		return c.handleGetTorrents(ctx, sCtx)
	case "get_files", "files":
		return c.handleGetFiles(ctx, sCtx)
	case "test_connection", "ping":
		return c.handleTestConnection(ctx, sCtx)
	default:
		return nil, fmt.Errorf("unsupported downloader action: %s", action)
	}
}

func (c *DownloaderComponent) handleAddTags(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := extractHashes(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("torrent hash or hashes parameter required for add_tags")
	}

	tags := extractTags(sCtx, c.cfg.SuccessTag, "MediaHelper:Done")
	if len(tags) == 0 {
		return nil, fmt.Errorf("tags parameter required for add_tags")
	}

	if err := c.dl.AddTags(ctx, hashes, tags); err != nil {
		sCtx.Logf("Failed to add tags %v to torrents %v: %v", tags, hashes, err)
		return nil, fmt.Errorf("add tags error: %w", err)
	}

	sCtx.Logf("Successfully added tags %v to torrents: %v", tags, hashes)
	return map[string]any{
		"action":  "add_tags",
		"success": true,
		"hashes":  hashes,
		"tags":    tags,
	}, nil
}

func (c *DownloaderComponent) handleRemoveTags(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := extractHashes(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("torrent hash or hashes parameter required for remove_tags")
	}

	tags := extractTags(sCtx, c.cfg.FailTag, "MediaHelper:Error")
	if len(tags) == 0 {
		return nil, fmt.Errorf("tags parameter required for remove_tags")
	}

	if err := c.dl.RemoveTags(ctx, hashes, tags); err != nil {
		sCtx.Logf("Failed to remove tags %v from torrents %v: %v", tags, hashes, err)
		return nil, fmt.Errorf("remove tags error: %w", err)
	}

	sCtx.Logf("Successfully removed tags %v from torrents: %v", tags, hashes)
	return map[string]any{
		"action":  "remove_tags",
		"success": true,
		"hashes":  hashes,
		"tags":    tags,
	}, nil
}

func (c *DownloaderComponent) handleGetCompleted(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	categories := extractStringSlice(sCtx, "categories", "category")
	if len(categories) == 0 {
		categories = c.cfg.Categories
	}

	torrents, err := c.dl.GetCompletedTorrents(ctx, categories)
	if err != nil {
		sCtx.Logf("Failed to fetch completed torrents: %v", err)
		return nil, fmt.Errorf("get completed torrents error: %w", err)
	}

	sCtx.Logf("Fetched %d completed torrents for categories: %v", len(torrents), categories)
	return map[string]any{
		"action":   "get_completed",
		"count":    len(torrents),
		"torrents": torrents,
	}, nil
}

func (c *DownloaderComponent) handleGetTorrents(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	filter := sCtx.GetStringParam("filter", "all")
	category := sCtx.GetStringParam("category", "")

	torrents, err := c.dl.GetAllTorrents(ctx, filter, category)
	if err != nil {
		sCtx.Logf("Failed to fetch torrents: %v", err)
		return nil, fmt.Errorf("get torrents error: %w", err)
	}

	sCtx.Logf("Fetched %d torrents (filter=%s, category=%s)", len(torrents), filter, category)
	return map[string]any{
		"action":   "get_torrents",
		"count":    len(torrents),
		"torrents": torrents,
	}, nil
}

func (c *DownloaderComponent) handleGetFiles(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := extractHashes(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("hash parameter required for get_files")
	}

	targetHash := hashes[0]
	files, err := c.dl.GetTorrentFiles(ctx, targetHash)
	if err != nil {
		sCtx.Logf("Failed to fetch files for torrent %s: %v", targetHash, err)
		return nil, fmt.Errorf("get torrent files error: %w", err)
	}

	sCtx.Logf("Fetched %d files for torrent %s", len(files), targetHash)
	return map[string]any{
		"action": "get_files",
		"hash":   targetHash,
		"count":  len(files),
		"files":  files,
	}, nil
}

func (c *DownloaderComponent) handleTestConnection(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if err := c.dl.TestConnection(ctx); err != nil {
		sCtx.Logf("Downloader connection test failed: %v", err)
		return nil, fmt.Errorf("test connection error: %w", err)
	}

	sCtx.Logf("Downloader '%s' connection test passed", c.dl.Name())
	return map[string]any{
		"action":  "test_connection",
		"success": true,
		"name":    c.dl.Name(),
	}, nil
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

		id := dlCfg.ID
		if id == "" {
			id = "downloader"
		}

		dlClient, err := CreateDownloader(dlCfg)
		if err != nil {
			return nil, fmt.Errorf("create downloader client '%s' error: %w", id, err)
		}

		dlType := dlCfg.Type
		if dlType == "" {
			dlType = "qbittorrent"
		}

		comp := NewDownloaderComponent(id, dlCfg.Name, dlClient, dlCfg)
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register downloader component '%s' error: %w", id, err)
		}

		slog.Info("Registered downloader pipeline component", "id", id, "type", dlType, "name", dlCfg.Name)
		registered = append(registered, comp)
	}

	// If only 1 component was registered and its ID is not "downloader", register an alias as "downloader"
	if len(registered) == 1 && registered[0].ID() != "downloader" {
		if _, exists := registry.Get("downloader"); !exists {
			single := registered[0]
			aliasComp := NewDownloaderComponent("downloader", single.Name(), single.Downloader(), single.Config())
			_ = registry.Register(aliasComp)
			slog.Info("Registered alias 'downloader' for pipeline component", "targetID", single.ID())
		}
	}

	return registered, nil
}

func extractHashes(sCtx *pipeline.StepContext) []string {
	var hashes []string
	if v, ok := sCtx.GetParam("hashes"); ok {
		hashes = append(hashes, parseSlice(v)...)
	}
	if v, ok := sCtx.GetParam("hash"); ok {
		hashes = append(hashes, parseSlice(v)...)
	}
	if len(hashes) == 0 {
		if h, ok := sCtx.Event["hash"]; ok && h != nil {
			str := strings.TrimSpace(fmt.Sprintf("%v", h))
			if str != "" && str != "<nil>" {
				hashes = append(hashes, str)
			}
		}
	}
	return deduplicateStrings(hashes)
}

func extractTags(sCtx *pipeline.StepContext, fallbackTag string, defaultTag string) []string {
	var tags []string
	if v, ok := sCtx.GetParam("tags"); ok {
		tags = append(tags, parseSlice(v)...)
	}
	if v, ok := sCtx.GetParam("tag"); ok {
		tags = append(tags, parseSlice(v)...)
	}
	if len(tags) == 0 {
		if fallbackTag != "" {
			tags = append(tags, fallbackTag)
		} else if defaultTag != "" {
			tags = append(tags, defaultTag)
		}
	}
	return deduplicateStrings(tags)
}

func extractStringSlice(sCtx *pipeline.StepContext, keys ...string) []string {
	var res []string
	for _, k := range keys {
		if v, ok := sCtx.GetParam(k); ok {
			res = append(res, parseSlice(v)...)
		}
	}
	return deduplicateStrings(res)
}

func parseSlice(val any) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			var res []string
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					res = append(res, trimmed)
				}
			}
			return res
		}
		return []string{v}
	case []string:
		return v
	case []any:
		var res []string
		for _, item := range v {
			if item != nil {
				trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
				if trimmed != "" && trimmed != "<nil>" {
					res = append(res, trimmed)
				}
			}
		}
		return res
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" && str != "<nil>" {
			return []string{str}
		}
		return nil
	}
}

func deduplicateStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return result
}
