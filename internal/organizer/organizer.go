package organizer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/jackie8tao/mediahelper/internal/scraper/nfo"
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

// Organizer 文件整理引擎
type Organizer struct {
	cfg        config.OrganizerInstanceConfig
	scraperSvc scraper.Scraper
}

// NewOrganizer 创建整理引擎实例 (向下兼容全局配置)
func NewOrganizer(cfg config.Config, scraperSvc scraper.Scraper) *Organizer {
	instCfg := cfg.Library.ToInstanceConfig("organizer")
	instCfg.GenerateNFO = cfg.TMDB.GenerateNFO
	instCfg.DownloadImages = cfg.TMDB.DownloadImages
	return NewOrganizerFromInstance(instCfg, scraperSvc)
}

// NewOrganizerFromInstance 创建特定实例配置的整理引擎
func NewOrganizerFromInstance(cfg config.OrganizerInstanceConfig, scraperSvc scraper.Scraper) *Organizer {
	return &Organizer{
		cfg:        cfg,
		scraperSvc: scraperSvc,
	}
}

// Organize 执行整理流水线
func (o *Organizer) Organize(ctx context.Context, task OrganizeTask) (*OrganizeResult, error) {
	result := &OrganizeResult{}

	// 1. 基础校验
	srcFi, err := os.Stat(task.SourcePath)
	if err != nil {
		result.Error = fmt.Sprintf("stat source path error: %v", err)
		return result, err
	}

	meta := task.Metadata
	if meta == nil {
		// 如果没有刮削到元数据，使用 parser 基础信息作为兜底
		meta = &scraper.MediaMetadata{
			Title:     task.MediaInfo.Title,
			Year:      task.MediaInfo.Year,
			MediaType: task.MediaInfo.MediaType,
			Season:    task.MediaInfo.Season,
			Episode:   task.MediaInfo.Episode,
		}
	}

	// 2. 确定媒体库目标根目录
	baseTargetDir := o.getTargetBaseDir(meta.MediaType)
	if baseTargetDir == "" {
		err := fmt.Errorf("target base directory not configured for media type %s", meta.MediaType)
		result.Error = err.Error()
		return result, err
	}

	// 规范化主标题与年份
	cleanTitle := SanitizeFilename(meta.Title)
	if cleanTitle == "" {
		cleanTitle = "Unknown"
	}
	titleYear := cleanTitle
	if meta.Year > 0 {
		titleYear = fmt.Sprintf("%s (%d)", cleanTitle, meta.Year)
	}

	linkMode := LinkMode(o.cfg.LinkMode)
	if linkMode == "" {
		linkMode = LinkModeHardlink
	}

	// 3. 收集所有待整理的源视频文件
	var videoFiles []string
	if srcFi.IsDir() {
		videoFiles = o.scanVideoFiles(task.SourcePath)
	} else {
		if o.isVideoFile(task.SourcePath) {
			videoFiles = append(videoFiles, task.SourcePath)
		}
	}

	if len(videoFiles) == 0 {
		err := fmt.Errorf("no eligible video files found in %s", task.SourcePath)
		result.Error = err.Error()
		return result, err
	}

	// 4. 根据电影或电视剧进行归档分类
	if meta.MediaType == parser.MediaTypeMovie {
		// === 电影整理 ===
		movieDir := filepath.Join(baseTargetDir, titleYear)
		result.TargetDir = movieDir

		// 电影通常只处理最大的主视频文件或首个视频
		mainVideo := o.findMainVideoFile(videoFiles)
		ext := filepath.Ext(mainVideo)

		targetFileName := titleYear
		if task.MediaInfo != nil && task.MediaInfo.Resolution != "" {
			targetFileName = fmt.Sprintf("%s - %s%s", titleYear, task.MediaInfo.Resolution, ext)
		} else {
			targetFileName = fmt.Sprintf("%s%s", titleYear, ext)
		}

		targetFilePath := filepath.Join(movieDir, targetFileName)

		// 执行主视频链接
		if err := TransferFile(mainVideo, targetFilePath, linkMode); err != nil {
			result.Error = fmt.Sprintf("transfer movie file error: %v", err)
			return result, err
		}
		result.LinkedFiles = append(result.LinkedFiles, FilePair{Source: mainVideo, Target: targetFilePath})

		// 伴生字幕处理
		o.transferSubtitles(mainVideo, targetFilePath, linkMode, result)

		// 写入 movie.nfo
		if o.cfg.GenerateNFO {
			nfoPath := filepath.Join(movieDir, "movie.nfo")
			if nfoData, err := nfo.GenerateMovieNFO(meta); err == nil {
				_ = os.WriteFile(nfoPath, nfoData, 0644)
				result.NFOPath = nfoPath
			}
		}

		// 下载海报与背景图
		o.downloadImages(ctx, meta, movieDir, result)

	} else {
		// === 电视剧 / 动漫整理 ===
		showDir := filepath.Join(baseTargetDir, titleYear)
		result.TargetDir = showDir

		for _, vFile := range videoFiles {
			// 解析单文件对应的季集信息
			subInfo := parser.Parse(filepath.Base(vFile))
			season := subInfo.Season
			if season == 0 {
				if meta.Season > 0 {
					season = meta.Season
				} else {
					season = 1
				}
			}

			episode := subInfo.Episode
			if episode == 0 && meta.Episode > 0 {
				episode = meta.Episode
			}

			seasonDirName := fmt.Sprintf("Season %02d", season)
			seasonDir := filepath.Join(showDir, seasonDirName)

			ext := filepath.Ext(vFile)
			var episodeFileName string
			if meta.EpisodeTitle != "" && episode == meta.Episode {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d - %s%s", cleanTitle, season, episode, SanitizeFilename(meta.EpisodeTitle), ext)
			} else {
				episodeFileName = fmt.Sprintf("%s - S%02dE%02d%s", cleanTitle, season, episode, ext)
			}

			targetFilePath := filepath.Join(seasonDir, episodeFileName)

			// 执行单集视频链接
			if err := TransferFile(vFile, targetFilePath, linkMode); err != nil {
				slog.Error("transfer episode file failed", "src", vFile, "dst", targetFilePath, "error", err)
				continue
			}
			result.LinkedFiles = append(result.LinkedFiles, FilePair{Source: vFile, Target: targetFilePath})

			// 伴生字幕处理
			o.transferSubtitles(vFile, targetFilePath, linkMode, result)

			// 生成单集 NFO
			if o.cfg.GenerateNFO && episode > 0 {
				epNFOPath := strings.TrimSuffix(targetFilePath, ext) + ".nfo"
				epMeta := *meta
				epMeta.Season = season
				epMeta.Episode = episode
				if nfoData, err := nfo.GenerateEpisodeNFO(&epMeta); err == nil {
					_ = os.WriteFile(epNFOPath, nfoData, 0644)
				}
			}
		}

		// 写入剧集主 tvshow.nfo
		if o.cfg.GenerateNFO {
			tvNFOPath := filepath.Join(showDir, "tvshow.nfo")
			if nfoData, err := nfo.GenerateTVShowNFO(meta); err == nil {
				_ = os.WriteFile(tvNFOPath, nfoData, 0644)
				result.NFOPath = tvNFOPath
			}
		}

		// 下载剧集海报与背景图
		o.downloadImages(ctx, meta, showDir, result)
	}

	result.Success = len(result.LinkedFiles) > 0
	return result, nil
}

// getTargetBaseDir 根据媒体类型返回库的根路径
func (o *Organizer) getTargetBaseDir(mType parser.MediaType) string {
	switch mType {
	case parser.MediaTypeMovie:
		return o.cfg.MovieDir
	case parser.MediaTypeAnime:
		if o.cfg.AnimeDir != "" {
			return o.cfg.AnimeDir
		}
		return o.cfg.TVDir
	default:
		return o.cfg.TVDir
	}
}

// isVideoFile 检查是否为有效视频文件且满足体积限制
func (o *Organizer) isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if !videoExts[ext] {
		return false
	}

	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	minSize := o.cfg.MinFileSize
	if minSize <= 0 {
		minSize = 10 * 1024 * 1024 // 默认至少 10MB
	}

	return fi.Size() >= minSize
}

// scanVideoFiles 递归扫描目录下的所有有效视频文件
func (o *Organizer) scanVideoFiles(dir string) []string {
	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if o.isVideoFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// findMainVideoFile 获取列表中体积最大的主文件 (常用于电影多文件场景)
func (o *Organizer) findMainVideoFile(files []string) string {
	if len(files) == 0 {
		return ""
	}
	var largestFile string
	var maxSize int64 = -1

	for _, f := range files {
		fi, err := os.Stat(f)
		if err == nil && fi.Size() > maxSize {
			maxSize = fi.Size()
			largestFile = f
		}
	}
	return largestFile
}

// transferSubtitles 查找与源视频同目录的同名字幕并一并链接
func (o *Organizer) transferSubtitles(srcVideo, dstVideo string, mode LinkMode, result *OrganizeResult) {
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
			// 提取字幕语言后缀，如 .zh.srt 或 .srt
			subSuffix := strings.TrimPrefix(name, srcBase)
			targetSub := dstBaseWithoutExt + subSuffix
			srcSubPath := filepath.Join(srcDir, name)

			if err := TransferFile(srcSubPath, targetSub, mode); err == nil {
				result.LinkedFiles = append(result.LinkedFiles, FilePair{
					Source: srcSubPath,
					Target: targetSub,
				})
			}
		}
	}
}

// downloadImages 下载海报与背景图
func (o *Organizer) downloadImages(ctx context.Context, meta *scraper.MediaMetadata, targetDir string, result *OrganizeResult) {
	if !o.cfg.DownloadImages || o.scraperSvc == nil {
		return
	}

	// 1. 海报 (poster.jpg)
	if meta.PosterURL != "" {
		posterPath := filepath.Join(targetDir, "poster.jpg")
		if _, err := os.Stat(posterPath); os.IsNotExist(err) {
			if err := o.scraperSvc.DownloadImage(ctx, meta.PosterURL, posterPath); err == nil {
				result.PosterPath = posterPath
			}
		}
	}

	// 2. 背景图 (fanart.jpg)
	if meta.BackdropURL != "" {
		fanartPath := filepath.Join(targetDir, "fanart.jpg")
		if _, err := os.Stat(fanartPath); os.IsNotExist(err) {
			if err := o.scraperSvc.DownloadImage(ctx, meta.BackdropURL, fanartPath); err == nil {
				result.FanartPath = fanartPath
			}
		}
	}
}
