package organizer

import (
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// LinkMode 归档处理模式
type LinkMode string

const (
	LinkModeHardlink LinkMode = "hardlink" // 硬链接 (推荐，零存储占用，不影响做种)
	LinkModeCopy     LinkMode = "copy"     // 复制文件
	LinkModeMove     LinkMode = "move"     // 移动文件
	LinkModeSymlink  LinkMode = "symlink"  // 符号链接
)

// OrganizeTask 整理归档任务请求
type OrganizeTask struct {
	SourcePath string                 `json:"source_path"` // 待整理源文件路径或目录 (本机绝对路径)
	MediaInfo  *parser.MediaInfo      `json:"media_info"`  // 解析出的规格信息
	Metadata   *scraper.MediaMetadata `json:"metadata"`    // 刮削出的 TMDB 规范元数据
}

// FilePair 整理前后的文件映射关系
type FilePair struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// OrganizeResult 整理结果报告
type OrganizeResult struct {
	Success     bool       `json:"success"`
	TargetDir   string     `json:"target_dir"`   // 目标影视目录
	LinkedFiles []FilePair `json:"linked_files"` // 成功归档的文件对清单
	NFOPath     string     `json:"nfo_path"`     // 生成的主 NFO 路径
	PosterPath  string     `json:"poster_path"`  // 海报保存路径
	FanartPath  string     `json:"fanart_path"`  // 背景图保存路径
	Error       string     `json:"error,omitempty"`
}
