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

// InitDB 初始化 SQLite 数据库连接并执行模型自动迁移
func InitDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "data/mediahelper.db"
	}

	// 内存数据库常用于单元测试 (:memory:)
	isMemory := strings.Contains(dsn, ":memory:")

	if !isMemory {
		// 确保 SQLite 数据库文件所在目录存在
		dbDir := filepath.Dir(dsn)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create database directory error: %w", err)
			}
		}

		// 为 SQLite 开启 WAL (预写日志模式) 与 5000ms 忙等待超时，避免数据库锁定
		if !strings.Contains(dsn, "_pragma") {
			delimiter := "?"
			if strings.Contains(dsn, "?") {
				delimiter = "&"
			}
			dsn = fmt.Sprintf("%s%s_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dsn, delimiter)
		}
	}

	// 初始化 GORM (使用静默日志避免刷屏)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database error: %w", err)
	}

	// 获取底层 SQL 连接池配置并发安全
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB error: %w", err)
	}

	// 针对 SQLite 单写入者特性设置安全连接池
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// 自动迁移表结构
	if err := db.AutoMigrate(&TorrentRecord{}, &TaskHistory{}, &MediaMapping{}); err != nil {
		return nil, fmt.Errorf("database auto migrate error: %w", err)
	}

	slog.Info("SQLite database initialized successfully", "dsn", dsn)
	return db, nil
}
