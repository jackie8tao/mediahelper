package database

import (
	"testing"
	"time"
)

func TestAutoTimestampsAndModels(t *testing.T) {
	// Initialize in-memory SQLite database
	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	t.Run("SystemSetting auto timestamps", func(t *testing.T) {
		setting := SystemSetting{
			Key:   "test_key",
			Value: "test_value",
		}
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("failed to create SystemSetting: %v", err)
		}

		if setting.CreatedAt.IsZero() {
			t.Errorf("expected CreatedAt to be set automatically, got zero time")
		}
		if setting.UpdatedAt.IsZero() {
			t.Errorf("expected UpdatedAt to be set automatically, got zero time")
		}

		origCreatedAt := setting.CreatedAt
		origUpdatedAt := setting.UpdatedAt

		// Wait briefly to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		setting.Value = "updated_value"
		if err := db.Save(&setting).Error; err != nil {
			t.Fatalf("failed to update SystemSetting: %v", err)
		}

		if !setting.CreatedAt.Equal(origCreatedAt) {
			t.Errorf("expected CreatedAt to remain %v, got %v", origCreatedAt, setting.CreatedAt)
		}
		if !setting.UpdatedAt.After(origUpdatedAt) {
			t.Errorf("expected UpdatedAt %v to be after %v", setting.UpdatedAt, origUpdatedAt)
		}
	})

	t.Run("Pipeline auto timestamps", func(t *testing.T) {
		pipeline := Pipeline{
			Name:        "test-pipeline",
			Description: "Pipeline test description",
			Enabled:     true,
			Triggers: []TriggerConfig{
				{Type: "event", Source: "webhook:qbittorrent"},
			},
		}
		if err := db.Create(&pipeline).Error; err != nil {
			t.Fatalf("failed to create Pipeline: %v", err)
		}

		if pipeline.CreatedAt.IsZero() {
			t.Errorf("expected Pipeline.CreatedAt to be set automatically")
		}
		if pipeline.UpdatedAt.IsZero() {
			t.Errorf("expected Pipeline.UpdatedAt to be set automatically")
		}

		origCreatedAt := pipeline.CreatedAt
		origUpdatedAt := pipeline.UpdatedAt

		time.Sleep(10 * time.Millisecond)

		pipeline.Description = "Updated pipeline description"
		if err := db.Save(&pipeline).Error; err != nil {
			t.Fatalf("failed to update Pipeline: %v", err)
		}

		if !pipeline.CreatedAt.Equal(origCreatedAt) {
			t.Errorf("expected CreatedAt to stay unchanged")
		}
		if !pipeline.UpdatedAt.After(origUpdatedAt) {
			t.Errorf("expected UpdatedAt %v to be after %v", pipeline.UpdatedAt, origUpdatedAt)
		}
	})

	t.Run("PipelineExecution and StepExecution cascade and timestamps", func(t *testing.T) {
		exec := PipelineExecution{
			ID:           "exec-test-1",
			PipelineID:   1,
			PipelineName: "test-pipeline",
			StartedAt:    time.Now(),
			Steps: []StepExecution{
				{
					ExecutionID: "exec-test-1",
					StepID:      "parse",
					StepName:    "Parse metadata",
					ComponentID: "parser",
					StartedAt:   time.Now(),
				},
			},
		}

		if err := db.Create(&exec).Error; err != nil {
			t.Fatalf("failed to create PipelineExecution: %v", err)
		}

		if exec.CreatedAt.IsZero() || exec.UpdatedAt.IsZero() {
			t.Errorf("expected execution timestamps to be automatically populated")
		}

		var step StepExecution
		if err := db.Where("execution_id = ?", "exec-test-1").First(&step).Error; err != nil {
			t.Fatalf("failed to retrieve StepExecution: %v", err)
		}

		if step.CreatedAt.IsZero() || step.UpdatedAt.IsZero() {
			t.Errorf("expected step execution timestamps to be automatically populated")
		}
		if step.Status != "pending" {
			t.Errorf("expected default status 'pending', got '%s'", step.Status)
		}
		if exec.Status != "pending" {
			t.Errorf("expected default status 'pending', got '%s'", exec.Status)
		}
	})
}
