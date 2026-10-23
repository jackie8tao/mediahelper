package parser

import (
	"strconv"
	"strings"
)

// Parse 解析媒体文件名或种子名称，输出结构化的 MediaInfo
func Parse(raw string) *MediaInfo {
	info := &MediaInfo{
		OriginalText: raw,
		MediaType:    MediaTypeMovie, // 默认电影，后续检测到季集特征调整为 tv/anime
	}

	s := strings.TrimSpace(raw)
	if s == "" {
		return info
	}

	// 1. 移除常见视频文件后缀
	s = reFileExtension.ReplaceAllString(s, "")

	// 2. 提取并剥离前置发布组/字幕组 (如 [ANi], 【诸神字幕组】)
	if matches := rePrefixGroup.FindStringSubmatch(s); len(matches) > 1 {
		info.ReleaseGroup = strings.TrimSpace(matches[1])
		s = s[len(matches[0]):]
	}

	// 3. 提取并剥离尾部发布组 (如 -FLUX, -DEMAND, -HDH, @PTH)
	if matches := reSuffixGroup.FindStringSubmatch(s); len(matches) > 1 {
		if info.ReleaseGroup == "" {
			info.ReleaseGroup = strings.TrimSpace(matches[1])
		}
		s = s[:len(s)-len(matches[0])]
	}

	// 记录所有锚点的最小起始下标，用于最终截断片名
	minAnchorIndex := len(s)
	updateMinAnchor := func(loc []int) {
		if len(loc) >= 2 && loc[0] >= 0 && loc[0] < minAnchorIndex {
			minAnchorIndex = loc[0]
		}
	}

	// 4. 提取技术规格 (分辨率、媒介、音视频编码)
	for _, p := range resolutionPatterns {
		if loc := p.re.FindStringIndex(s); len(loc) > 0 {
			info.Resolution = p.value
			updateMinAnchor(loc)
			break
		}
	}

	for _, p := range sourcePatterns {
		if loc := p.re.FindStringIndex(s); len(loc) > 0 {
			info.Source = p.value
			updateMinAnchor(loc)
			break
		}
	}

	for _, p := range videoCodecPatterns {
		if loc := p.re.FindStringIndex(s); len(loc) > 0 {
			info.VideoCodec = p.value
			updateMinAnchor(loc)
			break
		}
	}

	for _, p := range audioCodecPatterns {
		if loc := p.re.FindStringIndex(s); len(loc) > 0 {
			info.AudioCodec = p.value
			updateMinAnchor(loc)
			break
		}
	}

	// 5. 提取季集信息 (Season / Episode)
	extractSeasonEpisode(s, info, updateMinAnchor)

	// 6. 提取年份 (注意规避片名中的年份如 1917)
	extractYear(s, info, updateMinAnchor)

	// 7. 截取片名候选区
	var rawTitle string
	if minAnchorIndex > 0 && minAnchorIndex < len(s) {
		rawTitle = s[:minAnchorIndex]
	} else {
		rawTitle = s
	}

	// 8. 清洗片名并分离中英文标题
	cleanAndAssignTitles(rawTitle, info)

	return info
}

// extractSeasonEpisode 提取季数与集数信息
func extractSeasonEpisode(s string, info *MediaInfo, updateMinAnchor func([]int)) {
	// A. 标准欧美格式: S01E02, S1E2, S01E01-E02
	if matches := reSeasonEpisode.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		updateMinAnchor(matches[0:2])

		seasonStr := s[matches[2]:matches[3]]
		episodeStr := s[matches[4]:matches[5]]
		info.Season, _ = strconv.Atoi(seasonStr)
		info.Episode, _ = strconv.Atoi(episodeStr)

		if len(matches) >= 8 && matches[6] >= 0 {
			endStr := s[matches[6]:matches[7]]
			info.EpisodeEnd, _ = strconv.Atoi(endStr)
		}
		return
	}

	// B. 中文季: "第二季", "第1季"
	if matches := reChineseSeason.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		updateMinAnchor(matches[0:2])
		info.Season = parseChineseNumber(s[matches[2]:matches[3]])
	}

	// C. 中文集: "第05集", "第1-2集"
	if matches := reChineseEpisode.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		updateMinAnchor(matches[0:2])
		info.Episode, _ = strconv.Atoi(s[matches[2]:matches[3]])
		if len(matches) >= 6 && matches[4] >= 0 {
			info.EpisodeEnd, _ = strconv.Atoi(s[matches[4]:matches[5]])
		}
		if info.Season == 0 {
			info.Season = 1
		}
		return
	}

	// D. 动漫/通用 EP 格式: "EP01", "EP.05"
	if matches := reEpFormat.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		updateMinAnchor(matches[0:2])
		info.Episode, _ = strconv.Atoi(s[matches[2]:matches[3]])
		if len(matches) >= 6 && matches[4] >= 0 {
			info.EpisodeEnd, _ = strconv.Atoi(s[matches[4]:matches[5]])
		}
		if info.Season == 0 {
			info.Season = 1
		}
		return
	}

	// E. 动漫连字符格式: " - 28 "
	if matches := reAnimeDashEpisode.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeAnime
		updateMinAnchor(matches[0:2])
		info.Season = 1
		info.Episode, _ = strconv.Atoi(s[matches[2]:matches[3]])
		return
	}

	// F. 动漫方括号独立集数: "[28]"
	if matches := reAnimeBracketEpisode.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeAnime
		updateMinAnchor(matches[0:2])
		info.Season = 1
		info.Episode, _ = strconv.Atoi(s[matches[2]:matches[3]])
		return
	}

	// G. 全集 / 季包检测
	if loc := reComplete.FindStringIndex(s); len(loc) > 0 {
		info.MediaType = MediaTypeTV
		info.IsSeasonPack = true
		updateMinAnchor(loc)
	}

	if matches := reSeasonPack.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		info.IsSeasonPack = true
		updateMinAnchor(matches[0:2])
		if info.Season == 0 {
			info.Season, _ = strconv.Atoi(s[matches[2]:matches[3]])
		}
		return
	}

	if matches := reSeasonWord.FindStringSubmatchIndex(s); len(matches) > 0 {
		info.MediaType = MediaTypeTV
		info.IsSeasonPack = true
		updateMinAnchor(matches[0:2])
		if info.Season == 0 {
			info.Season, _ = strconv.Atoi(s[matches[2]:matches[3]])
		}
		return
	}
}

// extractYear 提取上映年份
func extractYear(s string, info *MediaInfo, updateMinAnchor func([]int)) {
	locs := reYear.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return
	}

	// 如果只有一个年份，直接提取
	if len(locs) == 1 {
		year, _ := strconv.Atoi(s[locs[0][0]:locs[0][1]])
		info.Year = year
		updateMinAnchor(locs[0])
		return
	}

	// 如果有多个类似年份 (例如片名本身含年份 "1917.2019.1080p")
	// 第一个在起始处往往是片名，第二个才是上映年份
	chosen := locs[0]
	if locs[0][0] <= 1 && len(locs) > 1 {
		chosen = locs[1]
	}

	year, _ := strconv.Atoi(s[chosen[0]:chosen[1]])
	info.Year = year
	updateMinAnchor(chosen)
}

// cleanAndAssignTitles 对截取的片名进行清洗、去噪并拆解中英标题
func cleanAndAssignTitles(raw string, info *MediaInfo) {
	// 符号规范化
	cleaned := normalizeSeparators(raw)

	// 剔除中文字幕、10bit、特效等干扰词
	cleaned = reNoiseWords.ReplaceAllString(cleaned, " ")

	// 清理多余标点与括号
	cleaned = strings.Trim(cleaned, " -_/|[]【】()（）")
	cleaned = cleanSpaces(cleaned)

	if cleaned == "" {
		return
	}

	// 尝试分离中文与英文标题
	cnTitle, enTitle := separateTitles(cleaned)

	info.CnTitle = cnTitle
	info.EnTitle = enTitle

	// 优先设置中文名，若无则使用英文或全量清洗名
	if cnTitle != "" {
		info.Title = cnTitle
	} else if enTitle != "" {
		info.Title = enTitle
	} else {
		info.Title = cleaned
	}
}
