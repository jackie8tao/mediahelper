package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// ParserComponent adapts a Parser implementation to the pipeline.Component interface.
type ParserComponent struct {
	parser Parser
	cfg    config.ParserComponentConfig
}

func (c *ParserComponent) ID() string                           { return c.cfg.ID }
func (c *ParserComponent) Name() string                         { return c.cfg.Name }
func (c *ParserComponent) Description() string                  { return c.cfg.Description }
func (c *ParserComponent) Parser() Parser                       { return c.parser }
func (c *ParserComponent) Config() config.ParserComponentConfig { return c.cfg }

// Execute handles media title parsing operations within a pipeline step.
func (c *ParserComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if c.parser == nil {
		return nil, fmt.Errorf("parser client is not initialized for component '%s'", c.cfg.ID)
	}
	if sCtx == nil {
		return nil, fmt.Errorf("step context cannot be nil for parser component '%s'", c.cfg.ID)
	}

	rawText := strings.TrimSpace(sCtx.GetStringParam("title", ""))
	if rawText == "" {
		if path := strings.TrimSpace(sCtx.GetStringParam("path", "")); path != "" {
			rawText = filepath.Base(path)
		}
	}
	if rawText == "" {
		return nil, fmt.Errorf("missing required parameter 'title' or 'path' for parser component '%s'", c.cfg.ID)
	}

	info, err := c.parser.Parse(ctx, rawText)
	if err != nil {
		return nil, fmt.Errorf("parser '%s' failed to parse '%s': %w", c.cfg.ID, rawText, err)
	}

	sCtx.Log(fmt.Sprintf("Parsed title '%s' via %s -> Title='%s', Year=%d, Type=%s, S%02dE%02d",
		rawText, c.parser.Name(), info.Title, info.Year, info.MediaType, info.Season, info.Episode))

	outputs := map[string]any{
		"media_info":  info,
		"title":       info.Title,
		"cn_title":    info.CnTitle,
		"en_title":    info.EnTitle,
		"year":        info.Year,
		"media_type":  string(info.MediaType),
		"season":      info.Season,
		"episode":     info.Episode,
		"resolution":  info.Resolution,
		"source":      info.Source,
		"video_codec": info.VideoCodec,
		"audio_codec": info.AudioCodec,
	}

	return outputs, nil
}
