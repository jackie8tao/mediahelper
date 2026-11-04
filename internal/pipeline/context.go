package pipeline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// StepContext carries resolved step parameters and execution logs for a single step run.
type StepContext struct {
	ExecutionID string
	PipelineID  uint
	StepID      string
	StepName    string
	ComponentID string

	// Params contains the resolved input parameters after template interpolation.
	Params map[string]any

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
) *StepContext {
	if params == nil {
		params = make(map[string]any)
	}

	return &StepContext{
		ExecutionID: executionID,
		PipelineID:  pipelineID,
		StepID:      stepID,
		StepName:    stepName,
		ComponentID: componentID,
		Params:      params,
		logs:        make([]string, 0),
	}
}

// BindParams decodes the resolved Params map into a target struct using JSON tags.
func (c *StepContext) BindParams(target any) error {
	if target == nil {
		return fmt.Errorf("bind target cannot be nil")
	}
	if len(c.Params) == 0 {
		return nil
	}
	raw, err := json.Marshal(c.Params)
	if err != nil {
		return fmt.Errorf("marshal step params error: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal step params error: %w", err)
	}
	return nil
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

// GetStringSliceParam retrieves a string slice parameter from a []string, []any, or comma-separated string.
func (c *StepContext) GetStringSliceParam(key string) []string {
	v, ok := c.Params[key]
	if !ok || v == nil {
		return nil
	}

	var result []string
	switch val := v.(type) {
	case string:
		for _, part := range strings.Split(val, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	case []string:
		for _, item := range val {
			for _, part := range strings.Split(item, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	case []any:
		for _, item := range val {
			if s, ok := item.(string); ok {
				for _, part := range strings.Split(s, ",") {
					if trimmed := strings.TrimSpace(part); trimmed != "" {
						result = append(result, trimmed)
					}
				}
			} else if item != nil {
				if trimmed := strings.TrimSpace(fmt.Sprintf("%v", item)); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
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
