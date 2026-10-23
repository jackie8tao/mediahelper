package nfo

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

func TestGenerateMovieNFO(t *testing.T) {
	meta := &scraper.MediaMetadata{
		TMDBID:        872585,
		IMDbID:        "tt15398776",
		MediaType:     parser.MediaTypeMovie,
		Title:         "奥本海默",
		OriginalTitle: "Oppenheimer",
		Year:          2023,
		Premiered:     "2023-07-21",
		Plot:          "关于美国理论物理学家罗伯特·奥本海默的故事...",
		Rating:        8.1,
		Votes:         8900,
		Genres:        []string{"剧情", "历史", "传记"},
		Studios:       []string{"Universal Pictures"},
		Directors:     []string{"克里斯托弗·诺兰"},
		Actors: []scraper.Actor{
			{Name: "基里安·墨菲", Role: "J. Robert Oppenheimer", Thumb: "https://example.com/cillian.jpg"},
		},
	}

	data, err := GenerateMovieNFO(meta)
	if err != nil {
		t.Fatalf("GenerateMovieNFO failed: %v", err)
	}

	xmlStr := string(data)
	if !strings.HasPrefix(xmlStr, xml.Header) {
		t.Errorf("expected xml header in nfo output")
	}

	var parsed MovieNFO
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal generated NFO failed: %v", err)
	}

	if parsed.Title != "奥本海默" || parsed.OriginalTitle != "Oppenheimer" {
		t.Errorf("parsed title mismatch: %+v", parsed)
	}
	if parsed.TMDBID != 872585 || parsed.ID != "tt15398776" {
		t.Errorf("parsed IDs mismatch: %+v", parsed)
	}
	if len(parsed.Actors) != 1 || parsed.Actors[0].Name != "基里安·墨菲" {
		t.Errorf("parsed actors mismatch: %+v", parsed.Actors)
	}
	if len(parsed.Genres) != 3 {
		t.Errorf("parsed genres mismatch: %+v", parsed.Genres)
	}
}

func TestGenerateTVShowNFO(t *testing.T) {
	meta := &scraper.MediaMetadata{
		TMDBID:        1396,
		IMDbID:        "tt0903747",
		MediaType:     parser.MediaTypeTV,
		Title:         "绝命毒师",
		OriginalTitle: "Breaking Bad",
		Year:          2008,
		Premiered:     "2008-01-20",
		Plot:          "化学老师老白制毒逆袭的故事...",
		Rating:        8.9,
		Votes:         14000,
		Genres:        []string{"剧情", "犯罪"},
		Studios:       []string{"AMC"},
		Actors: []scraper.Actor{
			{Name: "布莱恩·科兰斯顿", Role: "Walter White"},
		},
	}

	data, err := GenerateTVShowNFO(meta)
	if err != nil {
		t.Fatalf("GenerateTVShowNFO failed: %v", err)
	}

	var parsed TVShowNFO
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal generated TVShow NFO failed: %v", err)
	}

	if parsed.Title != "绝命毒师" || parsed.Year != 2008 {
		t.Errorf("parsed TV show mismatch: %+v", parsed)
	}
}

func TestGenerateEpisodeNFO(t *testing.T) {
	meta := &scraper.MediaMetadata{
		TMDBID:        1396,
		MediaType:     parser.MediaTypeTV,
		Title:         "绝命毒师",
		Season:        1,
		Episode:       1,
		EpisodeTitle:  "首播集",
		EpisodePlot:   "身患绝症的高中化学老师决定与昔日学生联手...",
		EpisodeRating: 8.5,
		EpisodeAired:  "2008-01-20",
	}

	data, err := GenerateEpisodeNFO(meta)
	if err != nil {
		t.Fatalf("GenerateEpisodeNFO failed: %v", err)
	}

	var parsed EpisodeNFO
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal generated Episode NFO failed: %v", err)
	}

	if parsed.Title != "首播集" || parsed.Season != 1 || parsed.Episode != 1 {
		t.Errorf("parsed Episode mismatch: %+v", parsed)
	}
}
