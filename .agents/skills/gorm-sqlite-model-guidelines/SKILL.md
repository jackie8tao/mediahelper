---
name: gorm-sqlite-model-guidelines
description: >-
  Use this skill when designing, reviewing, or refactoring GORM models,
  configuring database migrations, or ensuring auto-timestamps in Go + SQLite applications.
---

# GORM SQLite Model Guidelines Skill

This skill provides best practices, field tag conventions, and testing templates for GORM data models with SQLite storage in the `mediahelper` codebase.

## Field Tag Standards

All model structs in `internal/database/models.go` must explicitly define GORM tags rather than relying on default inferences:

| Field Purpose | Recommended GORM Tag | Example |
| :--- | :--- | :--- |
| Primary Key (Int) | `gorm:"primaryKey;autoIncrement"` | `ID uint` |
| Primary Key (String) | `gorm:"primaryKey;size:64;not null"` | `ID string` |
| Foreign Key | `gorm:"size:64;index;not null"` | `ExecutionID string` |
| Has-Many Relation | `gorm:"foreignKey:ExecutionID;references:ID;constraint:OnDelete:CASCADE"` | `Steps []StepExecution` |
| Status / Enum | `gorm:"size:32;index;not null;default:'pending'"` | `Status string` |
| Text / Blob / JSON | `gorm:"type:text;not null"` or `gorm:"serializer:json"` | `Params map[string]any` |
| Indexed Timestamp | `gorm:"index;not null"` or `gorm:"index"` | `StartedAt time.Time` |
| Auto Created Time | `gorm:"autoCreateTime;not null"` | `CreatedAt time.Time` |
| Auto Updated Time | `gorm:"autoUpdateTime;not null"` | `UpdatedAt time.Time` |

## Auto Timestamps Best Practices

1. **Explicit Declarations**: Always declare `autoCreateTime` on `CreatedAt` and `autoUpdateTime` on `UpdatedAt`.
2. **Behavior Guarantee**:
   - `autoCreateTime` automatically sets current timestamp during `db.Create()`.
   - `autoUpdateTime` automatically updates current timestamp during `db.Save()` and `db.Updates()`.
3. **Nullable Completion Timestamps**: Optional end times should be pointer types (`*time.Time`) without automatic tags:
   ```go
   FinishedAt *time.Time `gorm:"index" json:"finished_at,omitempty"`
   ```

## Unit Test Verification Template

When updating or introducing new database models, always verify against an in-memory SQLite database:

```go
package database

import (
	"testing"
	"time"
)

func TestModelAutoTimestamps(t *testing.T) {
	// Initialize shared memory SQLite instance
	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	record := YourModel{/* fields */}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("db.Create failed: %v", err)
	}

	if record.CreatedAt.IsZero() || record.UpdatedAt.IsZero() {
		t.Errorf("expected auto timestamps to be populated")
	}

	origUpdatedAt := record.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	record.SomeField = "new_value"
	if err := db.Save(&record).Error; err != nil {
		t.Fatalf("db.Save failed: %v", err)
	}

	if !record.UpdatedAt.After(origUpdatedAt) {
		t.Errorf("expected UpdatedAt to advance after save")
	}
}
```

