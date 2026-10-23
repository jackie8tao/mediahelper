package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 系统全局配置
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	QBittorrent QBittorrentConfig `yaml:"qbittorrent"`
	TMDB        TMDBConfig        `yaml:"tmdb"`
	Library     LibraryConfig     `yaml:"library"`
	Plugins     PluginConfig      `yaml:"plugins"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN string `yaml:"dsn"` // 数据库连接字符串或 SQLite 文件路径，例如: data/mediahelper.db
}

// PluginConfig 插件系统全局配置
type PluginConfig struct {
	Enabled bool                              `yaml:"enabled"` // 是否启用插件机制
	Items   map[string]map[string]interface{} `yaml:"items"`   // 每个插件的专属自定义配置
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// LibraryConfig 媒体库整理配置
type LibraryConfig struct {
	MovieDir    string `yaml:"movie_dir"`     // 电影库根目录
	TVDir       string `yaml:"tv_dir"`        // 电视剧库根目录
	AnimeDir    string `yaml:"anime_dir"`     // 动漫库根目录 (为空时默认归入 TVDir)
	LinkMode    string `yaml:"link_mode"`     // 整理模式: hardlink (默认), copy, move, symlink
	MinFileSize int64  `yaml:"min_file_size"` // 最小有效视频文件大小 (字节，默认 50MB)
}

// TMDBConfig TheMovieDatabase 刮削配置
type TMDBConfig struct {
	APIKey         string `yaml:"api_key"`          // TMDB API 密钥 (v3 auth)
	BaseURL        string `yaml:"base_url"`         // API 地址，国内网络可配置反代，如 https://api.tmdb.org/3
	ImageBaseURL   string `yaml:"image_base_url"`   // 图片 CDN 地址，如 https://image.tmdb.org/t/p/original
	Language       string `yaml:"language"`         // 语言代码，默认 zh-CN
	Proxy          string `yaml:"proxy"`            // HTTP/SOCKS5 代理地址，如 http://127.0.0.1:7890
	GenerateNFO    bool   `yaml:"generate_nfo"`     // 是否生成 NFO 刮削文件
	DownloadImages bool   `yaml:"download_images"`  // 是否下载海报与背景图
}

// PathMapping 路径映射配置 (针对 Docker / 远程 qBittorrent)
type PathMapping struct {
	Remote string `yaml:"remote"` // 远程/容器内路径前缀
	Local  string `yaml:"local"`  // 本机/当前容器内路径前缀
}

// QBittorrentConfig qBittorrent 客户端对接配置
type QBittorrentConfig struct {
	Enabled       bool          `yaml:"enabled"`
	URL           string        `yaml:"url"`           // WebUI 地址，例如 http://127.0.0.1:8080
	Username      string        `yaml:"username"`      // 用户名
	Password      string        `yaml:"password"`      // 密码
	Categories    []string      `yaml:"categories"`    // 监控的分类列表，为空则不过滤分类
	AutoTag       bool          `yaml:"auto_tag"`      // 是否自动回写标签
	SuccessTag    string        `yaml:"success_tag"`   // 整理成功后赋予的标签，默认 MediaHelper:Done
	FailTag       string        `yaml:"fail_tag"`      // 整理失败后赋予的标签，默认 MediaHelper:Error
	PollInterval  int           `yaml:"poll_interval"` // 轮询周期(秒)，默认 30
	PathMappings  []PathMapping `yaml:"path_mappings"` // 路径映射规则列表
}

// DefaultConfig 获取默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
		Database: DatabaseConfig{
			DSN: "data/mediahelper.db",
		},
		QBittorrent: QBittorrentConfig{
			Enabled:      true,
			URL:          "http://127.0.0.1:8080",
			Username:     "admin",
			Password:     "adminadmin",
			Categories:   []string{"movie", "tv", "anime"},
			AutoTag:      true,
			SuccessTag:   "MediaHelper:Done",
			FailTag:      "MediaHelper:Error",
			PollInterval: 30,
			PathMappings: []PathMapping{},
		},
		TMDB: TMDBConfig{
			APIKey:         "",
			BaseURL:        "https://api.themoviedb.org/3",
			ImageBaseURL:   "https://image.tmdb.org/t/p/original",
			Language:       "zh-CN",
			Proxy:          "",
			GenerateNFO:    true,
			DownloadImages: true,
		},
		Library: LibraryConfig{
			MovieDir:    "D:/Media/Movies",
			TVDir:       "D:/Media/TV",
			AnimeDir:    "D:/Media/Anime",
			LinkMode:    "hardlink",
			MinFileSize: 50 * 1024 * 1024, // 50MB
		},
		Plugins: PluginConfig{
			Enabled: true,
			Items:   make(map[string]map[string]interface{}),
		},
	}
}

// LoadConfig 从指定 YAML 文件加载配置
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file error: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml error: %w", err)
	}

	return cfg, nil
}
