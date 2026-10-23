package downloader

import (
	"path/filepath"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// PathMapper 路径映射处理器
type PathMapper struct {
	mappings []config.PathMapping
}

// NewPathMapper 创建路径映射器
func NewPathMapper(mappings []config.PathMapping) *PathMapper {
	return &PathMapper{mappings: mappings}
}

// ToLocal 将远程/下载器返回的路径转换为本地有效路径
func (m *PathMapper) ToLocal(remotePath string) string {
	if remotePath == "" || len(m.mappings) == 0 {
		return filepath.Clean(remotePath)
	}

	// 统一转为正斜杠便于前缀比对
	normalizedRemote := normalizeSlash(remotePath)

	for _, mapping := range m.mappings {
		if mapping.Remote == "" || mapping.Local == "" {
			continue
		}

		normalizedRuleRemote := normalizeSlash(mapping.Remote)
		// 确保以斜杠结尾做前缀匹配，避免 /data 误匹配 /database
		prefix := strings.TrimSuffix(normalizedRuleRemote, "/") + "/"

		if strings.HasPrefix(normalizedRemote, prefix) {
			rel := strings.TrimPrefix(normalizedRemote, prefix)
			// 本地目标路径拼接
			localBase := strings.TrimSuffix(normalizeSlash(mapping.Local), "/")
			combined := localBase + "/" + rel
			return filepath.FromSlash(filepath.Clean(combined))
		}

		// 精确匹配完全相等的情况
		if normalizedRemote == strings.TrimSuffix(normalizedRuleRemote, "/") {
			return filepath.FromSlash(filepath.Clean(mapping.Local))
		}
	}

	return filepath.FromSlash(filepath.Clean(remotePath))
}

// normalizeSlash 将 Windows 反斜杠转换为标准斜杠
func normalizeSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
