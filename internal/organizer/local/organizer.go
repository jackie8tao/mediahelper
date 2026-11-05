package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/organizer"
)

var (
	videoExts = map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".ts": true,
		".mov": true, ".wmv": true, ".m2ts": true, ".flv": true,
	}
	subtitleExts = map[string]bool{
		".srt": true, ".ass": true, ".ssa": true, ".vtt": true, ".sub": true,
	}
)

// Config holds configuration parameters for the local filesystem organizer.
type Config struct {
	MovieDir    string             `json:"movie_dir"`
	TVDir       string             `json:"tv_dir"`
	AnimeDir    string             `json:"anime_dir"`
	LinkMode    organizer.LinkMode `json:"link_mode"`
	MinFileSize int64              `json:"min_file_size"`
}

// Organizer implements organizer.Organizer for local filesystem media libraries.
type Organizer struct {
	cfg Config
}

func init() {
	organizer.RegisterOrganizerFactory("local", LocalFactory)
}

// LocalFactory initializes a new local filesystem organizer instance.
func LocalFactory(cfgVal json.RawMessage) (organizer.Organizer, error) {
	var cfgSt Config
	if len(cfgVal) > 0 {
		if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
			return nil, fmt.Errorf("unmarshal local organizer config error: %w", err)
		}
	}

	if cfgSt.LinkMode == "" {
		cfgSt.LinkMode = organizer.LinkModeHardlink
	}

	return &Organizer{
		cfg: cfgSt,
	}, nil
}

// Name returns the organizer provider name.
func (o *Organizer) Name() string {
	return "LocalOrganizer"
}

// Organize transfers or links media files specified in task.Files into the target library directory.
func (o *Organizer) Organize(ctx context.Context, task organizer.OrganizeTask) (*organizer.OrganizeResult, error) {
	result := &organizer.OrganizeResult{}

	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if len(task.Files) == 0 {
		err := fmt.Errorf("files list cannot be empty")
		result.Error = err.Error()
		return result, err
	}

	title := strings.TrimSpace(task.Title)
	if title == "" {
		err := fmt.Errorf("title cannot be empty")
		result.Error = err.Error()
		return result, err
	}

	cleanTitle := sanitizeFilename(title)
	if cleanTitle == "" {
		err := fmt.Errorf("title contains only invalid characters: %q", title)
		result.Error = err.Error()
		return result, err
	}

	titleYear := cleanTitle
	if task.Year > 0 {
		titleYear = fmt.Sprintf("%s (%d)", cleanTitle, task.Year)
	}

	mediaType := strings.ToLower(strings.TrimSpace(task.MediaType))
	if mediaType == "" {
		mediaType = "movie"
	}

	baseTargetDir := o.getTargetBaseDir(mediaType)
	if baseTargetDir == "" {
		err := fmt.Errorf("target base directory is not configured for media type '%s'", mediaType)
		result.Error = err.Error()
		return result, err
	}

	linkMode := o.cfg.LinkMode
	if linkMode == "" {
		linkMode = organizer.LinkModeHardlink
	}

	targetDir := filepath.Join(baseTargetDir, titleYear)
	result.TargetDir = targetDir

	for _, item := range task.Files {
		srcPath := strings.TrimSpace(item.FilePath)
		if srcPath == "" || !o.isVideoFile(srcPath) {
			continue
		}

		ext := filepath.Ext(srcPath)
		var targetFilePath string

		if mediaType == "movie" {
			targetFileName := titleYear
			if resTag := sanitizeFilename(task.Resolution); resTag != "" {
				targetFileName = fmt.Sprintf("%s - %s", titleYear, resTag)
			}
			targetFilePath = filepath.Join(targetDir, targetFileName+ext)
		} else {
			if item.Season <= 0 {
				err := fmt.Errorf("invalid season %d for file %q in media type %q", item.Season, srcPath, mediaType)
				result.Error = err.Error()
				return result, err
			}
			if item.Episode <= 0 {
				err := fmt.Errorf("invalid episode %d for file %q in media type %q", item.Episode, srcPath, mediaType)
				result.Error = err.Error()
				return result, err
			}

			seasonDir := filepath.Join(targetDir, fmt.Sprintf("Season %02d", item.Season))
			var episodeFileName string
			if epTitle := sanitizeFilename(item.EpisodeTitle); epTitle != "" {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d - %s%s", cleanTitle, item.Season, item.Episode, epTitle, ext)
			} else {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d%s", cleanTitle, item.Season, item.Episode, ext)
			}
			targetFilePath = filepath.Join(seasonDir, episodeFileName)
		}

		if err := transferFile(srcPath, targetFilePath, linkMode); err != nil {
			slog.Error("Transfer media file failed", "src", srcPath, "dst", targetFilePath, "error", err)
			continue
		}

		result.LinkedFiles = append(result.LinkedFiles, organizer.FilePair{
			Source: srcPath,
			Target: targetFilePath,
		})

		o.transferSubtitles(srcPath, targetFilePath, linkMode, result)
	}

	if len(result.LinkedFiles) == 0 {
		err := fmt.Errorf("no eligible files were organized for task '%s'", title)
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	return result, nil
}

func (o *Organizer) getTargetBaseDir(mediaType string) string {
	switch mediaType {
	case "movie":
		return o.cfg.MovieDir
	case "anime":
		if o.cfg.AnimeDir != "" {
			return o.cfg.AnimeDir
		}
		return o.cfg.TVDir
	default:
		return o.cfg.TVDir
	}
}

func (o *Organizer) isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if !videoExts[ext] {
		return false
	}

	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}

	if o.cfg.MinFileSize > 0 {
		return fi.Size() >= o.cfg.MinFileSize
	}
	return true
}

func (o *Organizer) transferSubtitles(srcVideo, dstVideo string, mode organizer.LinkMode, result *organizer.OrganizeResult) {
	srcDir := filepath.Dir(srcVideo)
	srcBase := strings.TrimSuffix(filepath.Base(srcVideo), filepath.Ext(srcVideo))
	dstBaseWithoutExt := strings.TrimSuffix(dstVideo, filepath.Ext(dstVideo))

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		subExt := strings.ToLower(filepath.Ext(name))
		if !subtitleExts[subExt] {
			continue
		}

		if strings.HasPrefix(name, srcBase) {
			subSuffix := strings.TrimPrefix(name, srcBase)
			targetSub := dstBaseWithoutExt + subSuffix
			srcSubPath := filepath.Join(srcDir, name)

			if err := transferFile(srcSubPath, targetSub, mode); err == nil {
				result.LinkedFiles = append(result.LinkedFiles, organizer.FilePair{
					Source: srcSubPath,
					Target: targetSub,
				})
			}
		}
	}
}

func transferFile(src, dst string, mode organizer.LinkMode) error {
	targetDir := filepath.Dir(dst)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target dir error: %w", err)
	}

	if dstFi, err := os.Stat(dst); err == nil {
		srcFi, err := os.Stat(src)
		if err == nil && os.SameFile(srcFi, dstFi) {
			return nil
		}
		_ = os.Remove(dst)
	}

	switch mode {
	case organizer.LinkModeHardlink:
		if err := os.Link(src, dst); err != nil {
			slog.Warn("Hardlink failed (possible cross-device link), falling back to copy", "src", src, "dst", dst, "error", err)
			return copyFile(src, dst)
		}
		return nil

	case organizer.LinkModeSymlink:
		return os.Symlink(src, dst)

	case organizer.LinkModeMove:
		if err := os.Rename(src, dst); err != nil {
			if copyErr := copyFile(src, dst); copyErr != nil {
				return copyErr
			}
			return os.Remove(src)
		}
		return nil

	case organizer.LinkModeCopy:
		return copyFile(src, dst)

	default:
		return fmt.Errorf("unsupported link mode: %s", mode)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file error: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file error: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file stream error: %w", err)
	}

	return out.Sync()
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	name = strings.ReplaceAll(name, ":", " - ")

	illegalChars := []string{"<", ">", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range illegalChars {
		name = strings.ReplaceAll(name, char, "")
	}

	fields := strings.Fields(name)
	name = strings.Join(fields, " ")
	name = strings.TrimRight(name, ". ")

	return name
}
