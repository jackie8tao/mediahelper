package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// StepContext carries resolved parameters, incoming event data, upstream step outputs, and execution logs for a step.
type StepContext struct {
	ExecutionID string
	PipelineID  uint
	StepID      string
	StepName    string
	ComponentID string

	// Params contains the resolved input parameters after template interpolation.
	Params map[string]any

	// Event is the original trigger event payload.
	Event map[string]any

	// Steps contains outputs from previously completed steps: steps[stepID] = outputMap.
	Steps map[string]map[string]any

	mu   sync.Mutex
	logs []string
}

// NewStepContext creates a new StepContext instance.
func NewStepContext(
	executionID string,
	pipelineID uint,
	stepID string,
	stepName string,
	componentID string,
	params map[string]any,
	event map[string]any,
	steps map[string]map[string]any,
) *StepContext {
	if params == nil {
		params = make(map[string]any)
	}
	if event == nil {
		event = make(map[string]any)
	}
	if steps == nil {
		steps = make(map[string]map[string]any)
	}

	return &StepContext{
		ExecutionID: executionID,
		PipelineID:  pipelineID,
		StepID:      stepID,
		StepName:    stepName,
		ComponentID: componentID,
		Params:      params,
		Event:       event,
		Steps:       steps,
		logs:        make([]string, 0),
	}
}

// Log records a single timestamped log entry.
func (c *StepContext) Log(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	c.logs = append(c.logs, fmt.Sprintf("[%s] %s", timestamp, msg))
}

// Logf formats and records a timestamped log entry.
func (c *StepContext) Logf(format string, args ...any) {
	c.Log(fmt.Sprintf(format, args...))
}

// GetLogs returns all collected logs joined by newline.
func (c *StepContext) GetLogs() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.logs, "\n")
}

// GetParam retrieves a parameter by key.
func (c *StepContext) GetParam(key string) (any, bool) {
	v, ok := c.Params[key]
	return v, ok
}

// GetStringParam retrieves a string parameter with a fallback default.
func (c *StepContext) GetStringParam(key string, defaultVal string) string {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return defaultVal
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// GetIntParam retrieves an integer parameter with a fallback default.
func (c *StepContext) GetIntParam(key string, defaultVal int) int {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return defaultVal
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetBoolParam retrieves a boolean parameter with a fallback default.
func (c *StepContext) GetBoolParam(key string, defaultVal bool) bool {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return defaultVal
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(strings.TrimSpace(val))
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" {
			return false
		}
	case int:
		return val != 0
	case float64:
		return val != 0
	}
	return defaultVal
}

// GetMapParam retrieves a map[string]any parameter.
func (c *StepContext) GetMapParam(key string) map[string]any {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// GetSliceParam retrieves a slice parameter.
func (c *StepContext) GetSliceParam(key string) []any {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}
