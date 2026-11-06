package douban

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// NFO XML structures for Kodi / Jellyfin / Emby

type nfoUniqueID struct {
	XMLName xml.Name `xml:"uniqueid"`
	Type    string   `xml:"type,attr"`
	Default string   `xml:"default,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type nfoActor struct {
	XMLName xml.Name `xml:"actor"`
	Name    string   `xml:"name"`
	Role    string   `xml:"role,omitempty"`
	Thumb   string   `xml:"thumb,omitempty"`
}

type movieNFO struct {
	XMLName       xml.Name      `xml:"movie"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	SortTitle     string        `xml:"sorttitle,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Rating        float64       `xml:"rating,omitempty"`
	Votes         int           `xml:"votes,omitempty"`
	UniqueIDs     []nfoUniqueID `xml:"uniqueid"`
	Genres        []string      `xml:"genre,omitempty"`
	Countries     []string      `xml:"country,omitempty"`
	Directors     []string      `xml:"director,omitempty"`
	Actors        []nfoActor    `xml:"actor,omitempty"`
}

type tvshowNFO struct {
	XMLName       xml.Name      `xml:"tvshow"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	SortTitle     string        `xml:"sorttitle,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Rating        float64       `xml:"rating,omitempty"`
	Votes         int           `xml:"votes,omitempty"`
	UniqueIDs     []nfoUniqueID `xml:"uniqueid"`
	Genres        []string      `xml:"genre,omitempty"`
	Countries     []string      `xml:"country,omitempty"`
	Directors     []string      `xml:"director,omitempty"`
	Actors        []nfoActor    `xml:"actor,omitempty"`
}

func buildMovieNFO(meta *scraper.MediaMetadata) ([]byte, error) {
	nfo := movieNFO{
		Title:         meta.Title,
		OriginalTitle: meta.OriginalTitle,
		SortTitle:     meta.SortTitle,
		Year:          meta.Year,
		Premiered:     meta.Premiered,
		Plot:          meta.Plot,
		Rating:        meta.Rating,
		Votes:         meta.Votes,
		Genres:        meta.Genres,
		Directors:     meta.Directors,
	}

	if meta.DoubanID != "" {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:    "douban",
			Default: "true",
			Value:   meta.DoubanID,
		})
	}
	if meta.TMDBID > 0 {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:  "tmdb",
			Value: strconv.Itoa(meta.TMDBID),
		})
	}
	if meta.IMDbID != "" {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:  "imdb",
			Value: meta.IMDbID,
		})
	}

	for _, a := range meta.Actors {
		nfo.Actors = append(nfo.Actors, nfoActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	out, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, "%s\n%s\n", xml.Header, out), nil
}

func buildTVShowNFO(meta *scraper.MediaMetadata) ([]byte, error) {
	nfo := tvshowNFO{
		Title:         meta.Title,
		OriginalTitle: meta.OriginalTitle,
		SortTitle:     meta.SortTitle,
		Year:          meta.Year,
		Premiered:     meta.Premiered,
		Plot:          meta.Plot,
		Rating:        meta.Rating,
		Votes:         meta.Votes,
		Genres:        meta.Genres,
		Directors:     meta.Directors,
	}

	if meta.DoubanID != "" {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:    "douban",
			Default: "true",
			Value:   meta.DoubanID,
		})
	}
	if meta.TMDBID > 0 {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:  "tmdb",
			Value: strconv.Itoa(meta.TMDBID),
		})
	}
	if meta.IMDbID != "" {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:  "imdb",
			Value: meta.IMDbID,
		})
	}

	for _, a := range meta.Actors {
		nfo.Actors = append(nfo.Actors, nfoActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	out, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, "%s\n%s\n", xml.Header, out), nil
}
