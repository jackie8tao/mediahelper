package parser

import (
	"regexp"
)

// 文件扩展名清理正则
var (
	reFileExtension = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|ts|iso|rmvb|wmv|mov|flv|m2ts)$`)
)

// 发布组/字幕组正则
var (
	// 前置字幕组，如 [ANi], [Lilith-Raws], 【诸神字幕组】
	rePrefixGroup = regexp.MustCompile(`^(?:\[|【)([^\]】]+)(?:\]|】)\s*`)
	// 尾部发布组，如 -FLUX, -DEMAND, -HDH, @PTH
	reSuffixGroup = regexp.MustCompile(`[-@]([a-zA-Z0-9]+(?:@[a-zA-Z0-9]+)?)$`)
)

// 季集匹配正则
var (
	// 标准欧美格式: S01E02, S1E2, S01E01-E02, S01E01-02
	reSeasonEpisode = regexp.MustCompile(`(?i)\bS(\d{1,2})\s*[-_.]?\s*E(\d{1,3})(?:[-_.]?E?(\d{1,3}))?\b`)

	// 单季标识 (季包): S01, S1-S3, Season 1
	reSeasonPack = regexp.MustCompile(`(?i)\bS(\d{1,2})(?:\s*[-~_]\s*S?(\d{1,2}))?\b`)
	reSeasonWord = regexp.MustCompile(`(?i)\bSeason\s*(\d{1,2})\b`)

	// 中文季: 第一季, 第2季
	reChineseSeason = regexp.MustCompile(`第\s*([0-9零一二三四五六七八九十]+)\s*季`)

	// 中文集: 第01集, 第1-2集, 第1至2集
	reChineseEpisode = regexp.MustCompile(`第\s*([0-9]+)(?:\s*[-~至到]\s*([0-9]+))?\s*集`)

	// 动漫 / EP 格式: EP01, EP.05, EP01-02
	reEpFormat = regexp.MustCompile(`(?i)\bEP?\.?\s*(\d{1,3})(?:[-~至到]\s*(\d{1,3}))?\b`)

	// 动漫连字符集数格式: " - 28 ", " - 01v2"
	reAnimeDashEpisode = regexp.MustCompile(`\s+-\s+(\d{1,3})(?:v\d)?(?:\s+|$)`)

	// 动漫方括号独立数字集数: "[28]", "[01]"
	reAnimeBracketEpisode = regexp.MustCompile(`\[(\d{1,3})\]`)

	// 全集标识
	reComplete = regexp.MustCompile(`(?i)\b(Complete|全集)\b`)
)

// 年份匹配正则 (1900 - 2099)
var (
	reYear = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
)

// 技术规格正则映射表
type specPattern struct {
	re    *regexp.Regexp
	value string
}

var resolutionPatterns = []specPattern{
	{regexp.MustCompile(`(?i)\b(2160p|4k|uhd)\b`), "4K"},
	{regexp.MustCompile(`(?i)\b(1080p|1080i|fhd)\b`), "1080p"},
	{regexp.MustCompile(`(?i)\b(720p|hd)\b`), "720p"},
	{regexp.MustCompile(`(?i)\b(576p|480p|sd)\b`), "480p"},
}

var sourcePatterns = []specPattern{
	{regexp.MustCompile(`(?i)\b(Remux)\b`), "Remux"},
	{regexp.MustCompile(`(?i)\b(BluRay|Blu-Ray|BDRip|BRRip)\b`), "BluRay"},
	{regexp.MustCompile(`(?i)\b(WEB-?DL|WEBRip|WEB)\b`), "WEB-DL"},
	{regexp.MustCompile(`(?i)\b(HDTV|HDTVRip)\b`), "HDTV"},
	{regexp.MustCompile(`(?i)\b(DVDRip|DVD)\b`), "DVD"},
}

var videoCodecPatterns = []specPattern{
	{regexp.MustCompile(`(?i)\b(x265|h265|hevc|h\.265)\b`), "HEVC"},
	{regexp.MustCompile(`(?i)\b(x264|h264|avc|h\.264)\b`), "H.264"},
	{regexp.MustCompile(`(?i)\b(av1)\b`), "AV1"},
	{regexp.MustCompile(`(?i)\b(vc-?1)\b`), "VC-1"},
}

var audioCodecPatterns = []specPattern{
	{regexp.MustCompile(`(?i)\b(TrueHD(?:\s*Atmos)?|Atmos)\b`), "Atmos"},
	{regexp.MustCompile(`(?i)\b(DTS-HD(?:\s*MA)?)\b`), "DTS-HD MA"},
	{regexp.MustCompile(`(?i)\b(DTS)\b`), "DTS"},
	{regexp.MustCompile(`(?i)\b(DDP\s*5\.1|EAC3|DD\+|Dolby\s*Digital\s*Plus)\b`), "DDP5.1"},
	{regexp.MustCompile(`(?i)\b(AC3|DD5\.1)\b`), "AC3"},
	{regexp.MustCompile(`(?i)\b(FLAC)\b`), "FLAC"},
	{regexp.MustCompile(`(?i)\b(AAC(?:\s*2\.0)?)\b`), "AAC"},
}

// 常见噪点修饰词清理正则
var reNoiseWords = regexp.MustCompile(`(?i)\b(10bit|8bit|HDR10\+?|HDR|DoVi|Dolby\s*Vision|PROPER|REPACK|EXTENDED|Unrated|IMAX|Baha|B-Global|CHT|CHS|GB|BIG5|简繁|繁体|简体|双语|国粤|国语|粤语|中字|特效|字幕|未删减版?|无字)\b`)
