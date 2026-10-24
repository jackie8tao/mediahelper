package downloader

import (
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// PathMapper translates remote/docker container paths into local accessible paths.
type PathMapper struct {
	mappings []config.PathMapping
}

// NewPathMapper creates a new PathMapper instance.
func NewPathMapper(mappings []config.PathMapping) *PathMapper {
	return &PathMapper{mappings: mappings}
}

// ToLocal converts a remote/downloader path into a valid local filesystem path.
func (m *PathMapper) ToLocal(remotePath string) string {
	if remotePath == "" || len(m.mappings) == 0 {
		return filepath.Clean(remotePath)
	}

	// Normalize to forward slashes for prefix matching
	normalizedRemote := normalizeSlash(remotePath)

	for _, mapping := range m.mappings {
		if mapping.Remote == "" || mapping.Local == "" {
			continue
		}

		normalizedRuleRemote := normalizeSlash(mapping.Remote)
		// Ensure trailing slash for prefix matching to avoid /data matching /database
		prefix := strings.TrimSuffix(normalizedRuleRemote, "/") + "/"

		if strings.HasPrefix(normalizedRemote, prefix) {
			rel := strings.TrimPrefix(normalizedRemote, prefix)
			// Concatenate with local base path
			localBase := strings.TrimSuffix(normalizeSlash(mapping.Local), "/")
			combined := localBase + "/" + rel
			return filepath.FromSlash(filepath.Clean(combined))
		}

		// Exact match without trailing relative path
		if normalizedRemote == strings.TrimSuffix(normalizedRuleRemote, "/") {
			return filepath.FromSlash(filepath.Clean(mapping.Local))
		}
	}

	return filepath.FromSlash(filepath.Clean(remotePath))
}

// normalizeSlash converts Windows backslashes to standard forward slashes.
func normalizeSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
