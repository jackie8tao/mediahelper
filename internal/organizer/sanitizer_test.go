package organizer

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Mission: Impossible - Fallout", "Mission - Impossible - Fallout"},
		{"What If...?", "What If"},
		{"Star Wars: Episode IV - A New Hope", "Star Wars - Episode IV - A New Hope"},
		{`Movie <Name> With "Quotes" & |Pipes|`, "Movie Name With Quotes & Pipes"},
		{"Trailing Dot Movie....", "Trailing Dot Movie"},
		{"  Spaces   and   Tabs  ", "Spaces and Tabs"},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
