package parser

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".ts": true,
	".mov": true, ".wmv": true, ".m2ts": true, ".flv": true,
}

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

// Execute handles media title and multi-file directory parsing operations within a pipeline step.
func (c *ParserComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if c.parser == nil {
		return nil, fmt.Errorf("parser client is not initialized for component '%s'", c.cfg.ID)
	}
	if sCtx == nil {
		return nil, fmt.Errorf("step context cannot be nil for parser component '%s'", c.cfg.ID)
	}

	var task ParseTask
	if err := sCtx.BindParams(&task); err != nil {
		return nil, fmt.Errorf("bind parser params failed for '%s': %w", c.cfg.ID, err)
	}

	rawText := strings.TrimSpace(task.RawTitle)
	path := strings.TrimSpace(task.Path)
	files := task.Files

	var isDirPath bool
	if path != "" {
		if rawText == "" {
			rawText = filepath.Base(path)
		}
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			isDirPath = true
			if len(files) == 0 {
				files, _ = scanRelativeVideoFiles(path)
			}
		}
	}

	if rawText == "" {
		return nil, fmt.Errorf("missing required parameter 'raw_title' or 'path' for parser component '%s'", c.cfg.ID)
	}

	var (
		info *MediaInfo
		err  error
	)
	if len(files) > 0 {
		info, err = c.parser.ParseFiles(ctx, rawText, files)
		if err == nil && isDirPath {
			for i := range info.Files {
				if info.Files[i].FilePath != "" && !filepath.IsAbs(info.Files[i].FilePath) {
					info.Files[i].FilePath = filepath.Join(path, filepath.FromSlash(info.Files[i].FilePath))
				}
			}
		}
	} else {
		info, err = c.parser.Parse(ctx, rawText)
		if err == nil && path != "" && !isDirPath {
			if len(info.Files) > 0 {
				info.Files[0].FilePath = path
			} else {
				info.Files = []FileItem{
					{
						FilePath: path,
					},
				}
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("parser '%s' failed to parse '%s': %w", c.cfg.ID, rawText, err)
	}

	sCtx.Logf("Parsed '%s' via %s -> Title='%s', Year=%d, Type=%s, Files=%d",
		rawText, c.parser.Name(), info.Title, info.Year, info.MediaType, len(info.Files))

	outputs := map[string]any{
		"media_info":  info,
		"title":       info.Title,
		"cn_title":    info.CnTitle,
		"en_title":    info.EnTitle,
		"year":        info.Year,
		"media_type":  string(info.MediaType),
		"resolution":  info.Resolution,
		"source":      info.Source,
		"video_codec": info.VideoCodec,
		"audio_codec": info.AudioCodec,
		"files":       info.Files,
	}

	return outputs, nil
}

// scanRelativeVideoFiles walks a directory and returns relative paths of all video files inside it.
func scanRelativeVideoFiles(rootDir string) ([]string, error) {
	var relFiles []string
	err := filepath.Walk(rootDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !videoExtensions[ext] {
			return nil
		}
		rel, relErr := filepath.Rel(rootDir, p)
		if relErr != nil {
			rel = info.Name()
		}
		relFiles = append(relFiles, filepath.ToSlash(rel))
		return nil
	})
	return relFiles, err
}

// RegisterComponents instantiates parser components according to system configuration
// and registers each enabled instance into the provided ComponentRegistry.
func RegisterComponents(cfg *config.ComponentConfig, registry *pipeline.ComponentRegistry) ([]*ParserComponent, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configs := cfg.GetParserConfigs()
	if len(configs) == 0 {
		slog.Debug("No parser instances configured")
		return nil, nil
	}

	var registered []*ParserComponent

	for _, pCfg := range configs {
		if !pCfg.Enabled {
			slog.Debug("Parser instance is disabled, skipping registration", "id", pCfg.ID)
			continue
		}

		p, err := CreateParser(pCfg)
		if err != nil {
			return nil, fmt.Errorf("create parser component '%s' error: %w", pCfg.ID, err)
		}
		comp := &ParserComponent{
			parser: p,
			cfg:    pCfg,
		}
		if err := registry.Register(comp); err != nil {
			return nil, fmt.Errorf("register parser component '%s' error: %w", pCfg.ID, err)
		}

		slog.Info("Registered parser pipeline component", "id", pCfg.ID, "type", pCfg.Type, "name", pCfg.Name)
		registered = append(registered, comp)
	}

	return registered, nil
}
