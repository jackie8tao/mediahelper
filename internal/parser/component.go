package parser

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

// ParserComponent wraps title/filename parsing logic as a pipeline component.
type ParserComponent struct {
	id  string
	cfg config.ParserInstanceConfig
}

// NewComponent creates a new ParserComponent instance.
func NewComponent(cfg config.ParserInstanceConfig) *ParserComponent {
	id := cfg.ID
	if id == "" {
		id = "parser"
	}
	name := cfg.Name
	if name == "" {
		name = "Parser"
	}
	cfg.ID = id
	cfg.Name = name

	return &ParserComponent{
		id:  id,
		cfg: cfg,
	}
}

// ID returns the unique component identifier.
func (p *ParserComponent) ID() string {
	return p.id
}

// Name returns the friendly display name.
func (p *ParserComponent) Name() string {
	return p.cfg.Name
}

// Description describes the component capability and input/output parameters.
func (p *ParserComponent) Description() string {
	return "Parses raw torrent names or media file paths into structured media metadata"
}

// Config returns the instance configuration.
func (p *ParserComponent) Config() config.ParserInstanceConfig {
	return p.cfg
}

// Execute performs media title and specification parsing.
func (p *ParserComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	rawText := p.resolveRawInput(sCtx)
	if strings.TrimSpace(rawText) == "" {
		return nil, fmt.Errorf("no title, filename, or torrent provided for parser component '%s'", p.id)
	}

	info := Parse(rawText)

	// Apply default media type fallback if specified and season/episode wasn't detected
	if p.cfg.DefaultMediaType != "" && info.Season == 0 && info.Episode == 0 {
		info.MediaType = MediaType(p.cfg.DefaultMediaType)
	}

	// Apply category hint if provided via parameters or event
	category := sCtx.GetStringParam("category", "")
	if category == "" && sCtx.Event != nil {
		if cat, ok := sCtx.Event["category"].(string); ok {
			category = cat
		}
	}
	if category != "" {
		catLower := strings.ToLower(category)
		if catLower == "movie" || catLower == "movies" || catLower == "\u7535\u5f71" {
			info.MediaType = MediaTypeMovie
		} else if catLower == "tv" || catLower == "teleplay" || catLower == "\u7535\u89c6\u5267" {
			info.MediaType = MediaTypeTV
		} else if catLower == "anime" || catLower == "animation" || catLower == "\u52a8\u6f2b" {
			info.MediaType = MediaTypeAnime
		}
	}

	sCtx.Log(fmt.Sprintf("Parsed title '%s' -> Title='%s', Year=%d, Type=%s, S%02dE%02d",
		rawText, info.Title, info.Year, info.MediaType, info.Season, info.Episode))

	outputs := map[string]any{
		"media_info":    info,
		"title":         info.Title,
		"cn_title":      info.CnTitle,
		"en_title":      info.EnTitle,
		"year":          info.Year,
		"media_type":    string(info.MediaType),
		"season":        info.Season,
		"episode":       info.Episode,
		"episode_end":   info.EpisodeEnd,
		"resolution":    info.Resolution,
		"source":        info.Source,
		"release_group": info.ReleaseGroup,
	}

	return outputs, nil
}

// resolveRawInput extracts raw title or filename from parameters, event payload, or upstream steps.
func (p *ParserComponent) resolveRawInput(sCtx *pipeline.StepContext) string {
	// 1. Direct string parameters
	for _, key := range []string{"raw", "title", "name"} {
		if val := sCtx.GetStringParam(key, ""); val != "" {
			return val
		}
	}

	// 2. Direct path parameter (extract basename)
	for _, key := range []string{"path", "file_path", "local_content_path", "local_path"} {
		if val := sCtx.GetStringParam(key, ""); val != "" {
			return filepath.Base(val)
		}
	}

	// 3. Torrent object in parameters
	if tRaw, ok := sCtx.Params["torrent"]; ok {
		if t, ok := tRaw.(downloader.Torrent); ok && t.Name != "" {
			return t.Name
		}
	}

	// 4. Event payload extraction
	if sCtx.Event != nil {
		if tRaw, ok := sCtx.Event["torrent"]; ok {
			if t, ok := tRaw.(downloader.Torrent); ok && t.Name != "" {
				return t.Name
			}
		}
		for _, key := range []string{"raw", "title", "name"} {
			if val, ok := sCtx.Event[key].(string); ok && val != "" {
				return val
			}
		}
		for _, key := range []string{"path", "file_path", "local_content_path", "local_path", "content_path"} {
			if val, ok := sCtx.Event[key].(string); ok && val != "" {
				return filepath.Base(val)
			}
		}
	}

	// 5. Upstream steps extraction
	if sCtx.Steps != nil {
		for _, stepOutputs := range sCtx.Steps {
			if tRaw, ok := stepOutputs["torrent"]; ok {
				if t, ok := tRaw.(downloader.Torrent); ok && t.Name != "" {
					return t.Name
				}
			}
			for _, key := range []string{"local_content_path", "local_path", "content_path", "path"} {
				if val, ok := stepOutputs[key].(string); ok && val != "" {
					return filepath.Base(val)
				}
			}
			for _, key := range []string{"raw", "title", "name"} {
				if val, ok := stepOutputs[key].(string); ok && val != "" {
					return val
				}
			}
		}
	}

	return ""
}

// RegisterComponents registers configured parser instances into the pipeline registry.
func RegisterComponents(cfg *config.Config, registry *pipeline.ComponentRegistry) ([]*ParserComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}

	parserConfigs := cfg.GetParserConfigs()
	var registered []*ParserComponent

	for _, pCfg := range parserConfigs {
		if !pCfg.Enabled {
			slog.Debug("Skipping disabled parser instance", "id", pCfg.ID)
			continue
		}

		comp := NewComponent(pCfg)
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register parser component '%s' error: %w", comp.ID(), err)
		}

		registered = append(registered, comp)
		slog.Info("Registered parser pipeline component", "id", comp.ID(), "name", comp.Name())
	}

	// Register default alias if "parser" is not already registered
	if len(registered) > 0 {
		if _, exists := registry.Get("parser"); !exists {
			aliasComp := NewComponent(registered[0].cfg)
			aliasComp.id = "parser"
			_ = registry.Register(aliasComp)
			slog.Info("Registered default 'parser' alias pointing to first instance", "target", registered[0].ID())
		}
	}

	return registered, nil
}
