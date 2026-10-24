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

	// 4. 测试 Pipeline 及关联执行实体 CRUD
	t.Run("Pipeline and Execution CRUD", func(t *testing.T) {
		pipeline := Pipeline{
			Name:        "默认自动化入库流水线",
			Description: "下载完成后自动解析、TMDB刮削与归档",
			Enabled:     true,
			Triggers: []TriggerConfig{
				{
					Type:   "event",
					Source: "webhook:qbittorrent",
					Filter: "event.category == 'movie'",
				},
				{
					Type: "cron",
					Cron: "0 2 * * *",
				},
			},
			Conditions: []string{"event.size > 0"},
			Steps: []StepConfig{
				{
					ID:        "parse",
					Name:      "智能标题识别",
					Component: "parser",
					Params: map[string]any{
						"name": "{{ event.name }}",
					},
				},
				{
					ID:              "scrape",
					Name:            "TMDB刮削",
					Component:       "scraper",
					When:            "steps.parse.title != ''",
					ContinueOnError: true,
				},
			},
		}

		if err := db.Create(&pipeline).Error; err != nil {
			t.Fatalf("create Pipeline error: %v", err)
		}
		if pipeline.ID == 0 {
			t.Fatalf("expected pipeline ID > 0")
		}

		// 查询验证 JSON 序列化反序列化是否正常
		var query Pipeline
		if err := db.First(&query, "id = ?", pipeline.ID).Error; err != nil {
			t.Fatalf("query Pipeline error: %v", err)
		}
		if len(query.Triggers) != 2 || query.Triggers[0].Source != "webhook:qbittorrent" {
			t.Errorf("expected 2 triggers, got %+v", query.Triggers)
		}
		if len(query.Steps) != 2 || query.Steps[0].ID != "parse" {
			t.Errorf("expected 2 steps, got %+v", query.Steps)
		}

		// 创建执行实例与步骤执行记录
		now := time.Now()
		exec := PipelineExecution{
			ID:            "exec-test-uuid-001",
			PipelineID:    pipeline.ID,
			PipelineName:  pipeline.Name,
			TriggerSource: "webhook:qbittorrent",
			TriggerData: map[string]any{
				"name":     "Oppenheimer.2023.mkv",
				"category": "movie",
				"size":     1048576,
			},
			Status:    "running",
			StartedAt: now,
		}
		if err := db.Create(&exec).Error; err != nil {
			t.Fatalf("create PipelineExecution error: %v", err)
		}

		stepExec := StepExecution{
			ExecutionID: exec.ID,
			StepID:      "parse",
			StepName:    "智能标题识别",
			ComponentID: "parser",
			Status:      "success",
			InputData: map[string]any{
				"name": "Oppenheimer.2023.mkv",
			},
			OutputData: map[string]any{
				"title": "Oppenheimer",
				"year":  2023,
			},
			Log:       "Parsed title: Oppenheimer, Year: 2023",
			StartedAt: now,
		}
		if err := db.Create(&stepExec).Error; err != nil {
			t.Fatalf("create StepExecution error: %v", err)
		}

		// 关联查询验证
		var queryExec PipelineExecution
		if err := db.Preload("Steps").First(&queryExec, "id = ?", exec.ID).Error; err != nil {
			t.Fatalf("query PipelineExecution with Steps error: %v", err)
		}
		if len(queryExec.Steps) != 1 {
			t.Errorf("expected 1 step in execution, got %d", len(queryExec.Steps))
		}
		if queryExec.Steps[0].OutputData["title"] != "Oppenheimer" {
			t.Errorf("expected output title 'Oppenheimer', got %v", queryExec.Steps[0].OutputData["title"])
		}
	})
}
