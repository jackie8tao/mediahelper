package scraper

import (
	"github.com/jackie8tao/mediahelper/internal/parser"
)

// Actor 演职人员信息
type Actor struct {
	Name  string `json:"name"`  // 姓名
	Role  string `json:"role"`  // 角色名称
	Thumb string `json:"thumb"` // 头像链接
}

// MediaMetadata 刮削出的标准多媒体元数据
type MediaMetadata struct {
	TMDBID        int              `json:"tmdb_id"`        // TMDB ID
	IMDbID        string           `json:"imdb_id"`        // IMDb ID (如 tt15398776)
	MediaType     parser.MediaType `json:"media_type"`     // 媒体类型: movie / tv / anime
	Title         string           `json:"title"`          // 官方标题 (优先中文)
	OriginalTitle string           `json:"original_title"` // 原语种标题
	SortTitle     string           `json:"sort_title"`     // 排序标题
	Year          int              `json:"year"`           // 上映年份
	Premiered     string           `json:"premiered"`      // 首映日期 (YYYY-MM-DD)
	Plot          string           `json:"plot"`           // 剧情梗概 (Overview)
	Rating        float64          `json:"rating"`         // 评分 (0.0 - 10.0)
	Votes         int              `json:"votes"`          // 评分投票数
	Genres        []string         `json:"genres"`         // 流派分类标签
	Studios       []string         `json:"studios"`        // 出品/制作制片厂
	Directors     []string         `json:"directors"`      // 导演列表
	Actors        []Actor          `json:"actors"`         // 主演列表
	PosterURL     string           `json:"poster_url"`     // 海报图片绝对地址
	BackdropURL   string           `json:"backdrop_url"`   // 背景图/Fanart 绝对地址

	// 针对电视剧分集 (Episode) 的特有字段
	Season         int     `json:"season"`          // 季度编号
	Episode        int     `json:"episode"`         // 分集编号
	EpisodeTitle   string  `json:"episode_title"`   // 单集标题
	EpisodePlot    string  `json:"episode_plot"`    // 单集剧情
	EpisodeRating  float64 `json:"episode_rating"`  // 单集评分
	EpisodeAired   string  `json:"episode_aired"`   // 单集播出日期
	EpisodeThumb   string  `json:"episode_thumb"`   // 单集截图封面
}
