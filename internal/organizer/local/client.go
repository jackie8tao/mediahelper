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

// Client implements organizer.Organizer for local filesystem media libraries.
type Client struct {
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

	return &Client{
		cfg: cfgSt,
	}, nil
}

// Name returns the organizer provider name.
func (c *Client) Name() string {
	return "LocalOrganizer"
}

// Organize transfers or links media files specified in task.Files into the target library directory.
func (c *Client) Organize(ctx context.Context, task organizer.OrganizeTask) (*organizer.OrganizeResult, error) {
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

	// Filter and validate eligible video files from task.Files
	var validItems []organizer.EpisodeFileItem
	for _, item := range task.Files {
		filePath := strings.TrimSpace(item.File)
		if filePath == "" {
			continue
		}
		if c.isVideoFile(filePath) {
			item.File = filePath
			validItems = append(validItems, item)
		}
	}

	if len(validItems) == 0 {
		err := fmt.Errorf("no eligible video files found in task files")
		result.Error = err.Error()
		return result, err
	}

	mediaType := strings.ToLower(strings.TrimSpace(task.MediaType))
	if mediaType == "" {
		mediaType = "movie"
	}

	baseTargetDir := c.getTargetBaseDir(mediaType)
	if baseTargetDir == "" {
		err := fmt.Errorf("target base directory is not configured for media type '%s'", mediaType)
		result.Error = err.Error()
		return result, err
	}

	title := strings.TrimSpace(task.Title)
	if title == "" {
		baseName := filepath.Base(validItems[0].File)
		title = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	cleanTitle := sanitizeFilename(title)
	if cleanTitle == "" {
		cleanTitle = "Unknown"
	}

	titleYear := cleanTitle
	if task.Year > 0 {
		titleYear = fmt.Sprintf("%s (%d)", cleanTitle, task.Year)
	}

	linkMode := c.cfg.LinkMode
	if linkMode == "" {
		linkMode = organizer.LinkModeHardlink
	}

	if mediaType == "movie" {
		movieDir := filepath.Join(baseTargetDir, titleYear)
		result.TargetDir = movieDir

		mainVideo := c.findMainVideoFile(validItems)
		ext := filepath.Ext(mainVideo)

		targetFileName := titleYear
		if resTag := sanitizeFilename(task.Resolution); resTag != "" {
			targetFileName = fmt.Sprintf("%s - %s", titleYear, resTag)
		}
		targetFilePath := filepath.Join(movieDir, targetFileName+ext)

		if err := transferFile(mainVideo, targetFilePath, linkMode); err != nil {
			result.Error = fmt.Sprintf("transfer movie file failed: %v", err)
			return result, err
		}
		result.LinkedFiles = append(result.LinkedFiles, organizer.FilePair{
			Source: mainVideo,
			Target: targetFilePath,
		})

		c.transferSubtitles(mainVideo, targetFilePath, linkMode, result)
	} else {
		showDir := filepath.Join(baseTargetDir, titleYear)
		result.TargetDir = showDir

		for _, item := range validItems {
			season := item.Season
			if season <= 0 {
				season = 1
			}
			episode := item.Episode
			if episode <= 0 {
				if len(validItems) == 1 {
					episode = 1
				} else {
					slog.Warn("Skipping episode file with missing episode number", "file", item.File)
					continue
				}
			}

			seasonDir := filepath.Join(showDir, fmt.Sprintf("Season %02d", season))
			ext := filepath.Ext(item.File)

			var episodeFileName string
			if epTitle := sanitizeFilename(item.EpisodeTitle); epTitle != "" {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d - %s%s", cleanTitle, season, episode, epTitle, ext)
			} else {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d%s", cleanTitle, season, episode, ext)
			}

			targetFilePath := filepath.Join(seasonDir, episodeFileName)

			if err := transferFile(item.File, targetFilePath, linkMode); err != nil {
				slog.Error("Transfer episode file failed", "src", item.File, "dst", targetFilePath, "error", err)
				continue
			}
			result.LinkedFiles = append(result.LinkedFiles, organizer.FilePair{
				Source: item.File,
				Target: targetFilePath,
			})

			c.transferSubtitles(item.File, targetFilePath, linkMode, result)
		}
	}

	result.Success = len(result.LinkedFiles) > 0
	return result, nil
}

func (c *Client) getTargetBaseDir(mediaType string) string {
	switch mediaType {
	case "movie":
		return c.cfg.MovieDir
	case "anime":
		if c.cfg.AnimeDir != "" {
			return c.cfg.AnimeDir
		}
		return c.cfg.TVDir
	default:
		return c.cfg.TVDir
	}
}

func (c *Client) isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if !videoExts[ext] {
		return false
	}

	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}

	if c.cfg.MinFileSize > 0 {
		return fi.Size() >= c.cfg.MinFileSize
	}
	return true
}

func (c *Client) findMainVideoFile(items []organizer.EpisodeFileItem) string {
	if len(items) == 0 {
		return ""
	}
	var largestFile string
	var maxSize int64 = -1

	for _, item := range items {
		fi, err := os.Stat(item.File)
		if err == nil && fi.Size() > maxSize {
			maxSize = fi.Size()
			largestFile = item.File
		}
	}
	return largestFile
}

func (c *Client) transferSubtitles(srcVideo, dstVideo string, mode organizer.LinkMode, result *organizer.OrganizeResult) {
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
