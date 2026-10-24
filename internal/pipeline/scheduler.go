package pipeline

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"
)

// Scheduler defines the cron schedule management interface.
type Scheduler interface {
	Register(pipelineID uint, cronExpr string, job func()) error
	Remove(pipelineID uint)
	Start()
	Stop()
}

// CronScheduler manages cron schedules for pipelines using robfig/cron/v3.
type CronScheduler struct {
	cron    *cron.Cron
	mu      sync.RWMutex
	entries map[uint][]cron.EntryID
	started bool
}

// NewCronScheduler creates a new CronScheduler instance.
func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		cron:    cron.New(),
		entries: make(map[uint][]cron.EntryID),
	}
}

// Register registers a cron trigger for a given pipeline ID.
func (s *CronScheduler) Register(pipelineID uint, cronExpr string, job func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, err := s.cron.AddFunc(cronExpr, job)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	s.entries[pipelineID] = append(s.entries[pipelineID], entryID)
	slog.Info("Registered cron trigger for pipeline", "pipelineID", pipelineID, "cron", cronExpr, "entryID", entryID)
	return nil
}

// Remove removes all cron jobs registered for a given pipeline ID.
func (s *CronScheduler) Remove(pipelineID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryIDs, exists := s.entries[pipelineID]
	if !exists {
		return
	}

	for _, id := range entryIDs {
		s.cron.Remove(id)
	}
	delete(s.entries, pipelineID)
	slog.Info("Removed cron triggers for pipeline", "pipelineID", pipelineID)
}

// Start starts the background cron scheduler.
func (s *CronScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		s.cron.Start()
		s.started = true
		slog.Info("Pipeline cron scheduler started")
	}
}

// Stop stops the background cron scheduler.
func (s *CronScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		s.cron.Stop()
		s.started = false
		slog.Info("Pipeline cron scheduler stopped")
	}
}
