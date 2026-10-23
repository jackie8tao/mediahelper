package downloader

import (
	"context"
)

// Torrent 通用种子任务信息
type Torrent struct {
	Hash        string   `json:"hash"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	SavePath    string   `json:"save_path"`    // 实际下载路径 (已映射为本地路径)
	ContentPath string   `json:"content_path"` // 内容根路径 (单文件或顶层目录，已映射为本地路径)
	Progress    float64  `json:"progress"`     // 下载进度 (0.0 - 1.0)
	State       string   `json:"state"`        // 状态 (如 pausedUP, uploading, downloading 等)
	Size        int64    `json:"size"`         // 总大小 (字节)
	IsCompleted bool     `json:"is_completed"` // 是否已下载完成 (进度 100%)
	DlSpeed     int64    `json:"dl_speed"`     // 下载速度 (B/s)
	UpSpeed     int64    `json:"up_speed"`     // 上传速度 (B/s)
	ETA         int64    `json:"eta"`          // 预计剩余时间 (秒)
	AddedOn     int64    `json:"added_on"`     // 添加时间戳
}

// TorrentFile 种子包含的单个文件信息
type TorrentFile struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`     // 相对路径文件名
	Size     int64   `json:"size"`     // 文件大小 (字节)
	Progress float64 `json:"progress"` // 该文件完成进度 (0.0 - 1.0)
}

// Downloader 统一下载器接口
type Downloader interface {
	// Name 获取下载器名称，如 "qBittorrent"
	Name() string

	// TestConnection 测试与下载器 WebUI 的连通性和认证有效性
	TestConnection(ctx context.Context) error

	// GetAllTorrents 获取下载器当前所有种子任务 (支持按状态 filter 与分类 category 筛选)
	GetAllTorrents(ctx context.Context, filter string, category string) ([]Torrent, error)

	// GetCompletedTorrents 获取符合分类条件的已完成种子任务
	GetCompletedTorrents(ctx context.Context, categories []string) ([]Torrent, error)

	// GetTorrentFiles 获取指定种子内的所有文件列表
	GetTorrentFiles(ctx context.Context, hash string) ([]TorrentFile, error)

	// AddTags 为指定 Hash 的种子添加标签
	AddTags(ctx context.Context, hashes []string, tags []string) error

	// RemoveTags 为指定 Hash 的种子移除标签
	RemoveTags(ctx context.Context, hashes []string, tags []string) error
}
