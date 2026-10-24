package pipeline

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackie8tao/mediahelper/internal/database"
	"gorm.io/gorm"
)

// Repository manages database operations for pipelines, executions, and step records.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new pipeline repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying GORM database instance.
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// CreatePipeline creates a new pipeline record.
func (r *Repository) CreatePipeline(p *database.Pipeline) error {
	if p == nil {
		return errors.New("pipeline cannot be nil")
	}
	return r.db.Create(p).Error
}

// UpdatePipeline updates an existing pipeline record.
func (r *Repository) UpdatePipeline(p *database.Pipeline) error {
	if p == nil {
		return errors.New("pipeline cannot be nil")
	}
	return r.db.Save(p).Error
}

// DeletePipeline deletes a pipeline record by ID.
func (r *Repository) DeletePipeline(id uint) error {
	return r.db.Delete(&database.Pipeline{}, "id = ?", id).Error
}

// GetPipeline finds a pipeline by ID.
func (r *Repository) GetPipeline(id uint) (*database.Pipeline, error) {
	var p database.Pipeline
	if err := r.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPipelineByName finds a pipeline by unique name.
func (r *Repository) GetPipelineByName(name string) (*database.Pipeline, error) {
	var p database.Pipeline
	if err := r.db.First(&p, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPipelines retrieves pipelines, optionally filtering by enabled status.
func (r *Repository) ListPipelines(enabledOnly bool) ([]database.Pipeline, error) {
	var list []database.Pipeline
	tx := r.db.Model(&database.Pipeline{})
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	if err := tx.Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// CreateExecution creates a pipeline execution instance.
func (r *Repository) CreateExecution(exec *database.PipelineExecution) error {
	if exec == nil {
		return errors.New("execution cannot be nil")
	}
	return r.db.Create(exec).Error
}

// UpdateExecution updates a pipeline execution instance.
func (r *Repository) UpdateExecution(exec *database.PipelineExecution) error {
	if exec == nil {
		return errors.New("execution cannot be nil")
	}
	return r.db.Save(exec).Error
}

// GetExecution fetches an execution instance by ID, preloading all associated steps.
func (r *Repository) GetExecution(id string) (*database.PipelineExecution, error) {
	var exec database.PipelineExecution
	if err := r.db.Preload("Steps").First(&exec, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// ListExecutions returns paginated pipeline execution records.
func (r *Repository) ListExecutions(pipelineID uint, limit int, offset int) ([]database.PipelineExecution, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var list []database.PipelineExecution
	var total int64

	tx := r.db.Model(&database.PipelineExecution{})
	if pipelineID > 0 {
		tx = tx.Where("pipeline_id = ?", pipelineID)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := tx.Preload("Steps").Order("started_at DESC").Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// CreateStepExecution creates a step execution record.
func (r *Repository) CreateStepExecution(step *database.StepExecution) error {
	if step == nil {
		return errors.New("step execution cannot be nil")
	}
	return r.db.Create(step).Error
}

// UpdateStepExecution updates a step execution record.
func (r *Repository) UpdateStepExecution(step *database.StepExecution) error {
	if step == nil {
		return errors.New("step execution cannot be nil")
	}
	return r.db.Save(step).Error
}

// GetStepExecutions returns all step execution records for a specific pipeline execution.
func (r *Repository) GetStepExecutions(executionID string) ([]database.StepExecution, error) {
	var steps []database.StepExecution
	if err := r.db.Where("execution_id = ?", executionID).Order("id ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	return steps, nil
}

// GetIncompleteExecutions finds all executions with running or pending status.
func (r *Repository) GetIncompleteExecutions() ([]database.PipelineExecution, error) {
	var list []database.PipelineExecution
	if err := r.db.Preload("Steps").Where("status IN ?", []string{StatusRunning, StatusPending}).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// MarkInterruptedExecutions marks all running or pending executions and their steps as interrupted during recovery.
func (r *Repository) MarkInterruptedExecutions(reason string) (int64, error) {
	incomplete, err := r.GetIncompleteExecutions()
	if err != nil {
		return 0, fmt.Errorf("query incomplete executions error: %w", err)
	}

	if len(incomplete) == 0 {
		return 0, nil
	}

	now := time.Now()
	for _, exec := range incomplete {
		// 1. Mark incomplete steps as interrupted
		for _, step := range exec.Steps {
			if step.Status == StatusRunning || step.Status == StatusPending {
				step.Status = StatusInterrupted
				step.Error = reason
				step.FinishedAt = &now
				_ = r.db.Save(&step)
			}
		}

		// 2. Mark parent pipeline execution as interrupted
		exec.Status = StatusInterrupted
		exec.Error = reason
		exec.FinishedAt = &now
		_ = r.db.Save(&exec)
	}

	return int64(len(incomplete)), nil
}
