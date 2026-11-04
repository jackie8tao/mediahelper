package organizer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// Supported organizer component action constants.
const (
	ActionOrganize = "organize"
)

// OrganizerComponent adapts an Organizer implementation to the pipeline.Component interface.
type OrganizerComponent struct {
	organizer Organizer
	cfg       config.OrganizerComponentConfig
}

func (c *OrganizerComponent) ID() string                             { return c.cfg.ID }
func (c *OrganizerComponent) Name() string                           { return c.cfg.Name }
func (c *OrganizerComponent) Description() string                    { return c.cfg.Description }
func (c *OrganizerComponent) Organizer() Organizer                   { return c.organizer }
func (c *OrganizerComponent) Config() config.OrganizerComponentConfig { return c.cfg }

// Execute dispatches organizer operations within a pipeline step according to the "action" parameter.
// If "action" is omitted, it defaults to ActionOrganize ("organize").
func (c *OrganizerComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if c.organizer == nil {
		return nil, fmt.Errorf("organizer client is not initialized for component '%s'", c.cfg.ID)
	}
	if sCtx == nil {
		return nil, fmt.Errorf("step context cannot be nil for organizer component '%s'", c.cfg.ID)
	}

	action := strings.ToLower(strings.TrimSpace(sCtx.GetStringParam("action", ActionOrganize)))
	if action == "" {
		action = ActionOrganize
	}

	switch action {
	case ActionOrganize:
		return c.execOrganize(ctx, sCtx)
	default:
		return nil, fmt.Errorf("unsupported action '%s' for organizer component '%s'", action, c.cfg.ID)
	}
}

// execOrganize resolves an OrganizeTask from context and delegates file organization to the underlying Organizer.
func (c *OrganizerComponent) execOrganize(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	task := c.resolveTask(sCtx)
	if strings.TrimSpace(task.SourcePath) == "" {
		return nil, fmt.Errorf("no source_path, content_path, or path provided for organizer component '%s'", c.cfg.ID)
	}

	res, err := c.organizer.Organize(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("organizer '%s' failed for '%s': %w", c.cfg.ID, task.SourcePath, err)
	}
	if res == nil {
		res = &OrganizeResult{}
	}

	sCtx.Logf("Organized '%s' via %s -> TargetDir='%s', LinkedFiles=%d, Success=%v",
		task.SourcePath, c.organizer.Name(), res.TargetDir, len(res.LinkedFiles), res.Success)

	return map[string]any{
		"action":          ActionOrganize,
		"organize_result": res,
		"success":         res.Success,
		"target_dir":      res.TargetDir,
		"linked_files":    res.LinkedFiles,
		"file_count":      len(res.LinkedFiles),
		"source_path":     task.SourcePath,
		"media_type":      task.MediaType,
		"title":           task.Title,
		"year":            task.Year,
		"season":          task.Season,
		"episode":         task.Episode,
	}, nil
}

// resolveTask constructs an OrganizeTask by merging fields in priority order:
// 1. Explicit step parameters (sCtx.Params)
// 2. Upstream scraped metadata steps (steps containing "tmdb_id" or "imdb_id")
// 3. Other upstream steps (e.g. parser or downloader outputs in sCtx.Steps)
// 4. Trigger event payload (sCtx.Event)
func (c *OrganizerComponent) resolveTask(sCtx *pipeline.StepContext) OrganizeTask {
	if sCtx == nil {
		return OrganizeTask{}
	}

	// Allow passing a pre-built OrganizeTask directly in Params["task"]
	if sCtx.Params != nil {
		switch t := sCtx.Params["task"].(type) {
		case OrganizeTask:
			return t
		case *OrganizeTask:
			if t != nil {
				return *t
			}
		}
	}

	// Partition upstream step outputs into scraped (higher priority for naming metadata) and general steps
	var scrapedSteps []map[string]any
	var generalSteps []map[string]any
	for _, stepOut := range sCtx.Steps {
		if len(stepOut) == 0 {
			continue
		}
		if isScrapedStepOutput(stepOut) {
			scrapedSteps = append(scrapedSteps, stepOut)
		} else {
			generalSteps = append(generalSteps, stepOut)
		}
	}

	// Ordered sources from highest priority to lowest priority
	sources := make([]map[string]any, 0, 2+len(scrapedSteps)+len(generalSteps))
	if len(sCtx.Params) > 0 {
		sources = append(sources, sCtx.Params)
	}
	sources = append(sources, scrapedSteps...)
	sources = append(sources, generalSteps...)
	if len(sCtx.Event) > 0 {
		sources = append(sources, sCtx.Event)
	}

	return OrganizeTask{
		SourcePath:    lookupString(sources, "source_path", "content_path", "path", "file_path", "save_path"),
		MediaType:    lookupString(sources, "media_type", "category"),
		Title:        lookupString(sources, "title", "cn_title", "en_title", "name"),
		Year:         lookupInt(sources, "year"),
		Season:       lookupInt(sources, "season"),
		Episode:      lookupInt(sources, "episode"),
		EpisodeTitle: lookupString(sources, "episode_title"),
		Resolution:   lookupString(sources, "resolution"),
	}
}

// isScrapedStepOutput checks whether a step output map originates from a metadata scraper step.
func isScrapedStepOutput(m map[string]any) bool {
	if _, ok := m["tmdb_id"]; ok {
		return true
	}
	if _, ok := m["imdb_id"]; ok {
		return true
	}
	if _, ok := m["metadata"]; ok {
		return true
	}
	return false
}

// lookupString searches across ordered maps and returns the first non-empty string value matching any of the keys.
func lookupString(sources []map[string]any, keys ...string) string {
	for _, src := range sources {
		for _, k := range keys {
			v, ok := src[k]
			if !ok || v == nil {
				continue
			}
			if s, ok := v.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

// lookupInt searches across ordered maps and returns the first positive integer value matching any of the keys.
func lookupInt(sources []map[string]any, keys ...string) int {
	for _, src := range sources {
		for _, k := range keys {
			v, ok := src[k]
			if !ok || v == nil {
				continue
			}
			switch val := v.(type) {
			case int:
				if val > 0 {
					return val
				}
			case int64:
				if val > 0 {
					return int(val)
				}
			case float64:
				if int(val) > 0 {
					return int(val)
				}
			case string:
				if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && i > 0 {
					return i
				}
			}
		}
	}
	return 0
}
