package organizer

import (
	"context"
	"fmt"
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

// execOrganize binds step parameters to an OrganizeTask and delegates file organization to the underlying Organizer.
func (c *OrganizerComponent) execOrganize(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	var task OrganizeTask
	if err := sCtx.BindParams(&task); err != nil {
		return nil, fmt.Errorf("bind organizer params failed for '%s': %w", c.cfg.ID, err)
	}

	if strings.TrimSpace(task.SourcePath) == "" {
		return nil, fmt.Errorf("missing required parameter 'source_path' for organizer component '%s'", c.cfg.ID)
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
