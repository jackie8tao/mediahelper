package database

import (
	"testing"
	"time"
)

func TestInitDB_Memory(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB in memory failed: %v", err)
	}

	// 1. 测试 TorrentRecord CRUD
	t.Run("TorrentRecord CRUD", func(t *testing.T) {
		record := TorrentRecord{
			Hash:        "mock_hash_001",
			Name:        "Oppenheimer.2023.2160p.mkv",
			Category:    "movie",
			Status:      "pending",
			SavePath:    "/downloads/movies",
			ContentPath: "/downloads/movies/Oppenheimer.2023.2160p.mkv",
		}

		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("create TorrentRecord error: %v", err)
		}

		var query TorrentRecord
		if err := db.First(&query, "hash = ?", "mock_hash_001").Error; err != nil {
			t.Fatalf("query TorrentRecord error: %v", err)
		}

		if query.Name != "Oppenheimer.2023.2160p.mkv" {
			t.Errorf("record name mismatch: %s", query.Name)
		}

		// 更新状态
		if err := db.Model(&query).Update("status", "success").Error; err != nil {
			t.Fatalf("update status error: %v", err)
		}

		var updated TorrentRecord
		_ = db.First(&updated, "hash = ?", "mock_hash_001")
		if updated.Status != "success" {
			t.Errorf("updated status mismatch: %s", updated.Status)
		}
	})

	// 2. 测试 TaskHistory
	t.Run("TaskHistory CRUD", func(t *testing.T) {
		history := TaskHistory{
			TorrentHash: "mock_hash_001",
			TorrentName: "Oppenheimer.2023.2160p.mkv",
			Title:       "奥本海默",
			Year:        2023,
			MediaType:   "movie",
			TargetDir:   "D:/Media/Movies/奥本海默 (2023)",
			FileCount:   1,
			Status:      "success",
			CreatedAt:   time.Now(),
		}

		if err := db.Create(&history).Error; err != nil {
			t.Fatalf("create TaskHistory error: %v", err)
		}

		if history.ID == 0 {
			t.Errorf("expected auto-increment ID, got 0")
		}

		var count int64
		db.Model(&TaskHistory{}).Where("torrent_hash = ?", "mock_hash_001").Count(&count)
		if count != 1 {
			t.Errorf("expected history count 1, got %d", count)
		}
	})

	// 3. 测试 MediaMapping
	t.Run("MediaMapping CRUD", func(t *testing.T) {
		mapping := MediaMapping{
			RawTitle:     "漫长的季节",
			TargetTitle:  "漫长的季节",
			TargetYear:   2023,
			TargetTMDBID: 224884,
		}

		if err := db.Create(&mapping).Error; err != nil {
			t.Fatalf("create MediaMapping error: %v", err)
		}

		// 测试唯一索引限制
		dup := MediaMapping{
			RawTitle:    "漫长的季节",
			TargetTitle: "其他名称",
		}
		if err := db.Create(&dup).Error; err == nil {
			t.Errorf("expected unique constraint error for duplicate RawTitle")
		}
	})
}
