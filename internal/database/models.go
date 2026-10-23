package database

import (
	"time"
)

// TorrentRecord 种子任务记录实体
type TorrentRecord struct {
	Hash        string    `gorm:"primaryKey;size:64;not null" json:"hash"`
	Name        string    `gorm:"size:512;not null" json:"name"`
	Category    string    `gorm:"size:128" json:"category"`
	Status      string    `gorm:"size:32;index" json:"status"` // pending / success / failed
	SavePath    string    `gorm:"size:1024" json:"save_path"`
	ContentPath string    `gorm:"size:1024" json:"content_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskHistory 归档整理执行历史记录实体
type TaskHistory struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TorrentHash string    `gorm:"size:64;index" json:"torrent_hash"`
	TorrentName string    `gorm:"size:512" json:"torrent_name"`
	Title       string    `gorm:"size:256;index" json:"title"`
	Year        int       `json:"year"`
	MediaType   string    `gorm:"size:32" json:"media_type"`
	TargetDir   string    `gorm:"size:1024" json:"target_dir"`
	FileCount   int       `json:"file_count"`
	Status      string    `gorm:"size:32;index" json:"status"` // success / failed / skipped
	Error       string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// MediaMapping 剧名纠错映射实体
type MediaMapping struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RawTitle     string    `gorm:"size:256;uniqueIndex;not null" json:"raw_title"` // 原始标题关键词或特征
	TargetTitle  string    `gorm:"size:256;not null" json:"target_title"`          // 矫正后的正式名称
	TargetYear   int       `json:"target_year"`                                    // 矫正后的年份
	TargetTMDBID int       `json:"target_tmdb_id"`                                 // 强制绑定的 TMDB ID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
