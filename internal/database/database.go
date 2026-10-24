package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes SQLite database connection and runs auto migrations.
func InitDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "data/mediahelper.db"
	}

	// In-memory databases are often used for unit tests (:memory:)
	isMemory := strings.Contains(dsn, ":memory:")

	if !isMemory {
		// Ensure SQLite database directory exists
		dbDir := filepath.Dir(dsn)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create database directory error: %w", err)
			}
		}

		// Enable WAL mode and 5000ms busy timeout to avoid database locks
		if !strings.Contains(dsn, "_pragma") {
			delimiter := "?"
			if strings.Contains(dsn, "?") {
				delimiter = "&"
			}
			dsn = fmt.Sprintf("%s%s_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dsn, delimiter)
		}
	}

	// Initialize GORM with silent logger
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database error: %w", err)
	}

	// Configure underlying connection pool for thread safety
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB error: %w", err)
	}

	// SQLite single-writer connection pool configuration
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Auto-migrate schema entities
	if err := db.AutoMigrate(
		&TorrentRecord{},
		&TaskHistory{},
		&MediaMapping{},
		&Pipeline{},
		&PipelineExecution{},
		&StepExecution{},
		&SystemSetting{},
	); err != nil {
		return nil, fmt.Errorf("database auto migrate error: %w", err)
	}

	slog.Info("SQLite database initialized successfully", "dsn", dsn)
	return db, nil
}
