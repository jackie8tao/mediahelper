package downloader

import (
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
)

func TestPathMapper_ToLocal(t *testing.T) {
	tests := []struct {
		name       string
		mappings   []config.PathMapping
		remotePath string
		expected   string
	}{
		{
			name: "Unix to Windows path mapping",
			mappings: []config.PathMapping{
				{
					Remote: "/downloads",
					Local:  `D:\Media\Downloads`,
				},
			},
			remotePath: "/downloads/movies/Inception (2010)/Inception.mkv",
			expected:   filepath.FromSlash("D:/Media/Downloads/movies/Inception (2010)/Inception.mkv"),
		},
		{
			name: "Prefix boundary safety (should not match /downloads2)",
			mappings: []config.PathMapping{
				{
					Remote: "/downloads",
					Local:  `D:\Downloads`,
				},
			},
			remotePath: "/downloads2/movie.mkv",
			expected:   filepath.FromSlash("/downloads2/movie.mkv"),
		},
		{
			name: "Windows to Unix mapping with trailing slash",
			mappings: []config.PathMapping{
				{
					Remote: `C:\Downloads\`,
					Local:  "/media/downloads/",
				},
			},
			remotePath: `C:\Downloads\series\S01E01.mp4`,
			expected:   filepath.FromSlash("/media/downloads/series/S01E01.mp4"),
		},
		{
			name:       "Empty mapping returns original clean path",
			mappings:   []config.PathMapping{},
			remotePath: "/media/downloads/file.mkv",
			expected:   filepath.FromSlash("/media/downloads/file.mkv"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewPathMapper(tt.mappings)
			result := mapper.ToLocal(tt.remotePath)
			if result != tt.expected {
				t.Errorf("ToLocal() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
