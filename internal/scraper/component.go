package scraper

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// ScraperComponent wraps metadata scraping as a pipeline component.
type ScraperComponent struct {
	id      string
	name    string
	cfg     config.ScraperInstanceConfig
	scraper Scraper
}

// NewComponent creates a new ScraperComponent instance.
func NewComponent(cfg config.ScraperInstanceConfig, scraper Scraper) *ScraperComponent {
	id := cfg.ID
	if id == "" {
		id = "scraper"
	}
	name := cfg.Name
	if name == "" {
		name = "Scraper"
	}

	return &ScraperComponent{
		id:      id,
		name:    name,
		cfg:     cfg,
		scraper: scraper,
	}
}

// ID returns the unique component identifier.
func (s *ScraperComponent) ID() string {
	return s.id
}

// Name returns the friendly display name.
func (s *ScraperComponent) Name() string {
	return s.name
}

// Description describes the component capability and input/output parameters.
func (s *ScraperComponent) Description() string {
	return "Scrapes rich media metadata and artwork details from external metadata providers"
}

// Scraper returns the underlying Scraper client.
func (s *ScraperComponent) Scraper() Scraper {
	return s.scraper
}

// Config returns the instance configuration.
func (s *ScraperComponent) Config() config.ScraperInstanceConfig {
	return s.cfg
}

// Execute performs metadata scraping for given media info or title parameters.
func (s *ScraperComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if s.scraper == nil {
		return nil, fmt.Errorf("scraper implementation is not initialized for component '%s'", s.id)
	}

	mediaInfo := s.resolveMediaInfo(sCtx)
	if mediaInfo == nil || mediaInfo.Title == "" {
		return nil, fmt.Errorf("no media info or title provided for scraper component '%s'", s.id)
	}

	meta, err := s.scraper.Scrape(ctx, mediaInfo)
	if err != nil {
		return nil, fmt.Errorf("scraper '%s' failed for '%s': %w", s.id, mediaInfo.Title, err)
	}

	sCtx.Log(fmt.Sprintf("Scraped metadata: TMDB_ID=%d, Title='%s' (%d), Type=%s",
		meta.TMDBID, meta.Title, meta.Year, meta.MediaType))

	outputs := map[string]any{
		"metadata":       meta,
		"tmdb_id":        meta.TMDBID,
		"title":          meta.Title,
		"original_title": meta.OriginalTitle,
		"year":           meta.Year,
		"media_type":     string(meta.MediaType),
		"season":         meta.Season,
		"episode":        meta.Episode,
		"poster_url":     meta.PosterURL,
		"backdrop_url":   meta.BackdropURL,
		"plot":           meta.Plot,
	}

	return outputs, nil
}

// resolveMediaInfo extracts MediaInfo from parameters, upstream steps, or event payload.
func (s *ScraperComponent) resolveMediaInfo(sCtx *pipeline.StepContext) *parser.MediaInfo {
	// 1. Direct MediaInfo struct in Params
	if v, ok := sCtx.Params["media_info"]; ok {
		if info, ok := v.(*parser.MediaInfo); ok && info != nil {
			return info
		}
	}

	// 2. Upstream steps output
	if sCtx.Steps != nil {
		for _, stepOutputs := range sCtx.Steps {
			if v, ok := stepOutputs["media_info"]; ok {
				if info, ok := v.(*parser.MediaInfo); ok && info != nil {
					return info
				}
			}
		}
	}

	// 3. Event payload
	if sCtx.Event != nil {
		if v, ok := sCtx.Event["media_info"]; ok {
			if info, ok := v.(*parser.MediaInfo); ok && info != nil {
				return info
			}
		}
	}

	// 4. Construct from individual parameters
	title := sCtx.GetStringParam("title", "")
	if title == "" {
		title = sCtx.GetStringParam("name", "")
	}
	if title == "" && sCtx.Event != nil {
		if t, ok := sCtx.Event["title"].(string); ok {
			title = t
		} else if n, ok := sCtx.Event["name"].(string); ok {
			title = n
		}
	}

	if title == "" {
		return nil
	}

	year := sCtx.GetIntParam("year", 0)
	mediaTypeStr := sCtx.GetStringParam("media_type", "movie")
	season := sCtx.GetIntParam("season", 0)
	episode := sCtx.GetIntParam("episode", 0)

	return &parser.MediaInfo{
		Title:     title,
		Year:      year,
		MediaType: parser.MediaType(mediaTypeStr),
		Season:    season,
		Episode:   episode,
	}
}

// RegisterComponents registers configured scraper instances into the pipeline registry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*ScraperComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}

	scraperConfigs := cfg.GetScraperConfigs()
	var registered []*ScraperComponent

	for _, scCfg := range scraperConfigs {
		if !scCfg.Enabled {
			slog.Debug("Skipping disabled scraper instance", "id", scCfg.ID)
			continue
		}

		sc, err := CreateScraper(scCfg)
		if err != nil {
			slog.Warn("Failed to create scraper client, registering stub component", "id", scCfg.ID, "error", err)
			continue
		}

		comp := NewComponent(scCfg, sc)
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register scraper component '%s' error: %w", comp.ID(), err)
		}

		registered = append(registered, comp)
		slog.Info("Registered scraper pipeline component", "id", comp.ID(), "type", scCfg.Type, "name", comp.Name())
	}

	// Register default alias if "scraper" is not already registered
	if len(registered) > 0 {
		if _, exists := registry.Get("scraper"); !exists {
			aliasComp := NewComponent(registered[0].cfg, registered[0].scraper)
			aliasComp.id = "scraper"
			_ = registry.Register(aliasComp)
			slog.Info("Registered default 'scraper' alias pointing to first instance", "target", registered[0].ID())
		}
	}

	return registered, nil
}
