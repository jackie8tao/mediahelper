package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackie8tao/mediahelper/internal/database"
	"gorm.io/gorm"
)

// Engine is the central pipeline orchestration and execution engine.
type Engine struct {
	repo         *Repository
	registry     *ComponentRegistry
	scheduler    *CronScheduler
	runningExecs sync.Map // executionID -> context.CancelFunc
}

// NewEngine creates a new Engine instance.
func NewEngine(db *gorm.DB, registry *ComponentRegistry) *Engine {
	if registry == nil {
		registry = NewComponentRegistry()
	}

	return &Engine{
		repo:      NewRepository(db),
		registry:  registry,
		scheduler: NewCronScheduler(),
	}
}

// Registry returns the component registry.
func (e *Engine) Registry() *ComponentRegistry {
	return e.registry
}

// Repo returns the underlying repository.
func (e *Engine) Repo() *Repository {
	return e.repo
}

// Scheduler returns the cron scheduler.
func (e *Engine) Scheduler() *CronScheduler {
	return e.scheduler
}

// Start boots the engine, performs crash recovery, loads cron triggers, and starts the scheduler.
func (e *Engine) Start(ctx context.Context) error {
	slog.Info("Starting pipeline engine...")

	// 1. Crash recovery: reconcile incomplete runs from prior shutdown or crash
	interruptedCount, err := e.Recover(ctx)
	if err != nil {
		slog.Warn("Pipeline recovery check encountered error", "error", err)
	} else if interruptedCount > 0 {
		slog.Warn("Recovered incomplete executions from previous crash", "count", interruptedCount)
	}

	// 2. Reload and register enabled cron triggers
	if err := e.ReloadSchedules(ctx); err != nil {
		slog.Error("Failed to reload pipeline schedules", "error", err)
	}

	// 3. Start cron scheduler
	e.scheduler.Start()
	slog.Info("Pipeline engine started successfully")
	return nil
}

// Stop gracefully shuts down the engine and cancels active in-flight executions.
func (e *Engine) Stop() {
	slog.Info("Stopping pipeline engine...")
	e.scheduler.Stop()

	// Cancel all in-flight executions
	e.runningExecs.Range(func(key, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})
	slog.Info("Pipeline engine stopped")
}

// Recover scans for orphaned running/pending executions and marks them as interrupted.
func (e *Engine) Recover(ctx context.Context) (int64, error) {
	reason := "execution interrupted by system restart or unexpected shutdown"
	return e.repo.MarkInterruptedExecutions(reason)
}

// ReloadSchedules loads all enabled pipelines from the database and configures their cron triggers.
func (e *Engine) ReloadSchedules(ctx context.Context) error {
	pipelines, err := e.repo.ListPipelines(true)
	if err != nil {
		return fmt.Errorf("list enabled pipelines error: %w", err)
	}

	for _, p := range pipelines {
		e.scheduler.Remove(p.ID)
		for _, trigger := range p.Triggers {
			if trigger.Type == TriggerTypeCron && trigger.Cron != "" {
				pipelineID := p.ID
				pipelineName := p.Name
				cronExpr := trigger.Cron

				jobHandler := func() {
					now := time.Now()
					triggerData := map[string]any{
						"timestamp": now.Unix(),
						"time":      now.Format(time.RFC3339),
					}
					slog.Info("Triggering cron pipeline", "pipelineID", pipelineID, "pipelineName", pipelineName)
					_, err := e.ExecutePipeline(context.Background(), pipelineID, "cron", triggerData)
					if err != nil {
						slog.Error("Cron pipeline execution failed", "pipelineID", pipelineID, "name", pipelineName, "error", err)
					}
				}

				if err := e.scheduler.Register(pipelineID, cronExpr, jobHandler); err != nil {
					slog.Error("Register cron trigger failed", "pipelineID", pipelineID, "cron", cronExpr, "error", err)
				}
			}
		}
	}

	return nil
}

// TriggerEvent broadcasts an event to matching pipelines and executes them.
func (e *Engine) TriggerEvent(ctx context.Context, source string, eventData map[string]any) ([]*database.PipelineExecution, error) {
	if eventData == nil {
		eventData = make(map[string]any)
	}

	pipelines, err := e.repo.ListPipelines(true)
	if err != nil {
		return nil, fmt.Errorf("list enabled pipelines error: %w", err)
	}

	var executions []*database.PipelineExecution

	for _, p := range pipelines {
		for _, trigger := range p.Triggers {
			if trigger.Type != TriggerTypeEvent {
				continue
			}

			// Match event source (exact match or wildcard "*")
			if trigger.Source != "*" && trigger.Source != source {
				continue
			}

			// Evaluate trigger filter expression
			if trigger.Filter != "" {
				env := map[string]any{
					"event": eventData,
				}
				pass, err := EvaluateCondition(trigger.Filter, env)
				if err != nil {
					slog.Warn("Evaluate trigger filter failed", "pipelineID", p.ID, "filter", trigger.Filter, "error", err)
					continue
				}
				if !pass {
					slog.Debug("Trigger filter evaluated to false, skipping", "pipelineID", p.ID, "filter", trigger.Filter)
					continue
				}
			}

			// Trigger pipeline execution
			exec, err := e.ExecutePipeline(ctx, p.ID, source, eventData)
			if err != nil {
				slog.Error("Pipeline execution returned error", "pipelineID", p.ID, "name", p.Name, "error", err)
			}
			if exec != nil {
				executions = append(executions, exec)
			}
		}
	}

	return executions, nil
}

// ExecutePipeline executes a pipeline by its ID.
func (e *Engine) ExecutePipeline(
	ctx context.Context,
	pipelineID uint,
	triggerSource string,
	triggerData map[string]any,
) (*database.PipelineExecution, error) {
	if triggerData == nil {
		triggerData = make(map[string]any)
	}

	p, err := e.repo.GetPipeline(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline with ID %d not found: %w", pipelineID, err)
	}

	if !p.Enabled {
		return nil, fmt.Errorf("pipeline '%s' (ID: %d) is disabled", p.Name, pipelineID)
	}

	// 1. Evaluate top-level pipeline conditions
	if skippedExec, shouldSkip := e.checkPipelineConditions(p, triggerSource, triggerData); shouldSkip {
		return skippedExec, nil
	}

	// 2. Initialize and persist running execution instance
	execID := uuid.NewString()
	startTime := time.Now()

	exec := &database.PipelineExecution{
		ID:            execID,
		PipelineID:    p.ID,
		PipelineName:  p.Name,
		TriggerSource: triggerSource,
		TriggerData:   triggerData,
		Status:        StatusRunning,
		StartedAt:     startTime,
	}

	if err := e.repo.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("create pipeline execution error: %w", err)
	}

	// 3. Create cancellable context
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	e.runningExecs.Store(execID, cancel)
	defer e.runningExecs.Delete(execID)

	slog.Info("Pipeline execution started", "executionID", execID, "pipeline", p.Name, "triggerSource", triggerSource)

	// 4. Execute steps sequentially
	pipelineErr := e.executeSteps(execCtx, execID, p, triggerData)

	// 5. Finalize execution status
	finishTime := time.Now()
	exec.FinishedAt = &finishTime

	if pipelineErr != nil {
		exec.Status = StatusFailed
		exec.Error = pipelineErr.Error()
	} else {
		exec.Status = StatusSuccess
	}

	_ = e.repo.UpdateExecution(exec)
	slog.Info("Pipeline execution finished", "executionID", execID, "status", exec.Status)

	return exec, pipelineErr
}

// checkPipelineConditions evaluates top-level pipeline conditions. If any condition fails,
// it logs and creates an audit record marked as StatusSkipped.
func (e *Engine) checkPipelineConditions(
	p *database.Pipeline,
	triggerSource string,
	triggerData map[string]any,
) (*database.PipelineExecution, bool) {
	if len(p.Conditions) == 0 {
		return nil, false
	}

	env := map[string]any{"event": triggerData}
	for _, cond := range p.Conditions {
		pass, err := EvaluateCondition(cond, env)
		if err != nil || !pass {
			now := time.Now()
			reason := fmt.Sprintf("Pipeline top condition not met: %s", cond)
			if err != nil {
				reason = fmt.Sprintf("Pipeline condition evaluation error: %v", err)
			}

			skippedExec := &database.PipelineExecution{
				ID:            uuid.NewString(),
				PipelineID:    p.ID,
				PipelineName:  p.Name,
				TriggerSource: triggerSource,
				TriggerData:   triggerData,
				Status:        StatusSkipped,
				Error:         reason,
				StartedAt:     now,
				FinishedAt:    &now,
			}
			_ = e.repo.CreateExecution(skippedExec)
			slog.Info("Pipeline skipped due to condition", "pipelineID", p.ID, "reason", reason)
			return skippedExec, true
		}
	}

	return nil, false
}

// executeSteps runs the configured pipeline steps sequentially, accumulating outputs across steps.
func (e *Engine) executeSteps(
	ctx context.Context,
	execID string,
	p *database.Pipeline,
	triggerData map[string]any,
) error {
	stepOutputs := make(map[string]map[string]any)

	for _, step := range p.Steps {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("pipeline execution cancelled: %w", err)
		}

		output, skipped, err := e.executeStep(ctx, execID, p.ID, step, triggerData, stepOutputs)
		if err != nil {
			slog.Error("Step execution failed", "executionID", execID, "stepID", step.ID, "error", err)
			if !step.ContinueOnError {
				return fmt.Errorf("step '%s' execution failed: %w", step.ID, err)
			}
			continue
		}

		if !skipped {
			if output == nil {
				output = make(map[string]any)
			}
			stepOutputs[step.ID] = output
		}
	}

	return nil
}

// executeStep executes a single pipeline step and records its lifecycle in the database.
func (e *Engine) executeStep(
	ctx context.Context,
	execID string,
	pipelineID uint,
	step database.StepConfig,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
) (map[string]any, bool, error) {
	stepStart := time.Now()
	stepExec := &database.StepExecution{
		ExecutionID: execID,
		StepID:      step.ID,
		StepName:    step.Name,
		ComponentID: step.Component,
		Status:      StatusRunning,
		StartedAt:   stepStart,
	}

	env := map[string]any{
		"event": triggerData,
		"steps": stepOutputs,
	}

	// 1. Evaluate step condition ('when')
	if step.When != "" {
		pass, whenErr := EvaluateCondition(step.When, env)
		if whenErr != nil {
			finish := time.Now()
			stepExec.Status = StatusFailed
			stepExec.Error = fmt.Sprintf("condition evaluation error: %v", whenErr)
			stepExec.FinishedAt = &finish
			_ = e.repo.CreateStepExecution(stepExec)
			return nil, false, whenErr
		}

		if !pass {
			finish := time.Now()
			stepExec.Status = StatusSkipped
			stepExec.Log = fmt.Sprintf("Condition '%s' evaluated to false, step skipped.", step.When)
			stepExec.FinishedAt = &finish
			_ = e.repo.CreateStepExecution(stepExec)
			slog.Debug("Step skipped", "executionID", execID, "stepID", step.ID, "condition", step.When)
			return nil, true, nil
		}
	}

	// 2. Lookup registered component
	comp, found := e.registry.Get(step.Component)
	if !found {
		finish := time.Now()
		stepExec.Status = StatusFailed
		stepExec.Error = fmt.Sprintf("component '%s' not registered", step.Component)
		stepExec.FinishedAt = &finish
		_ = e.repo.CreateStepExecution(stepExec)
		return nil, false, fmt.Errorf("component '%s' not registered", step.Component)
	}

	// 3. Resolve parameters via dynamic interpolation
	var resolvedParams map[string]any
	if len(step.Params) > 0 {
		interpResult, err := Interpolate(step.Params, env)
		if err != nil {
			finish := time.Now()
			stepExec.Status = StatusFailed
			stepExec.Error = fmt.Sprintf("parameter interpolation error: %v", err)
			stepExec.FinishedAt = &finish
			_ = e.repo.CreateStepExecution(stepExec)
			return nil, false, fmt.Errorf("parameter interpolation error: %w", err)
		}
		if m, ok := interpResult.(map[string]any); ok {
			resolvedParams = m
		}
	}
	stepExec.InputData = resolvedParams

	// Persist initial running step record
	if err := e.repo.CreateStepExecution(stepExec); err != nil {
		slog.Warn("Failed to create step execution record in database", "error", err)
	}

	// Build step context
	sCtx := NewStepContext(execID, pipelineID, step.ID, step.Name, step.Component, resolvedParams)

	// 4. Execute component with timeout and panic protection
	output, execErr := e.invokeComponent(ctx, comp, sCtx, step.TimeoutSeconds)

	// 5. Record and persist step execution results
	finish := time.Now()
	stepExec.FinishedAt = &finish
	stepExec.Log = sCtx.GetLogs()
	stepExec.OutputData = output

	if execErr != nil {
		stepExec.Status = StatusFailed
		stepExec.Error = execErr.Error()
		_ = e.repo.UpdateStepExecution(stepExec)
		return nil, false, execErr
	}

	stepExec.Status = StatusSuccess
	_ = e.repo.UpdateStepExecution(stepExec)
	slog.Info("Step execution succeeded", "executionID", execID, "stepID", step.ID)
	return output, false, nil
}

// invokeComponent safely executes a component with timeout control and panic recovery.
func (e *Engine) invokeComponent(
	ctx context.Context,
	comp Component,
	sCtx *StepContext,
	timeoutSeconds int,
) (output map[string]any, err error) {
	var stepCtx context.Context
	var stepCancel context.CancelFunc

	if timeoutSeconds > 0 {
		stepCtx, stepCancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	} else {
		stepCtx, stepCancel = context.WithCancel(ctx)
	}
	defer stepCancel()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("component panicked: %v\nstack trace:\n%s", r, string(debug.Stack()))
		}
	}()

	return comp.Execute(stepCtx, sCtx)
}

// CancelExecution cancels an in-flight execution by ID.
func (e *Engine) CancelExecution(executionID string) bool {
	if val, ok := e.runningExecs.Load(executionID); ok {
		if cancel, ok := val.(context.CancelFunc); ok {
			cancel()
			return true
		}
	}
	return false
}

// RetryExecution re-runs a previous pipeline execution using its original trigger payload.
func (e *Engine) RetryExecution(ctx context.Context, executionID string) (*database.PipelineExecution, error) {
	prev, err := e.repo.GetExecution(executionID)
	if err != nil {
		return nil, fmt.Errorf("find previous execution error: %w", err)
	}

	slog.Info("Retrying pipeline execution", "previousID", executionID, "pipelineID", prev.PipelineID)
	return e.ExecutePipeline(ctx, prev.PipelineID, fmt.Sprintf("retry:%s", executionID), prev.TriggerData)
}
