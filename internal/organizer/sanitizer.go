package organizer

import (
	"strings"
)

// SanitizeFilename 清理文件名与目录名中对操作系统非法的字符 (兼容 Windows/Linux/macOS)
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// 冒号替换为 " - "，例如 "Mission: Impossible" -> "Mission - Impossible"
	name = strings.ReplaceAll(name, ":", " - ")

	// 替换其他在 Windows / Unix 中非法的字符: < > " / \ | ? *
	illegalChars := []string{"<", ">", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range illegalChars {
		name = strings.ReplaceAll(name, char, "")
	}

	// 连续多空格缩减为一个空格
	fields := strings.Fields(name)
	name = strings.Join(fields, " ")

	// Windows 系统禁止目录或文件名以点号或空格结尾
	name = strings.TrimRight(name, ". ")

	return name
}
