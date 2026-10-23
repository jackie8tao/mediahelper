package nfo

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// XMLActor 演职人员 XML 节点
type XMLActor struct {
	Name  string `xml:"name"`
	Role  string `xml:"role,omitempty"`
	Thumb string `xml:"thumb,omitempty"`
}

// XMLUniqueID 唯一标识符节点
type XMLUniqueID struct {
	Type    string `xml:"type,attr"`
	Default bool   `xml:"default,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// MovieNFO 电影 NFO 结构
type MovieNFO struct {
	XMLName       xml.Name      `xml:"movie"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	SortTitle     string        `xml:"sorttitle,omitempty"`
	Rating        string        `xml:"rating,omitempty"`
	Votes         string        `xml:"votes,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Outline       string        `xml:"outline,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	UniqueID      []XMLUniqueID `xml:"uniqueid"`
	ID            string        `xml:"id,omitempty"`
	TMDBID        int           `xml:"tmdbid,omitempty"`
	Genres        []string      `xml:"genre,omitempty"`
	Studios       []string      `xml:"studio,omitempty"`
	Directors     []string      `xml:"director,omitempty"`
	Actors        []XMLActor    `xml:"actor,omitempty"`
}

// TVShowNFO 电视剧 NFO 结构 (tvshow.nfo)
type TVShowNFO struct {
	XMLName       xml.Name      `xml:"tvshow"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	Rating        string        `xml:"rating,omitempty"`
	Votes         string        `xml:"votes,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	UniqueID      []XMLUniqueID `xml:"uniqueid"`
	ID            string        `xml:"id,omitempty"`
	TMDBID        int           `xml:"tmdbid,omitempty"`
	Genres        []string      `xml:"genre,omitempty"`
	Studios       []string      `xml:"studio,omitempty"`
	Actors        []XMLActor    `xml:"actor,omitempty"`
}

// EpisodeNFO 电视剧单集 NFO 结构 (episodedetails)
type EpisodeNFO struct {
	XMLName   xml.Name      `xml:"episodedetails"`
	Title     string        `xml:"title"`
	ShowTitle string        `xml:"showtitle"`
	Season    int           `xml:"season"`
	Episode   int           `xml:"episode"`
	Plot      string        `xml:"plot,omitempty"`
	Aired     string        `xml:"aired,omitempty"`
	Rating    string        `xml:"rating,omitempty"`
	UniqueID  []XMLUniqueID `xml:"uniqueid,omitempty"`
}

// GenerateMovieNFO 生成电影 NFO XML 数据
func GenerateMovieNFO(meta *scraper.MediaMetadata) ([]byte, error) {
	if meta == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	nfo := MovieNFO{
		Title:         meta.Title,
		OriginalTitle: meta.OriginalTitle,
		SortTitle:     meta.SortTitle,
		Year:          meta.Year,
		Plot:          meta.Plot,
		Outline:       meta.Plot,
		Premiered:     meta.Premiered,
		ID:            meta.IMDbID,
		TMDBID:        meta.TMDBID,
		Genres:        meta.Genres,
		Studios:       meta.Studios,
		Directors:     meta.Directors,
	}

	if meta.Rating > 0 {
		nfo.Rating = fmt.Sprintf("%.1f", meta.Rating)
	}
	if meta.Votes > 0 {
		nfo.Votes = strconv.Itoa(meta.Votes)
	}

	if meta.TMDBID > 0 {
		nfo.UniqueID = append(nfo.UniqueID, XMLUniqueID{
			Type:    "tmdb",
			Default: true,
			Value:   strconv.Itoa(meta.TMDBID),
		})
	}
	if meta.IMDbID != "" {
		nfo.UniqueID = append(nfo.UniqueID, XMLUniqueID{
			Type:  "imdb",
			Value: meta.IMDbID,
		})
	}

	for _, a := range meta.Actors {
		nfo.Actors = append(nfo.Actors, XMLActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	data, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}

// GenerateTVShowNFO 生成电视剧主 NFO XML 数据
func GenerateTVShowNFO(meta *scraper.MediaMetadata) ([]byte, error) {
	if meta == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	nfo := TVShowNFO{
		Title:         meta.Title,
		OriginalTitle: meta.OriginalTitle,
		Year:          meta.Year,
		Plot:          meta.Plot,
		Premiered:     meta.Premiered,
		ID:            meta.IMDbID,
		TMDBID:        meta.TMDBID,
		Genres:        meta.Genres,
		Studios:       meta.Studios,
	}

	if meta.Rating > 0 {
		nfo.Rating = fmt.Sprintf("%.1f", meta.Rating)
	}
	if meta.Votes > 0 {
		nfo.Votes = strconv.Itoa(meta.Votes)
	}

	if meta.TMDBID > 0 {
		nfo.UniqueID = append(nfo.UniqueID, XMLUniqueID{
			Type:    "tmdb",
			Default: true,
			Value:   strconv.Itoa(meta.TMDBID),
		})
	}
	if meta.IMDbID != "" {
		nfo.UniqueID = append(nfo.UniqueID, XMLUniqueID{
			Type:  "imdb",
			Value: meta.IMDbID,
		})
	}

	for _, a := range meta.Actors {
		nfo.Actors = append(nfo.Actors, XMLActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	data, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}

// GenerateEpisodeNFO 生成单集分集 NFO XML 数据
func GenerateEpisodeNFO(meta *scraper.MediaMetadata) ([]byte, error) {
	if meta == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	title := meta.EpisodeTitle
	if title == "" {
		title = fmt.Sprintf("第 %d 集", meta.Episode)
	}

	nfo := EpisodeNFO{
		Title:     title,
		ShowTitle: meta.Title,
		Season:    meta.Season,
		Episode:   meta.Episode,
		Plot:      meta.EpisodePlot,
		Aired:     meta.EpisodeAired,
	}

	if meta.EpisodeRating > 0 {
		nfo.Rating = fmt.Sprintf("%.1f", meta.EpisodeRating)
	}

	if meta.TMDBID > 0 {
		nfo.UniqueID = append(nfo.UniqueID, XMLUniqueID{
			Type:  "tmdb",
			Value: strconv.Itoa(meta.TMDBID),
		})
	}

	data, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}
