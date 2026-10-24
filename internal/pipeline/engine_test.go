package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/database"
)

func TestEngine_SequentialStepsAndParams(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	// Register component 1: Parser
	_ = registry.Register(NewFuncComponent("parser", "Parser", "Parses media title and year", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		name := sCtx.GetStringParam("raw_name", "")
		sCtx.Logf("Received raw name: %s", name)
		if strings.Contains(name, "Oppenheimer") {
			return map[string]any{
				"title": "Oppenheimer",
				"year":  2023,
			}, nil
		}
		return map[string]any{"title": "Unknown", "year": 0}, nil
	}))

	// Register component 2: Scraper
	_ = registry.Register(NewFuncComponent("scraper", "Scraper", "Fetches TMDB metadata", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		title := sCtx.GetStringParam("title", "")
		year := sCtx.GetIntParam("year", 0)
		sCtx.Logf("Scraping TMDB for: %s (%d)", title, year)
		return map[string]any{
			"tmdb_id":   872585,
			"vote":      8.9,
			"formatted": fmt.Sprintf("%s (%d)", title, year),
		}, nil
	}))

	engine := NewEngine(db, registry)

	// Create pipeline definition
	p := &database.Pipeline{
		Name:        "Automated Organization Pipeline",
		Description: "Parses and scrapes metadata",
		Enabled:     true,
		Steps: []database.StepConfig{
			{
				ID:        "step_parse",
				Name:      "Step 1: Parse Torrent Name",
				Component: "parser",
				Params: map[string]any{
					"raw_name": "{{ event.torrent_name }}",
				},
			},
			{
				ID:        "step_scrape",
				Name:      "Step 2: Scrape Metadata",
				Component: "scraper",
				Params: map[string]any{
					"title": "{{ steps.step_parse.title }}",
					"year":  "{{ steps.step_parse.year }}",
				},
			},
		},
	}
	if err := engine.Repo().CreatePipeline(p); err != nil {
		t.Fatalf("CreatePipeline error: %v", err)
	}

	// Execute pipeline
	eventData := map[string]any{
		"torrent_name": "Oppenheimer.2023.2160p.mkv",
		"category":     "movie",
	}
	exec, err := engine.ExecutePipeline(context.Background(), p.ID, "manual", eventData)
	if err != nil {
		t.Fatalf("ExecutePipeline error: %v", err)
	}

	if exec.Status != StatusSuccess {
		t.Errorf("expected pipeline status success, got %s, err: %s", exec.Status, exec.Error)
	}

	// Verify step executions and persistent logs
	steps, err := engine.Repo().GetStepExecutions(exec.ID)
	if err != nil {
		t.Fatalf("GetStepExecutions error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 step executions, got %d", len(steps))
	}

	// Verify Step 1
	if steps[0].Status != StatusSuccess || steps[0].OutputData["title"] != "Oppenheimer" {
		t.Errorf("step 1 unexpected output: %+v", steps[0])
	}
	if !strings.Contains(steps[0].Log, "Received raw name: Oppenheimer.2023.2160p.mkv") {
		t.Errorf("step 1 log missing: %s", steps[0].Log)
	}

	// Verify Step 2 (correctly received output from Step 1 as inputs)
	if steps[1].Status != StatusSuccess || steps[1].OutputData["formatted"] != "Oppenheimer (2023)" {
		t.Errorf("step 2 unexpected output: %+v", steps[1])
	}
	if !strings.Contains(steps[1].Log, "Scraping TMDB for: Oppenheimer (2023)") {
		t.Errorf("step 2 log missing: %s", steps[1].Log)
	}
}

func TestEngine_StepConditionSkipping(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	var step2Executed bool
	_ = registry.Register(NewFuncComponent("comp1", "C1", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		return map[string]any{"is_anime": false}, nil
	}))
	_ = registry.Register(NewFuncComponent("comp2", "C2", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		step2Executed = true
		return map[string]any{"result": "anime processed"}, nil
	}))

	engine := NewEngine(db, registry)

	p := &database.Pipeline{
		Name:    "Condition Skipping Test Pipeline",
		Enabled: true,
		Steps: []database.StepConfig{
			{
				ID:        "step1",
				Name:      "Step 1",
				Component: "comp1",
			},
			{
				ID:        "step2",
				Name:      "Step 2 (Anime only)",
				Component: "comp2",
				When:      "steps.step1.is_anime == true",
			},
		},
	}
	_ = engine.Repo().CreatePipeline(p)

	exec, err := engine.ExecutePipeline(context.Background(), p.ID, "test", nil)
	if err != nil {
		t.Fatalf("ExecutePipeline error: %v", err)
	}

	if step2Executed {
		t.Errorf("expected step2 to be skipped, but it executed")
	}
	if exec.Status != StatusSuccess {
		t.Errorf("expected pipeline status success, got %s", exec.Status)
	}

	steps, _ := engine.Repo().GetStepExecutions(exec.ID)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[1].Status != StatusSkipped {
		t.Errorf("expected step2 status to be skipped, got %s", steps[1].Status)
	}
}

func TestEngine_TopConditionSkipping(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()
	_ = registry.Register(NewFuncComponent("dummy", "Dummy", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		return nil, nil
	}))

	engine := NewEngine(db, registry)

	p := &database.Pipeline{
		Name:       "Top-level Condition Test Pipeline",
		Enabled:    true,
		Conditions: []string{"event.size > 1000"},
		Steps: []database.StepConfig{
			{ID: "s1", Component: "dummy"},
		},
	}
	_ = engine.Repo().CreatePipeline(p)

	// Size does not meet condition
	exec, err := engine.ExecutePipeline(context.Background(), p.ID, "webhook", map[string]any{"size": 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != StatusSkipped {
		t.Errorf("expected pipeline status skipped, got %s", exec.Status)
	}
}

func TestEngine_StepFailureAndContinueOnError(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	_ = registry.Register(NewFuncComponent("fail_comp", "Fail", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		sCtx.Log("About to fail")
		return nil, errors.New("simulated error")
	}))
	_ = registry.Register(NewFuncComponent("next_comp", "Next", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		return map[string]any{"done": true}, nil
	}))

	engine := NewEngine(db, registry)

	// Case 1: continue_on_error = false (default: abort pipeline)
	p1 := &database.Pipeline{
		Name:    "Strict Failure Pipeline",
		Enabled: true,
		Steps: []database.StepConfig{
			{ID: "s1", Component: "fail_comp", ContinueOnError: false},
			{ID: "s2", Component: "next_comp"},
		},
	}
	_ = engine.Repo().CreatePipeline(p1)

	exec1, err1 := engine.ExecutePipeline(context.Background(), p1.ID, "manual", nil)
	if err1 == nil {
		t.Errorf("expected error, got nil")
	}
	if exec1.Status != StatusFailed {
		t.Errorf("expected pipeline status failed, got %s", exec1.Status)
	}

	steps1, _ := engine.Repo().GetStepExecutions(exec1.ID)
	if len(steps1) != 1 {
		t.Errorf("expected only 1 step executed before abort, got %d", len(steps1))
	}

	// Case 2: continue_on_error = true (continue executing remaining steps)
	p2 := &database.Pipeline{
		Name:    "Fault-Tolerant Pipeline",
		Enabled: true,
		Steps: []database.StepConfig{
			{ID: "s1", Component: "fail_comp", ContinueOnError: true},
			{ID: "s2", Component: "next_comp"},
		},
	}
	_ = engine.Repo().CreatePipeline(p2)

	exec2, err2 := engine.ExecutePipeline(context.Background(), p2.ID, "manual", nil)
	if err2 != nil {
		t.Errorf("expected nil error due to continue_on_error, got %v", err2)
	}
	if exec2.Status != StatusSuccess {
		t.Errorf("expected pipeline status success, got %s", exec2.Status)
	}

	steps2, _ := engine.Repo().GetStepExecutions(exec2.ID)
	if len(steps2) != 2 {
		t.Errorf("expected 2 steps executed, got %d", len(steps2))
	}
	if steps2[0].Status != StatusFailed || steps2[1].Status != StatusSuccess {
		t.Errorf("steps status unexpected: step1=%s, step2=%s", steps2[0].Status, steps2[1].Status)
	}
}

func TestEngine_PanicRecovery(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	_ = registry.Register(NewFuncComponent("panic_comp", "Panic", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		panic("nil pointer dereference or fatal error inside plugin")
	}))

	engine := NewEngine(db, registry)

	p := &database.Pipeline{
		Name:    "Panic Recovery Pipeline",
		Enabled: true,
		Steps: []database.StepConfig{
			{ID: "s1", Component: "panic_comp"},
		},
	}
	_ = engine.Repo().CreatePipeline(p)

	exec, err := engine.ExecutePipeline(context.Background(), p.ID, "manual", nil)
	if err == nil {
		t.Fatalf("expected error from panic, got nil")
	}
	if exec.Status != StatusFailed {
		t.Errorf("expected failed status, got %s", exec.Status)
	}
	if !strings.Contains(exec.Error, "panicked: nil pointer dereference") {
		t.Errorf("expected panic details in execution error, got: %s", exec.Error)
	}

	steps, _ := engine.Repo().GetStepExecutions(exec.ID)
	if len(steps) != 1 || steps[0].Status != StatusFailed {
		t.Errorf("step execution should be marked as failed")
	}
	if !strings.Contains(steps[0].Error, "component panicked") {
		t.Errorf("step error should contain panic stack, got: %s", steps[0].Error)
	}
}

func TestEngine_TriggerEventRouting(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	var animeRunCount int32
	var movieRunCount int32

	_ = registry.Register(NewFuncComponent("anime_worker", "Anime", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		atomic.AddInt32(&animeRunCount, 1)
		return nil, nil
	}))
	_ = registry.Register(NewFuncComponent("movie_worker", "Movie", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		atomic.AddInt32(&movieRunCount, 1)
		return nil, nil
	}))

	engine := NewEngine(db, registry)

	// Pipeline 1: Matches webhook:qbittorrent with category == 'anime'
	p1 := &database.Pipeline{
		Name:    "Anime Pipeline",
		Enabled: true,
		Triggers: []database.TriggerConfig{
			{
				Type:   TriggerTypeEvent,
				Source: "webhook:qbittorrent",
				Filter: "event.category == 'anime'",
			},
		},
		Steps: []database.StepConfig{
			{ID: "step_anime", Component: "anime_worker"},
		},
	}
	_ = engine.Repo().CreatePipeline(p1)

	// Pipeline 2: Matches webhook:qbittorrent with category == 'movie'
	p2 := &database.Pipeline{
		Name:    "Movie Pipeline",
		Enabled: true,
		Triggers: []database.TriggerConfig{
			{
				Type:   TriggerTypeEvent,
				Source: "webhook:qbittorrent",
				Filter: "event.category == 'movie'",
			},
		},
		Steps: []database.StepConfig{
			{ID: "step_movie", Component: "movie_worker"},
		},
	}
	_ = engine.Repo().CreatePipeline(p2)

	// Send movie trigger event
	execs, err := engine.TriggerEvent(context.Background(), "webhook:qbittorrent", map[string]any{
		"category": "movie",
		"name":     "Avatar.2009.mkv",
	})
	if err != nil {
		t.Fatalf("TriggerEvent error: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 pipeline triggered, got %d", len(execs))
	}
	if execs[0].PipelineID != p2.ID {
		t.Errorf("expected movie pipeline triggered, got pipelineID %d", execs[0].PipelineID)
	}

	if atomic.LoadInt32(&movieRunCount) != 1 {
		t.Errorf("movieRunCount expected 1, got %d", movieRunCount)
	}
	if atomic.LoadInt32(&animeRunCount) != 0 {
		t.Errorf("animeRunCount expected 0, got %d", animeRunCount)
	}
}

func TestEngine_CrashRecovery(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()
	engine := NewEngine(db, registry)

	// Simulate incomplete running execution left in DB from prior crash
	now := time.Now()
	zombieExec := &database.PipelineExecution{
		ID:            "zombie-exec-001",
		PipelineID:    1,
		PipelineName:  "Interrupted Pipeline",
		TriggerSource: "webhook",
		Status:        StatusRunning,
		StartedAt:     now,
	}
	_ = engine.Repo().CreateExecution(zombieExec)

	zombieStep := &database.StepExecution{
		ExecutionID: zombieExec.ID,
		StepID:      "step1",
		StepName:    "Step 1",
		ComponentID: "comp",
		Status:      StatusRunning,
		StartedAt:   now,
	}
	_ = engine.Repo().CreateStepExecution(zombieStep)

	// Engine startup crash recovery check
	recovered, err := engine.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover error: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected 1 recovered execution, got %d", recovered)
	}

	// Verify status updated to StatusInterrupted
	updatedExec, err := engine.Repo().GetExecution(zombieExec.ID)
	if err != nil {
		t.Fatalf("GetExecution error: %v", err)
	}
	if updatedExec.Status != StatusInterrupted {
		t.Errorf("expected execution status interrupted, got %s", updatedExec.Status)
	}
	if updatedExec.FinishedAt == nil {
		t.Errorf("expected execution finished_at to be populated")
	}

	updatedSteps, _ := engine.Repo().GetStepExecutions(zombieExec.ID)
	if len(updatedSteps) != 1 || updatedSteps[0].Status != StatusInterrupted {
		t.Errorf("expected step status interrupted, got %+v", updatedSteps)
	}
}

func TestEngine_RetryExecution(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	var callTimes int32
	_ = registry.Register(NewFuncComponent("worker", "Worker", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		times := atomic.AddInt32(&callTimes, 1)
		if times == 1 {
			return nil, errors.New("network glitch")
		}
		return map[string]any{"status": "ok"}, nil
	}))

	engine := NewEngine(db, registry)

	p := &database.Pipeline{
		Name:    "Retry Test Pipeline",
		Enabled: true,
		Steps: []database.StepConfig{
			{ID: "w1", Component: "worker"},
		},
	}
	_ = engine.Repo().CreatePipeline(p)

	// First execution fails
	exec1, err := engine.ExecutePipeline(context.Background(), p.ID, "manual", map[string]any{"data": "123"})
	if err == nil || exec1.Status != StatusFailed {
		t.Fatalf("expected first execution to fail")
	}

	// Perform retry
	retryExec, err := engine.RetryExecution(context.Background(), exec1.ID)
	if err != nil {
		t.Fatalf("RetryExecution error: %v", err)
	}
	if retryExec.Status != StatusSuccess {
		t.Errorf("expected retry status success, got %s", retryExec.Status)
	}
	if retryExec.TriggerData["data"] != "123" {
		t.Errorf("expected retry to preserve trigger data, got %+v", retryExec.TriggerData)
	}
}

func TestEngine_CronTrigger(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	registry := NewComponentRegistry()

	var cronRan int32
	_ = registry.Register(NewFuncComponent("cron_job", "Job", "", func(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
		atomic.AddInt32(&cronRan, 1)
		return nil, nil
	}))

	engine := NewEngine(db, registry)

	p := &database.Pipeline{
		Name:    "Scheduled Pipeline",
		Enabled: true,
		Triggers: []database.TriggerConfig{
			{
				Type: TriggerTypeCron,
				Cron: "@hourly",
			},
		},
		Steps: []database.StepConfig{
			{ID: "clean", Component: "cron_job"},
		},
	}
	_ = engine.Repo().CreatePipeline(p)

	// Start engine to register scheduled jobs
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("engine start error: %v", err)
	}
	defer engine.Stop()

	// Verify scheduler registered entry
	if len(engine.Scheduler().entries[p.ID]) != 1 {
		t.Errorf("expected 1 cron entry registered for pipeline %d", p.ID)
	}
}
