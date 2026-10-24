package pipeline

import (
	"reflect"
	"testing"
)

func TestEvaluateCondition(t *testing.T) {
	env := map[string]any{
		"event": map[string]any{
			"category": "movie",
			"size":     1048576,
			"tags":     []string{"4k", "hdr"},
		},
		"steps": map[string]any{
			"parse": map[string]any{
				"title": "Oppenheimer",
				"year":  2023,
			},
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
		hasError  bool
	}{
		{"empty condition", "", true, false},
		{"simple string match", "event.category == 'movie'", true, false},
		{"simple string mismatch", "event.category == 'tv'", false, false},
		{"numeric comparison", "event.size > 1000", true, false},
		{"nested step check", "steps.parse.title == 'Oppenheimer' && steps.parse.year == 2023", true, false},
		{"in operator", "'4k' in event.tags", true, false},
		{"not in operator", "'remux' in event.tags", false, false},
		{"numeric truthy", "event.size", true, false},
		{"numeric zero falsy", "0", false, false},
		{"int64 zero falsy", "int(0)", false, false},
		{"syntax error", "event.category === 'bad'", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvaluateCondition(tt.condition, env)
			if (err != nil) != tt.hasError {
				t.Fatalf("expected error: %v, got: %v", tt.hasError, err)
			}
			if !tt.hasError && res != tt.expected {
				t.Errorf("condition '%s': expected %v, got %v", tt.condition, tt.expected, res)
			}
		})
	}
}

func TestInterpolate(t *testing.T) {
	env := map[string]any{
		"event": map[string]any{
			"hash": "abc123456",
			"name": "Oppenheimer.2023.2160p.mkv",
		},
		"steps": map[string]any{
			"parse": map[string]any{
				"title": "Oppenheimer",
				"year":  2023,
			},
		},
	}

	// 1. Test preserving exact types for single expression
	t.Run("typed exact expression", func(t *testing.T) {
		res, err := Interpolate("{{ steps.parse.year }}", env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != 2023 {
			t.Errorf("expected int 2023, got %v (%T)", res, res)
		}
	})

	// 2. Test inline string interpolation
	t.Run("inline string interpolation", func(t *testing.T) {
		res, err := Interpolate("Movie: {{ steps.parse.title }} ({{ steps.parse.year }}) - {{ event.hash }}", env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Movie: Oppenheimer (2023) - abc123456"
		if res != expected {
			t.Errorf("expected '%s', got '%v'", expected, res)
		}
	})

	// 3. Test nested Map / Slice recursive interpolation
	t.Run("nested structure interpolation", func(t *testing.T) {
		input := map[string]any{
			"title":    "{{ steps.parse.title }}",
			"year":     "{{ steps.parse.year }}",
			"filename": "prefix_{{ event.name }}",
			"tags":     []any{"release", "{{ steps.parse.title }}"},
			"static":   true,
		}

		res, err := Interpolate(input, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resMap, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", res)
		}

		if resMap["title"] != "Oppenheimer" {
			t.Errorf("expected title 'Oppenheimer', got %v", resMap["title"])
		}
		if resMap["year"] != 2023 {
			t.Errorf("expected year 2023, got %v", resMap["year"])
		}
		if resMap["filename"] != "prefix_Oppenheimer.2023.2160p.mkv" {
			t.Errorf("expected filename 'prefix_Oppenheimer.2023.2160p.mkv', got %v", resMap["filename"])
		}
		expectedTags := []any{"release", "Oppenheimer"}
		if !reflect.DeepEqual(resMap["tags"], expectedTags) {
			t.Errorf("expected tags %v, got %v", expectedTags, resMap["tags"])
		}
		if resMap["static"] != true {
			t.Errorf("expected static true, got %v", resMap["static"])
		}
	})
}
