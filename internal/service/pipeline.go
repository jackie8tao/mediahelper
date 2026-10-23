package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// Pipeline 影视自动化全流程串联流水线
type Pipeline struct {
	cfg              *config.Config
	downloader       downloader.Downloader
	scraperSvc       scraper.Scraper
	organizer        *organizer.Organizer
	pluginMgr        *plugin.Manager
	history          *HistoryManager
	processingHashes sync.Map // 正在处理的种子 Hash，防止并发重复整理
}

// NewPipeline 初始化流水线服务
func NewPipeline(
	cfg *config.Config,
	dl downloader.Downloader,
	sc scraper.Scraper,
	org *organizer.Organizer,
	pm *plugin.Manager,
) *Pipeline {
	return &Pipeline{
		cfg:        cfg,
		downloader: dl,
		scraperSvc: sc,
		organizer:  org,
		pluginMgr:  pm,
		history:    NewHistoryManager(200),
	}
}

// History 获取历史管理器
func (p *Pipeline) History() *HistoryManager {
	return p.history
}

// PluginManager 获取插件管理器
func (p *Pipeline) PluginManager() *plugin.Manager {
	return p.pluginMgr
}

// Downloader 获取当前下载器客户端
func (p *Pipeline) Downloader() downloader.Downloader {
	return p.downloader
}

// ProcessTorrent 处理单个种子任务 (核心流程: 去重 -> 解析 -> 刮削 -> 整理 -> 回写标签)
func (p *Pipeline) ProcessTorrent(ctx context.Context, t downloader.Torrent) (*organizer.OrganizeResult, error) {
	// 1. 标签去重检查 (若已经打上完成标签，直接跳过)
	successTag := p.cfg.QBittorrent.SuccessTag
	if successTag == "" {
		successTag = "MediaHelper:Done"
	}
	for _, tag := range t.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), successTag) {
			slog.Debug("Torrent already tagged as done, skipping", "hash", t.Hash, "name", t.Name)
			return &organizer.OrganizeResult{Success: true}, nil
		}
	}

	// 2. 内存锁并发防重
	if _, loaded := p.processingHashes.LoadOrStore(t.Hash, struct{}{}); loaded {
		slog.Warn("Torrent is currently being processed by another worker", "hash", t.Hash, "name", t.Name)
		return nil, fmt.Errorf("torrent %s is currently being processed", t.Hash)
	}
	defer p.processingHashes.Delete(t.Hash)

	slog.Info("Starting processing torrent", "hash", t.Hash, "name", t.Name, "category", t.Category)

	// 插件钩子: 种子任务前置事件
	if p.pluginMgr != nil {
		p.pluginMgr.EmitTorrentCompleted(ctx, plugin.TorrentEvent{Torrent: t})
	}

	// 3. 智能解析标题与季集规格
	mediaInfo := parser.Parse(t.Name)

	// 分类辅助校准 (例如下载分类明确为 movie 或 tv)
	cat := strings.ToLower(t.Category)
	if cat == "movie" || cat == "movies" || cat == "电影" {
		mediaInfo.MediaType = parser.MediaTypeMovie
	} else if cat == "tv" || cat == "teleplay" || cat == "电视剧" {
		if mediaInfo.MediaType == parser.MediaTypeMovie {
			mediaInfo.MediaType = parser.MediaTypeTV
			if mediaInfo.Season == 0 {
				mediaInfo.Season = 1
			}
		}
	} else if cat == "anime" || cat == "动漫" || cat == "动画" {
		mediaInfo.MediaType = parser.MediaTypeAnime
	}

	// 4. TMDB 刮削元数据
	var meta *scraper.MediaMetadata
	if p.scraperSvc != nil && p.cfg.TMDB.APIKey != "" {
		scrapedMeta, err := p.scraperSvc.Scrape(ctx, mediaInfo)
		if err != nil {
			slog.Warn("Metadata scraping failed, using parsed fallback info", "name", t.Name, "error", err)
		} else {
			meta = scrapedMeta
		}
	}

	if meta == nil {
		// 未开启刮削或刮削失败时的默认兜底元数据
		meta = &scraper.MediaMetadata{
			Title:     mediaInfo.Title,
			Year:      mediaInfo.Year,
			MediaType: mediaInfo.MediaType,
			Season:    mediaInfo.Season,
			Episode:   mediaInfo.Episode,
		}
	}

	// 插件钩子: 刮削后置事件 (可供插件进行豆瓣补全或分级修正)
	if p.pluginMgr != nil {
		p.pluginMgr.EmitPostScrape(ctx, plugin.ScrapeEvent{
			MediaInfo: mediaInfo,
			Metadata:  meta,
		})
	}

	// 5. 整理并归档到媒体库
	organizeTask := organizer.OrganizeTask{
		SourcePath: t.ContentPath,
		MediaInfo:  mediaInfo,
		Metadata:   meta,
	}

	result, err := p.organizer.Organize(ctx, organizeTask)

	// 6. 状态回写与历史记录
	rec := HistoryRecord{
		Hash:        t.Hash,
		TorrentName: t.Name,
		Title:       meta.Title,
		Year:        meta.Year,
		MediaType:   string(meta.MediaType),
	}

	if err != nil || !result.Success {
		failTag := p.cfg.QBittorrent.FailTag
		if failTag == "" {
			failTag = "MediaHelper:Error"
		}
		errMsg := "unknown error"
		if err != nil {
			errMsg = err.Error()
		} else if result.Error != "" {
			errMsg = result.Error
		}

		slog.Error("Organize torrent failed", "hash", t.Hash, "name", t.Name, "error", errMsg)

		if p.downloader != nil && p.cfg.QBittorrent.AutoTag && t.Hash != "" {
			_ = p.downloader.AddTags(ctx, []string{t.Hash}, []string{failTag})
		}

		rec.Status = "failed"
		rec.Error = errMsg
		p.history.Add(rec)
		return result, err
	}

	// 整理成功处理
	slog.Info("Successfully organized torrent", "hash", t.Hash, "title", meta.Title, "targetDir", result.TargetDir, "files", len(result.LinkedFiles))

	if p.downloader != nil && p.cfg.QBittorrent.AutoTag && t.Hash != "" {
		_ = p.downloader.AddTags(ctx, []string{t.Hash}, []string{successTag})
		if p.cfg.QBittorrent.FailTag != "" {
			_ = p.downloader.RemoveTags(ctx, []string{t.Hash}, []string{p.cfg.QBittorrent.FailTag})
		}
	}

	rec.Status = "success"
	rec.TargetDir = result.TargetDir
	rec.FileCount = len(result.LinkedFiles)
	p.history.Add(rec)

	// 插件钩子: 归档成功后置事件 (触发媒体库刷新、消息推送卡片、字幕下载等)
	if p.pluginMgr != nil {
		p.pluginMgr.EmitPostOrganize(ctx, plugin.OrganizeEvent{
			Torrent:   t,
			MediaInfo: mediaInfo,
			Metadata:  meta,
			Result:    result,
		})
	}

	return result, nil
}

// SyncOnce 执行一次完整的已完成种子扫描与整理
func (p *Pipeline) SyncOnce(ctx context.Context) (int, error) {
	if p.downloader == nil {
		return 0, fmt.Errorf("downloader client is not initialized")
	}

	torrents, err := p.downloader.GetCompletedTorrents(ctx, p.cfg.QBittorrent.Categories)
	if err != nil {
		return 0, fmt.Errorf("fetch completed torrents error: %w", err)
	}

	successTag := p.cfg.QBittorrent.SuccessTag
	if successTag == "" {
		successTag = "MediaHelper:Done"
	}

	processedCount := 0
	for _, t := range torrents {
		// 跳过已打标的完成任务
		alreadyDone := false
		for _, tag := range t.Tags {
			if strings.EqualFold(strings.TrimSpace(tag), successTag) {
				alreadyDone = true
				break
			}
		}
		if alreadyDone {
			continue
		}

		res, err := p.ProcessTorrent(ctx, t)
		if err == nil && res != nil && res.Success {
			processedCount++
		}
	}

	return processedCount, nil
}

// StartBackgroundWorker 启动后台周期轮询 Worker
func (p *Pipeline) StartBackgroundWorker(ctx context.Context) {
	interval := p.cfg.QBittorrent.PollInterval
	if interval <= 0 {
		interval = 30
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	slog.Info("Started background sync worker", "pollInterval", fmt.Sprintf("%ds", interval))

	// 启动时先执行一次初次扫描
	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		count, err := p.SyncOnce(syncCtx)
		if err != nil {
			slog.Warn("Initial background sync failed", "error", err)
		} else if count > 0 {
			slog.Info("Initial background sync finished", "processed", count)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Background sync worker stopped")
			return
		case <-ticker.C:
			syncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			count, err := p.SyncOnce(syncCtx)
			cancel()
			if err != nil {
				slog.Warn("Scheduled background sync failed", "error", err)
			} else if count > 0 {
				slog.Info("Scheduled background sync finished", "processed", count)
			}
		}
	}
}
