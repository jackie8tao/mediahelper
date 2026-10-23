package parser

// MediaType 媒体资源类型
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
)

// MediaInfo 解析出的媒体结构化元数据
type MediaInfo struct {
	OriginalText string    `json:"original_text"` // 原始输入文件名/种子标题
	MediaType    MediaType `json:"media_type"`    // 媒体类型: movie / tv / anime

	// 标题信息
	Title   string `json:"title"`    // 主标题 (优先中文，无中文则为英文)
	CnTitle string `json:"cn_title"` // 纯净中文片名 (如有)
	EnTitle string `json:"en_title"` // 纯净英文片名 (如有)
	Year    int    `json:"year"`     // 上映年份 (如 2023，未能识别则为 0)

	// 季集信息 (针对剧集 / 动漫)
	Season       int  `json:"season"`         // 季度 (电影为 0，剧集默认 1)
	Episode      int  `json:"episode"`        // 集数 (电影为 0)
	EpisodeEnd   int  `json:"episode_end"`    // 多集截止数 (如 E01-E02 时为 2)
	IsSeasonPack bool `json:"is_season_pack"` // 是否为整季包 (如 S01-S03, Complete)

	// 技术规格元数据
	Resolution   string `json:"resolution"`    // 4K, 1080p, 720p, 480p
	Source       string `json:"source"`        // BluRay, WEB-DL, HDTV, Remux, DVD
	VideoCodec   string `json:"video_codec"`   // HEVC, H.264, AV1, VC-1
	AudioCodec   string `json:"audio_codec"`   // Atmos, TrueHD, DTS-HD MA, DTS, DDP5.1, AAC, FLAC
	ReleaseGroup string `json:"release_group"` // 发布组/字幕组 (如 FLUX, CHD, ANi, Lilith-Raws)
}
