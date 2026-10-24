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

	// 1. Test TorrentRecord CRUD
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

		// Update status
		if err := db.Model(&query).Update("status", "success").Error; err != nil {
			t.Fatalf("update status error: %v", err)
		}

		var updated TorrentRecord
		_ = db.First(&updated, "hash = ?", "mock_hash_001")
		if updated.Status != "success" {
			t.Errorf("updated status mismatch: %s", updated.Status)
		}
	})

	// 2. Test TaskHistory CRUD
	t.Run("TaskHistory CRUD", func(t *testing.T) {
		history := TaskHistory{
			TorrentHash: "mock_hash_001",
			TorrentName: "Oppenheimer.2023.2160p.mkv",
			Title:       "Oppenheimer",
			Year:        2023,
			MediaType:   "movie",
			TargetDir:   "D:/Media/Movies/Oppenheimer (2023)",
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

	// 3. Test MediaMapping CRUD
	t.Run("MediaMapping CRUD", func(t *testing.T) {
		mapping := MediaMapping{
			RawTitle:     "The Long Season",
			TargetTitle:  "The Long Season",
			TargetYear:   2023,
			TargetTMDBID: 224884,
		}

		if err := db.Create(&mapping).Error; err != nil {
			t.Fatalf("create MediaMapping error: %v", err)
		}

		// Test unique constraint error
		dup := MediaMapping{
			RawTitle:    "The Long Season",
			TargetTitle: "Another Name",
		}
		if err := db.Create(&dup).Error; err == nil {
			t.Errorf("expected unique constraint error for duplicate RawTitle")
		}
	})

	// 4. Test Pipeline and Execution CRUD with JSON serialization
	t.Run("Pipeline and Execution CRUD", func(t *testing.T) {
		pipeline := Pipeline{
			Name:        "Default Automated Pipeline",
			Description: "Parses, scrapes and organizes media upon download completion",
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
					Name:      "Parse Title",
					Component: "parser",
					Params: map[string]any{
						"name": "{{ event.name }}",
					},
				},
				{
					ID:              "scrape",
					Name:            "Scrape TMDB",
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

		// Verify JSON serialization and deserialization
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

		// Create execution instance and step execution records
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
			StepName:    "Parse Title",
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

		// Verify relation preloading
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

	// 5. Test SystemSetting CRUD
	t.Run("SystemSetting CRUD", func(t *testing.T) {
		setting := SystemSetting{
			Key:         "server",
			Value:       `{"host":"0.0.0.0","port":9090}`,
			Category:    "server",
			Description: "HTTP server network configuration",
			UpdatedAt:   time.Now(),
		}

		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("create SystemSetting error: %v", err)
		}

		var query SystemSetting
		if err := db.First(&query, "key = ?", "server").Error; err != nil {
			t.Fatalf("query SystemSetting error: %v", err)
		}
		if query.Category != "server" || query.Value != `{"host":"0.0.0.0","port":9090}` {
			t.Errorf("query SystemSetting mismatch: %+v", query)
		}

		// Update
		query.Value = `{"host":"127.0.0.1","port":9091}`
		if err := db.Save(&query).Error; err != nil {
			t.Fatalf("update SystemSetting error: %v", err)
		}

		var updated SystemSetting
		_ = db.First(&updated, "key = ?", "server")
		if updated.Value != `{"host":"127.0.0.1","port":9091}` {
			t.Errorf("expected updated value, got %s", updated.Value)
		}
	})
}
