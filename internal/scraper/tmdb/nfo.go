package tmdb

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// buildMovieNFO generates XML bytes for a movie.nfo file.
func buildMovieNFO(m *scraper.MediaMetadata) ([]byte, error) {
	nfo := movieNFO{
		Title:         m.Title,
		OriginalTitle: m.OriginalTitle,
		SortTitle:     m.SortTitle,
		Year:          m.Year,
		Premiered:     m.Premiered,
		Plot:          m.Plot,
		Rating:        m.Rating,
		Votes:         m.Votes,
		Genres:        m.Genres,
		Studios:       m.Studios,
		Directors:     m.Directors,
	}

	if m.TMDBID > 0 {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:    "tmdb",
			Default: "true",
			Value:   strconv.Itoa(m.TMDBID),
		})
	}
	if m.IMDbID != "" {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:  "imdb",
			Value: m.IMDbID,
		})
	}

	for _, a := range m.Actors {
		nfo.Actors = append(nfo.Actors, nfoActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	raw, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal movie.nfo error: %w", err)
	}

	header := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	return append(header, raw...), nil
}

// buildTVShowNFO generates XML bytes for a tvshow.nfo file.
func buildTVShowNFO(m *scraper.MediaMetadata) ([]byte, error) {
	nfo := tvshowNFO{
		Title:         m.Title,
		OriginalTitle: m.OriginalTitle,
		SortTitle:     m.SortTitle,
		Year:          m.Year,
		Premiered:     m.Premiered,
		Plot:          m.Plot,
		Rating:        m.Rating,
		Votes:         m.Votes,
		Genres:        m.Genres,
		Studios:       m.Studios,
	}

	if m.TMDBID > 0 {
		nfo.UniqueIDs = append(nfo.UniqueIDs, nfoUniqueID{
			Type:    "tmdb",
			Default: "true",
			Value:   strconv.Itoa(m.TMDBID),
		})
	}

	for _, a := range m.Actors {
		nfo.Actors = append(nfo.Actors, nfoActor{
			Name:  a.Name,
			Role:  a.Role,
			Thumb: a.Thumb,
		})
	}

	raw, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal tvshow.nfo error: %w", err)
	}

	header := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	return append(header, raw...), nil
}

// buildEpisodeNFO generates XML bytes for a single episode NFO file.
func buildEpisodeNFO(ep *scraper.EpisodeMetadata) ([]byte, error) {
	nfo := episodeNFO{
		Title:   ep.Title,
		Season:  ep.Season,
		Episode: ep.Episode,
		Plot:    ep.Plot,
		Rating:  ep.Rating,
		Aired:   ep.Aired,
	}

	raw, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal episode.nfo error: %w", err)
	}

	header := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	return append(header, raw...), nil
}

