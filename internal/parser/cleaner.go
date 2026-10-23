package parser

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// normalizeSeparators 将点号、下划线、加号等无意义符号转为空格
func normalizeSeparators(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '.', '_', '+':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return cleanSpaces(b.String())
}

// cleanSpaces 清理首尾与多余空格
func cleanSpaces(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	return strings.Join(parts, " ")
}

// containsHan 判断字符串中是否包含汉字
func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// parseChineseNumber 将中文数字(一到十)转换为阿拉伯数字
func parseChineseNumber(s string) int {
	s = strings.TrimSpace(s)
	// 尝试直接转数字
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}

	chineseMap := map[string]int{
		"零": 0, "一": 1, "二": 2, "两": 2, "三": 3,
		"四": 4, "五": 5, "六": 6, "七": 7, "八": 8,
		"九": 9, "十": 10,
	}

	if val, ok := chineseMap[s]; ok {
		return val
	}

	// 处理十开头的两位数 (如 十一, 十二)
	if strings.HasPrefix(s, "十") && len([]rune(s)) == 2 {
		r := []rune(s)
		return 10 + chineseMap[string(r[1])]
	}

	return 0
}

var reBilingualSplit = regexp.MustCompile(`^([\p{Han}0-9\s·:：\-]+?)\s+([a-zA-Z0-9\s:.'’\-]+)$`)

// separateTitles 尝试从中英混排片名中分离纯中文与纯英文标题
func separateTitles(title string) (cnTitle string, enTitle string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", ""
	}

	// 1. 如果包含显式斜杠分隔符 (如 "咒术回战 / Jujutsu Kaisen" 或 "繁花 | Blossoms Shanghai")
	for _, sep := range []string{"/", "|"} {
		if strings.Contains(title, sep) {
			parts := strings.Split(title, sep)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if containsHan(p) && cnTitle == "" {
					cnTitle = p
				} else if !containsHan(p) && enTitle == "" {
					enTitle = p
				}
			}
			if cnTitle != "" || enTitle != "" {
				return cnTitle, enTitle
			}
		}
	}

	// 2. 正则拆分：中文片名在前，英文片名在后 (如 "繁花 Blossoms Shanghai", "流浪地球2 The Wandering Earth II")
	if matches := reBilingualSplit.FindStringSubmatch(title); len(matches) == 3 {
		part1 := strings.TrimSpace(matches[1])
		part2 := strings.TrimSpace(matches[2])
		if containsHan(part1) && !containsHan(part2) {
			return part1, part2
		}
	}

	// 3. 只有单一语言
	if containsHan(title) {
		return title, ""
	}
	return "", title
}
