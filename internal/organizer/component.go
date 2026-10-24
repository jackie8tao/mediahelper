package organizer

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// OrganizerComponent wraps library organization logic as a pipeline component.
type OrganizerComponent struct {
	id        string
	name      string
	cfg       config.OrganizerInstanceConfig
	organizer *Organizer
}

// NewComponent creates a new OrganizerComponent instance.
func NewComponent(cfg config.OrganizerInstanceConfig, organizer *Organizer) *OrganizerComponent {
	id := cfg.ID
	if id == "" {
		id = "organizer"
	}
	name := cfg.Name
	if name == "" {
		name = "Organizer"
	}

	return &OrganizerComponent{
		id:        id,
		name:      name,
		cfg:       cfg,
		organizer: organizer,
	}
}

// ID returns the unique component identifier.
func (o *OrganizerComponent) ID() string {
	return o.id
}

// Name returns the friendly display name.
func (o *OrganizerComponent) Name() string {
	return o.name
}

// Description describes the component capability and input/output parameters.
func (o *OrganizerComponent) Description() string {
	return "Organizes media files into standardized library structures via hardlink, copy, move, or symlink"
}

// Organizer returns the underlying Organizer engine.
func (o *OrganizerComponent) Organizer() *Organizer {
	return o.organizer
}

// Config returns the instance configuration.
func (o *OrganizerComponent) Config() config.OrganizerInstanceConfig {
	return o.cfg
}

// Execute performs media organization for incoming task or extracted parameters.
func (o *OrganizerComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if o.organizer == nil {
		return nil, fmt.Errorf("organizer engine is not initialized for component '%s'", o.id)
	}

	task := o.resolveTask(sCtx)
	if task.SourcePath == "" {
		return nil, fmt.Errorf("no source_path, content_path, or torrent provided for organizer component '%s'", o.id)
	}

	if task.MediaInfo == nil {
		task.MediaInfo = parser.Parse(filepath.Base(task.SourcePath))
	}

	res, err := o.organizer.Organize(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("organizer '%s' failed for '%s': %w", o.id, task.SourcePath, err)
	}

	sCtx.Log(fmt.Sprintf("Organized successfully: TargetDir='%s', LinkedFiles=%d, Success=%v",
		res.TargetDir, len(res.LinkedFiles), res.Success))

	outputs := map[string]any{
		"organize_result": res,
		"success":         res.Success,
		"target_dir":      res.TargetDir,
		"linked_files":    res.LinkedFiles,
		"nfo_path":        res.NFOPath,
		"poster_path":     res.PosterPath,
		"fanart_path":     res.FanartPath,
	}

	return outputs, nil
}

// resolveTask extracts or constructs OrganizeTask from parameters, upstream steps, or event payload.
func (o *OrganizerComponent) resolveTask(sCtx *pipeline.StepContext) OrganizeTask {
	// 1. Direct OrganizeTask in Params
	if v, ok := sCtx.Params["task"]; ok {
		if t, ok := v.(OrganizeTask); ok {
			return t
		}
	}

	task := OrganizeTask{}

	// 2. Resolve source path
	for _, key := range []string{"source_path", "local_content_path", "local_path", "content_path", "path"} {
		if val := sCtx.GetStringParam(key, ""); val != "" {
			task.SourcePath = val
			break
		}
	}

	if task.SourcePath == "" && sCtx.Params != nil {
		if tRaw, ok := sCtx.Params["torrent"]; ok {
			if t, ok := tRaw.(downloader.Torrent); ok {
				if t.ContentPath != "" {
					task.SourcePath = t.ContentPath
				} else {
					task.SourcePath = t.SavePath
				}
			}
		}
	}

	// Fallback source path from upstream steps
	if task.SourcePath == "" && sCtx.Steps != nil {
		for _, stepOutputs := range sCtx.Steps {
			for _, key := range []string{"local_content_path", "local_path", "content_path", "path"} {
				if val, ok := stepOutputs[key].(string); ok && val != "" {
					task.SourcePath = val
					break
				}
			}
			if task.SourcePath != "" {
				break
			}
			if tRaw, ok := stepOutputs["torrent"]; ok {
				if t, ok := tRaw.(downloader.Torrent); ok {
					if t.ContentPath != "" {
						task.SourcePath = t.ContentPath
					} else {
						task.SourcePath = t.SavePath
					}
					break
				}
			}
		}
	}

	// Fallback source path from event payload
	if task.SourcePath == "" && sCtx.Event != nil {
		for _, key := range []string{"local_content_path", "local_path", "content_path", "path"} {
			if val, ok := sCtx.Event[key].(string); ok && val != "" {
				task.SourcePath = val
				break
			}
		}
		if task.SourcePath == "" {
			if tRaw, ok := sCtx.Event["torrent"]; ok {
				if t, ok := tRaw.(downloader.Torrent); ok {
					if t.ContentPath != "" {
						task.SourcePath = t.ContentPath
					} else {
						task.SourcePath = t.SavePath
					}
				}
			}
		}
	}

	// 3. Resolve MediaInfo
	if v, ok := sCtx.Params["media_info"]; ok {
		if info, ok := v.(*parser.MediaInfo); ok {
			task.MediaInfo = info
		}
	}
	if task.MediaInfo == nil && sCtx.Steps != nil {
		for _, stepOutputs := range sCtx.Steps {
			if v, ok := stepOutputs["media_info"]; ok {
				if info, ok := v.(*parser.MediaInfo); ok {
					task.MediaInfo = info
					break
				}
			}
		}
	}
	if task.MediaInfo == nil && sCtx.Event != nil {
		if v, ok := sCtx.Event["media_info"]; ok {
			if info, ok := v.(*parser.MediaInfo); ok {
				task.MediaInfo = info
			}
		}
	}

	// 4. Resolve MediaMetadata
	if v, ok := sCtx.Params["metadata"]; ok {
		if meta, ok := v.(*scraper.MediaMetadata); ok {
			task.Metadata = meta
		}
	}
	if task.Metadata == nil && sCtx.Steps != nil {
		for _, stepOutputs := range sCtx.Steps {
			if v, ok := stepOutputs["metadata"]; ok {
				if meta, ok := v.(*scraper.MediaMetadata); ok {
					task.Metadata = meta
					break
				}
			}
		}
	}
	if task.Metadata == nil && sCtx.Event != nil {
		if v, ok := sCtx.Event["metadata"]; ok {
			if meta, ok := v.(*scraper.MediaMetadata); ok {
				task.Metadata = meta
			}
		}
	}

	return task
}

// RegisterComponents registers configured organizer instances into the pipeline registry.
func RegisterComponents(cfg *config.Config, registry *pipeline.ComponentRegistry, scraperSvc scraper.Scraper) ([]*OrganizerComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}

	organizerConfigs := cfg.GetOrganizerConfigs()
	var registered []*OrganizerComponent

	for _, orgCfg := range organizerConfigs {
		if !orgCfg.Enabled {
			slog.Debug("Skipping disabled organizer instance", "id", orgCfg.ID)
			continue
		}

		org := NewOrganizerFromInstance(orgCfg, scraperSvc)
		comp := NewComponent(orgCfg, org)

		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register organizer component '%s' error: %w", comp.ID(), err)
		}

		registered = append(registered, comp)
		slog.Info("Registered organizer pipeline component", "id", comp.ID(), "link_mode", orgCfg.LinkMode, "name", comp.Name())
	}

	// Register default alias if "organizer" is not already registered
	if len(registered) > 0 {
		if _, exists := registry.Get("organizer"); !exists {
			aliasComp := NewComponent(registered[0].cfg, registered[0].organizer)
			aliasComp.id = "organizer"
			_ = registry.Register(aliasComp)
			slog.Info("Registered default 'organizer' alias pointing to first instance", "target", registered[0].ID())
		}
	}

	return registered, nil
}
