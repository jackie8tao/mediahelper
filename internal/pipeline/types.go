package pipeline

import (
	"time"

	"github.com/jackie8tao/mediahelper/internal/database"
)

// Execution and Step Status constants
const (
	StatusPending     = "pending"
	StatusRunning     = "running"
	StatusSuccess     = "success"
	StatusFailed      = "failed"
	StatusSkipped     = "skipped"
	StatusInterrupted = "interrupted"
)

// Trigger Type constants
const (
	TriggerTypeEvent  = "event"
	TriggerTypeCron   = "cron"
	TriggerTypeManual = "manual"
)

// TriggerEvent represents an incoming event that can trigger pipelines
type TriggerEvent struct {
	Source    string         `json:"source"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// Convenient type aliases to database models
type PipelineModel = database.Pipeline
type TriggerConfig = database.TriggerConfig
type StepConfig = database.StepConfig
type PipelineExecutionModel = database.PipelineExecution
type StepExecutionModel = database.StepExecution
