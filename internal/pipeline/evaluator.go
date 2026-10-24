package pipeline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

var (
	// exactExprRegex matches a standalone expression block like "{{ event.title }}" or "${ steps.parse.year }"
	exactExprRegex = regexp.MustCompile(`^(?:\{\{|\$\{)\s*(.+?)\s*(?:\}\}|\})$`)
	// inlineExprRegex matches embedded expressions like "prefix-{{ event.id }}-suffix"
	inlineExprRegex = regexp.MustCompile(`(?:\{\{|\$\{)\s*(.+?)\s*(?:\}\}|\})`)
)

// EvaluateCondition evaluates a boolean condition expression against the provided environment.
func EvaluateCondition(condition string, env map[string]any) (bool, error) {
	trimmed := strings.TrimSpace(condition)
	if trimmed == "" {
		return true, nil
	}

	if env == nil {
		env = make(map[string]any)
	}

	program, err := expr.Compile(trimmed, expr.Env(env))
	if err != nil {
		return false, fmt.Errorf("compile condition '%s' failed: %w", condition, err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("evaluate condition '%s' failed: %w", condition, err)
	}

	switch v := output.(type) {
	case bool:
		return v, nil
	case nil:
		return false, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int8:
		return v != 0, nil
	case uint:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case uint32:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case uint8:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case string:
		return v != "", nil
	default:
		return true, nil
	}
}

// Interpolate evaluates dynamic expressions and template placeholders on arbitrary values (string, map, slice).
func Interpolate(val any, env map[string]any) (any, error) {
	if val == nil {
		return nil, nil
	}
	if env == nil {
		env = make(map[string]any)
	}

	switch v := val.(type) {
	case string:
		return interpolateString(v, env)
	case map[string]any:
		result := make(map[string]any, len(v))
		for k, item := range v {
			interpVal, err := Interpolate(item, env)
			if err != nil {
				return nil, fmt.Errorf("field '%s': %w", k, err)
			}
			result[k] = interpVal
		}
		return result, nil
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			interpVal, err := Interpolate(item, env)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			result[i] = interpVal
		}
		return result, nil
	default:
		// Return primitive types (number, boolean) as-is
		return val, nil
	}
}

func interpolateString(s string, env map[string]any) (any, error) {
	trimmed := strings.TrimSpace(s)

	// 1. Check if the string is a standalone expression, preserving the returned type
	if match := exactExprRegex.FindStringSubmatch(trimmed); len(match) == 2 {
		exprStr := match[1]
		program, err := expr.Compile(exprStr, expr.Env(env))
		if err != nil {
			return nil, fmt.Errorf("compile expression '%s' error: %w", exprStr, err)
		}
		output, err := expr.Run(program, env)
		if err != nil {
			return nil, fmt.Errorf("evaluate expression '%s' error: %w", exprStr, err)
		}
		return output, nil
	}

	// 2. Check if the string contains embedded expressions, stringifying evaluated fragments
	if inlineExprRegex.MatchString(s) {
		var evalErr error
		replaced := inlineExprRegex.ReplaceAllStringFunc(s, func(sub string) string {
			if evalErr != nil {
				return sub
			}
			match := exactExprRegex.FindStringSubmatch(strings.TrimSpace(sub))
			if len(match) != 2 {
				return sub
			}
			exprStr := match[1]
			program, err := expr.Compile(exprStr, expr.Env(env))
			if err != nil {
				evalErr = fmt.Errorf("compile expression '%s' error: %w", exprStr, err)
				return sub
			}
			output, err := expr.Run(program, env)
			if err != nil {
				evalErr = fmt.Errorf("evaluate expression '%s' error: %w", exprStr, err)
				return sub
			}
			if output == nil {
				return ""
			}
			return fmt.Sprintf("%v", output)
		})
		if evalErr != nil {
			return nil, evalErr
		}
		return replaced, nil
	}

	// 3. Regular string without placeholders
	return s, nil
}
