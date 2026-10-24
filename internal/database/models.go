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

// Pipeline 流水线定义实体
type Pipeline struct {
	ID          uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string          `gorm:"size:512" json:"description"`
	Enabled     bool            `gorm:"default:true;index" json:"enabled"`
	Triggers    []TriggerConfig `gorm:"serializer:json" json:"triggers"`
	Conditions  []string        `gorm:"serializer:json" json:"conditions"`
	Steps       []StepConfig    `gorm:"serializer:json" json:"steps"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TriggerConfig 触发器配置定义
type TriggerConfig struct {
	Type   string `json:"type"`             // event / cron / manual
	Source string `json:"source,omitempty"` // e.g. "webhook:qbittorrent", "downloader:complete"
	Cron   string `json:"cron,omitempty"`   // e.g. "0 2 * * *" or "@daily"
	Filter string `json:"filter,omitempty"` // e.g. "event.category == 'movie'"
}

// StepConfig 步骤配置定义
type StepConfig struct {
	ID              string         `json:"id"`                          // 流水线内唯一标识，如 "parse"
	Name            string         `json:"name"`                        // 友好展示名称
	Component       string         `json:"component"`                   // 绑定的组件 ID，如 "parser"
	When            string         `json:"when,omitempty"`              // 步骤前置条件表达式
	Params          map[string]any `json:"params,omitempty"`            // 传递给组件的入参模板
	ContinueOnError bool           `json:"continue_on_error,omitempty"` // 步骤失败是否继续执行
	TimeoutSeconds  int            `json:"timeout_seconds,omitempty"`   // 超时限制 (秒)
}

// PipelineExecution 流水线执行实例实体
type PipelineExecution struct {
	ID            string          `gorm:"primaryKey;size:64;not null" json:"id"`
	PipelineID    uint            `gorm:"index;not null" json:"pipeline_id"`
	PipelineName  string          `gorm:"size:128" json:"pipeline_name"`
	TriggerSource string          `gorm:"size:64;index" json:"trigger_source"` // 触发源
	TriggerData   map[string]any  `gorm:"serializer:json" json:"trigger_data"`
	Status        string          `gorm:"size:32;index" json:"status"` // pending / running / success / failed / interrupted / skipped
	Error         string          `gorm:"type:text" json:"error,omitempty"`
	StartedAt     time.Time       `gorm:"index" json:"started_at"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	Steps         []StepExecution `gorm:"foreignKey:ExecutionID;references:ID" json:"steps,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// StepExecution 步骤执行记录实体
type StepExecution struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutionID string         `gorm:"size:64;index;not null" json:"execution_id"`
	StepID      string         `gorm:"size:64;index;not null" json:"step_id"`
	StepName    string         `gorm:"size:128" json:"step_name"`
	ComponentID string         `gorm:"size:64;index;not null" json:"component_id"`
	Status      string         `gorm:"size:32;index" json:"status"` // pending / running / success / failed / skipped / interrupted
	InputData   map[string]any `gorm:"serializer:json" json:"input_data"`
	OutputData  map[string]any `gorm:"serializer:json" json:"output_data"`
	Log         string         `gorm:"type:text" json:"log"`
	Error       string         `gorm:"type:text" json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
