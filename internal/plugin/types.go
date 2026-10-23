package plugin

import (
	"context"

	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// Plugin 插件生命周期统一接口
type Plugin interface {
	// ID 返回插件唯一标识符 (例如: "jellyfin_refresh", "webhook_notify")
	ID() string
	// Name 返回插件显示名称
	Name() string
	// Version 返回插件版本号
	Version() string
	// Description 返回插件功能描述
	Description() string

	// Init 初始化插件并注入专属配置
	Init(ctx context.Context, config map[string]interface{}) error
	// Start 启动插件后台任务或监听 (无需后台任务可直接返回 nil)
	Start(ctx context.Context) error
	// Stop 优雅关停插件
	Stop(ctx context.Context) error
}

// TorrentEvent 种子任务事件
type TorrentEvent struct {
	Torrent downloader.Torrent
}

// ScrapeEvent 元数据刮削完成事件
type ScrapeEvent struct {
	MediaInfo *parser.MediaInfo
	Metadata  *scraper.MediaMetadata
}

// OrganizeEvent 整理归档完成事件
type OrganizeEvent struct {
	Torrent   downloader.Torrent
	MediaInfo *parser.MediaInfo
	Metadata  *scraper.MediaMetadata
	Result    *organizer.OrganizeResult
}

// TorrentCompletedHook 种子下载完成事件钩子
type TorrentCompletedHook interface {
	OnTorrentCompleted(ctx context.Context, event TorrentEvent) error
}

// PostScrapeHook 元数据刮削完成后的扩展钩子
type PostScrapeHook interface {
	OnPostScrape(ctx context.Context, event ScrapeEvent) error
}

// PostOrganizeHook 媒体文件硬链接整理完成后的扩展钩子 (最常用: 刷新媒体库/发送推送)
type PostOrganizeHook interface {
	OnPostOrganize(ctx context.Context, event OrganizeEvent) error
}

// Info 插件展示信息 (供 WebUI 或 API 查询)
type Info struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Hooks       []string `json:"hooks"`
}
