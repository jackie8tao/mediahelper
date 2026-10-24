package database

import (
	"time"
)

// TorrentRecord represents a downloader torrent task entity.
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

// TaskHistory represents a media organization execution history entry.
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

// MediaMapping represents a title correction and ID override mapping.
type MediaMapping struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RawTitle     string    `gorm:"size:256;uniqueIndex;not null" json:"raw_title"` // Raw title keyword or pattern
	TargetTitle  string    `gorm:"size:256;not null" json:"target_title"`          // Corrected official title
	TargetYear   int       `json:"target_year"`                                    // Corrected release year
	TargetTMDBID int       `json:"target_tmdb_id"`                                 // Forced TMDB ID binding
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Pipeline represents a pipeline workflow definition entity.
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

// TriggerConfig defines a trigger condition for a pipeline.
type TriggerConfig struct {
	Type   string `json:"type"`             // event / cron / manual
	Source string `json:"source,omitempty"` // e.g. "webhook:qbittorrent", "downloader:complete"
	Cron   string `json:"cron,omitempty"`   // e.g. "0 2 * * *" or "@daily"
	Filter string `json:"filter,omitempty"` // e.g. "event.category == 'movie'"
}

// StepConfig defines a step within a pipeline workflow.
type StepConfig struct {
	ID              string         `json:"id"`                          // Unique identifier within pipeline, e.g. "parse"
	Name            string         `json:"name"`                        // Friendly display name
	Component       string         `json:"component"`                   // Registered component ID, e.g. "parser"
	When            string         `json:"when,omitempty"`              // Step execution condition expression
	Params          map[string]any `json:"params,omitempty"`            // Component input parameter templates
	ContinueOnError bool           `json:"continue_on_error,omitempty"` // Whether to continue on step failure
	TimeoutSeconds  int            `json:"timeout_seconds,omitempty"`   // Step timeout limit in seconds
}

// PipelineExecution represents an instance of a running or completed pipeline.
type PipelineExecution struct {
	ID            string          `gorm:"primaryKey;size:64;not null" json:"id"`
	PipelineID    uint            `gorm:"index;not null" json:"pipeline_id"`
	PipelineName  string          `gorm:"size:128" json:"pipeline_name"`
	TriggerSource string          `gorm:"size:64;index" json:"trigger_source"` // Trigger source identifier
	TriggerData   map[string]any  `gorm:"serializer:json" json:"trigger_data"`
	Status        string          `gorm:"size:32;index" json:"status"` // pending / running / success / failed / interrupted / skipped
	Error         string          `gorm:"type:text" json:"error,omitempty"`
	StartedAt     time.Time       `gorm:"index" json:"started_at"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	Steps         []StepExecution `gorm:"foreignKey:ExecutionID;references:ID" json:"steps,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// StepExecution represents the execution record of an individual step within a pipeline execution.
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

// SystemSetting represents a persistent system configuration key-value item.
type SystemSetting struct {
	Key       string    `gorm:"primaryKey;size:128;not null" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
