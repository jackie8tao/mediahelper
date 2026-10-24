package pathmapper

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// PathMapper translates remote/container paths into local accessible filesystem paths.
type PathMapper struct {
	mappings []config.PathMapperInstanceConfig
}

// NewPathMapper creates a new PathMapper instance from configured mapping rules.
func NewPathMapper(mappings []config.PathMapperInstanceConfig) *PathMapper {
	return &PathMapper{mappings: mappings}
}

// ToLocal converts a remote path into a valid local filesystem path based on rules.
func (m *PathMapper) ToLocal(remotePath string) string {
	if remotePath == "" || len(m.mappings) == 0 {
		return filepath.Clean(remotePath)
	}

	normalizedRemote := normalizeSlash(remotePath)

	for _, mapping := range m.mappings {
		if mapping.Remote == "" || mapping.Local == "" {
			continue
		}

		normalizedRuleRemote := normalizeSlash(mapping.Remote)
		prefix := strings.TrimSuffix(normalizedRuleRemote, "/") + "/"

		if strings.HasPrefix(normalizedRemote, prefix) {
			rel := strings.TrimPrefix(normalizedRemote, prefix)
			localBase := strings.TrimSuffix(normalizeSlash(mapping.Local), "/")
			combined := localBase + "/" + rel
			return filepath.FromSlash(filepath.Clean(combined))
		}

		// Exact match without trailing relative path
		if normalizedRemote == strings.TrimSuffix(normalizedRuleRemote, "/") {
			return filepath.FromSlash(filepath.Clean(mapping.Local))
		}
	}

	return filepath.Clean(remotePath)
}

// ToRemote converts a local path back into a remote/container path.
func (m *PathMapper) ToRemote(localPath string) string {
	if localPath == "" || len(m.mappings) == 0 {
		return filepath.ToSlash(filepath.Clean(localPath))
	}

	normalizedLocal := normalizeSlash(localPath)

	for _, mapping := range m.mappings {
		if mapping.Remote == "" || mapping.Local == "" {
			continue
		}

		normalizedRuleLocal := normalizeSlash(mapping.Local)
		prefix := strings.TrimSuffix(normalizedRuleLocal, "/") + "/"

		if strings.HasPrefix(normalizedLocal, prefix) {
			rel := strings.TrimPrefix(normalizedLocal, prefix)
			remoteBase := strings.TrimSuffix(normalizeSlash(mapping.Remote), "/")
			combined := remoteBase + "/" + rel
			return filepath.ToSlash(filepath.Clean(combined))
		}

		if normalizedLocal == strings.TrimSuffix(normalizedRuleLocal, "/") {
			return filepath.ToSlash(filepath.Clean(mapping.Remote))
		}
	}

	return filepath.ToSlash(filepath.Clean(localPath))
}

// MapTorrent translates SavePath and ContentPath of a Torrent to local filesystem paths.
func (m *PathMapper) MapTorrent(t downloader.Torrent) downloader.Torrent {
	mapped := t
	mapped.SavePath = m.ToLocal(t.SavePath)
	mapped.ContentPath = m.ToLocal(t.ContentPath)
	return mapped
}

func normalizeSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// Component implements pipeline.Component for path translation in workflow pipelines.
type Component struct {
	id     string
	name   string
	mapper *PathMapper
}

// NewComponent creates a new PathMapper pipeline component.
func NewComponent(id string, name string, mapper *PathMapper) *Component {
	if id == "" {
		id = "path_mapper"
	}
	if name == "" {
		name = "Path Mapper"
	}
	return &Component{
		id:     id,
		name:   name,
		mapper: mapper,
	}
}

// ID returns the component identifier.
func (c *Component) ID() string {
	return c.id
}

// Name returns the friendly component name.
func (c *Component) Name() string {
	return c.name
}

// Description returns the component specification.
func (c *Component) Description() string {
	return "Translates remote download directory paths to local accessible paths"
}

// Execute performs path translation based on parameters or incoming event/step context.
func (c *Component) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	outputs := make(map[string]any)

	// 1. Check for single path parameter
	pathVal := sCtx.GetStringParam("path", "")
	if pathVal == "" {
		pathVal = sCtx.GetStringParam("file_path", "")
	}

	// 2. Check for save_path and content_path parameters
	savePathVal := sCtx.GetStringParam("save_path", "")
	contentPathVal := sCtx.GetStringParam("content_path", "")

	// 3. Fallback extraction from incoming Event payload if parameters not explicitly provided
	if pathVal == "" && sCtx.Event != nil {
		if p, ok := sCtx.Event["path"].(string); ok {
			pathVal = p
		} else if fp, ok := sCtx.Event["file_path"].(string); ok {
			pathVal = fp
		}
	}
	if savePathVal == "" && sCtx.Event != nil {
		if sp, ok := sCtx.Event["save_path"].(string); ok {
			savePathVal = sp
		}
	}
	if contentPathVal == "" && sCtx.Event != nil {
		if cp, ok := sCtx.Event["content_path"].(string); ok {
			contentPathVal = cp
		}
	}

	// 4. Check for torrent struct in params or event
	var torrentObj *downloader.Torrent
	if tRaw, ok := sCtx.Params["torrent"]; ok {
		if t, ok := tRaw.(downloader.Torrent); ok {
			torrentObj = &t
		}
	} else if tRaw, ok := sCtx.Event["torrent"]; ok {
		if t, ok := tRaw.(downloader.Torrent); ok {
			torrentObj = &t
		}
	}

	// Translate paths
	if pathVal != "" {
		localPath := c.mapper.ToLocal(pathVal)
		outputs["local_path"] = localPath
		sCtx.Log(fmt.Sprintf("Mapped path: '%s' -> '%s'", pathVal, localPath))
	}

	if savePathVal != "" {
		localSavePath := c.mapper.ToLocal(savePathVal)
		outputs["local_save_path"] = localSavePath
		sCtx.Log(fmt.Sprintf("Mapped save_path: '%s' -> '%s'", savePathVal, localSavePath))
	}

	if contentPathVal != "" {
		localContentPath := c.mapper.ToLocal(contentPathVal)
		outputs["local_content_path"] = localContentPath
		sCtx.Log(fmt.Sprintf("Mapped content_path: '%s' -> '%s'", contentPathVal, localContentPath))
	}

	if torrentObj != nil {
		mappedTorrent := c.mapper.MapTorrent(*torrentObj)
		outputs["torrent"] = mappedTorrent
		outputs["local_save_path"] = mappedTorrent.SavePath
		outputs["local_content_path"] = mappedTorrent.ContentPath
		sCtx.Log(fmt.Sprintf("Mapped torrent '%s' paths: save='%s', content='%s'",
			mappedTorrent.Name, mappedTorrent.SavePath, mappedTorrent.ContentPath))
	}

	return outputs, nil
}

// RegisterComponent instantiates and registers the path mapper component into the pipeline registry.
func RegisterComponent(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) (*Component, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry is nil")
	}

	mappings := cfg.GetPathMappings()
	mapper := NewPathMapper(mappings)
	comp := NewComponent("path_mapper", "Path Mapper", mapper)

	if err := registry.Register(comp); err != nil {
		return nil, fmt.Errorf("failed to register path_mapper component: %w", err)
	}

	slog.Info("Registered path mapper pipeline component", "id", comp.ID(), "rules_count", len(mappings))
	return comp, nil
}
