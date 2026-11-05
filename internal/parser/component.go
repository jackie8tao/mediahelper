package parser

import (
	"context"
	"fmt"
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

	rawText := strings.TrimSpace(sCtx.GetStringParam("title", ""))
	path := strings.TrimSpace(sCtx.GetStringParam("path", ""))
	files := sCtx.GetStringSliceParam("files")

	var isDirPath bool
	if path != "" {
		if rawText == "" {
			rawText = filepath.Base(path)
		}
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			isDirPath = true
			if len(files) == 0 {
				files, _ = ScanRelativeVideoFiles(path)
			}
		}
	}

	if rawText == "" {
		return nil, fmt.Errorf("missing required parameter 'title' or 'path' for parser component '%s'", c.cfg.ID)
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
			info.Files = []FileItem{
				{
					FilePath: path,
					Season:   info.Season,
					Episode:  info.Episode,
				},
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("parser '%s' failed to parse '%s': %w", c.cfg.ID, rawText, err)
	}

	sCtx.Logf("Parsed '%s' via %s -> Title='%s', Year=%d, Type=%s, S%02dE%02d, Files=%d",
		rawText, c.parser.Name(), info.Title, info.Year, info.MediaType, info.Season, info.Episode, len(info.Files))

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
		"files":       info.Files,
	}

	return outputs, nil
}

// ScanRelativeVideoFiles walks a directory and returns relative paths of all video files inside it.
func ScanRelativeVideoFiles(rootDir string) ([]string, error) {
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
