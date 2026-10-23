package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MediaInfo
	}{
		{
			name:  "Standard Western Movie (Oppenheimer)",
			input: "Oppenheimer.2023.2160p.UHD.BluRay.x265.10bit.DTS-HD.MA.5.1-FLUX.mkv",
			expected: MediaInfo{
				MediaType:    MediaTypeMovie,
				Title:        "Oppenheimer",
				EnTitle:      "Oppenheimer",
				Year:         2023,
				Resolution:   "4K",
				Source:       "BluRay",
				VideoCodec:   "HEVC",
				AudioCodec:   "DTS-HD MA",
				ReleaseGroup: "FLUX",
			},
		},
		{
			name:  "Standard Western TV Show (Breaking Bad)",
			input: "Breaking.Bad.S01E01.Pilot.1080p.BluRay.x264-DEMAND.mkv",
			expected: MediaInfo{
				MediaType:    MediaTypeTV,
				Title:        "Breaking Bad",
				EnTitle:      "Breaking Bad",
				Season:       1,
				Episode:      1,
				Resolution:   "1080p",
				Source:       "BluRay",
				VideoCodec:   "H.264",
				ReleaseGroup: "DEMAND",
			},
		},
		{
			name:  "Domestic TV Series with Bilingual Title (Blossoms Shanghai S01 Complete)",
			input: "繁花.Blossoms.Shanghai.2023.S01.Complete.2160p.WEB-DL.H265.AAC-HDH",
			expected: MediaInfo{
				MediaType:    MediaTypeTV,
				Title:        "繁花",
				CnTitle:      "繁花",
				EnTitle:      "Blossoms Shanghai",
				Year:         2023,
				Season:       1,
				IsSeasonPack: true,
				Resolution:   "4K",
				Source:       "WEB-DL",
				VideoCodec:   "HEVC",
				AudioCodec:   "AAC",
				ReleaseGroup: "HDH",
			},
		},
		{
			name:  "Domestic TV Series Episode (The Knockout)",
			input: "狂飙.The.Knockout.2023.S01E15.2160p.WEB-DL.H265.AAC-PTH.mp4",
			expected: MediaInfo{
				MediaType:    MediaTypeTV,
				Title:        "狂飙",
				CnTitle:      "狂飙",
				EnTitle:      "The Knockout",
				Year:         2023,
				Season:       1,
				Episode:      15,
				Resolution:   "4K",
				Source:       "WEB-DL",
				VideoCodec:   "HEVC",
				AudioCodec:   "AAC",
				ReleaseGroup: "PTH",
			},
		},
		{
			name:  "Chinese Season and Episode Naming (Joy of Life)",
			input: "庆余年.第二季.第05集.2024.4k.mp4",
			expected: MediaInfo{
				MediaType:  MediaTypeTV,
				Title:      "庆余年",
				CnTitle:    "庆余年",
				Year:       2024,
				Season:     2,
				Episode:    5,
				Resolution: "4K",
			},
		},
		{
			name:  "Domestic Movie (The Wandering Earth II)",
			input: "流浪地球2.The.Wandering.Earth.II.2023.2160p.WEB-DL.x265.DDP5.1.Atmos-HDS.mkv",
			expected: MediaInfo{
				MediaType:    MediaTypeMovie,
				Title:        "流浪地球2",
				CnTitle:      "流浪地球2",
				EnTitle:      "The Wandering Earth II",
				Year:         2023,
				Resolution:   "4K",
				Source:       "WEB-DL",
				VideoCodec:   "HEVC",
				AudioCodec:   "Atmos",
				ReleaseGroup: "HDS",
			},
		},
		{
			name:  "Anime Fansub Release (Frieren)",
			input: "[ANi] 葬送的芙莉莲 - 28 [1080P][Baha][WEB-DL][AAC][CHT].mp4",
			expected: MediaInfo{
				MediaType:    MediaTypeAnime,
				Title:        "葬送的芙莉莲",
				CnTitle:      "葬送的芙莉莲",
				Season:       1,
				Episode:      28,
				Resolution:   "1080p",
				Source:       "WEB-DL",
				AudioCodec:   "AAC",
				ReleaseGroup: "ANi",
			},
		},
		{
			name:  "Anime with Slash Separated Titles (Jujutsu Kaisen)",
			input: "[Lilith-Raws] 咒术回战 / Jujutsu Kaisen - 47 [Baha][WEB-DL][1080p][AVC AAC].mp4",
			expected: MediaInfo{
				MediaType:    MediaTypeAnime,
				Title:        "咒术回战",
				CnTitle:      "咒术回战",
				EnTitle:      "Jujutsu Kaisen",
				Season:       1,
				Episode:      47,
				Resolution:   "1080p",
				Source:       "WEB-DL",
				AudioCodec:   "AAC",
				ReleaseGroup: "Lilith-Raws",
			},
		},
		{
			name:  "TV Show Multi-Episode (Stranger Things S04E01-E02)",
			input: "Stranger.Things.S04E01-E02.2160p.WEB-DL.x265",
			expected: MediaInfo{
				MediaType:  MediaTypeTV,
				Title:      "Stranger Things",
				EnTitle:    "Stranger Things",
				Season:     4,
				Episode:    1,
				EpisodeEnd: 2,
				Resolution: "4K",
				Source:     "WEB-DL",
				VideoCodec: "HEVC",
			},
		},
		{
			name:  "Movie Title with Year (1917)",
			input: "1917.2019.1080p.BluRay.x264-SPARKS.mkv",
			expected: MediaInfo{
				MediaType:    MediaTypeMovie,
				Title:        "1917",
				Year:         2019,
				Resolution:   "1080p",
				Source:       "BluRay",
				VideoCodec:   "H.264",
				ReleaseGroup: "SPARKS",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)

			if got.MediaType != tt.expected.MediaType {
				t.Errorf("MediaType mismatch: got %v, want %v", got.MediaType, tt.expected.MediaType)
			}
			if got.Title != tt.expected.Title {
				t.Errorf("Title mismatch: got %q, want %q", got.Title, tt.expected.Title)
			}
			if tt.expected.CnTitle != "" && got.CnTitle != tt.expected.CnTitle {
				t.Errorf("CnTitle mismatch: got %q, want %q", got.CnTitle, tt.expected.CnTitle)
			}
			if tt.expected.EnTitle != "" && got.EnTitle != tt.expected.EnTitle {
				t.Errorf("EnTitle mismatch: got %q, want %q", got.EnTitle, tt.expected.EnTitle)
			}
			if tt.expected.Year != 0 && got.Year != tt.expected.Year {
				t.Errorf("Year mismatch: got %v, want %v", got.Year, tt.expected.Year)
			}
			if tt.expected.Season != 0 && got.Season != tt.expected.Season {
				t.Errorf("Season mismatch: got %v, want %v", got.Season, tt.expected.Season)
			}
			if tt.expected.Episode != 0 && got.Episode != tt.expected.Episode {
				t.Errorf("Episode mismatch: got %v, want %v", got.Episode, tt.expected.Episode)
			}
			if tt.expected.EpisodeEnd != 0 && got.EpisodeEnd != tt.expected.EpisodeEnd {
				t.Errorf("EpisodeEnd mismatch: got %v, want %v", got.EpisodeEnd, tt.expected.EpisodeEnd)
			}
			if tt.expected.IsSeasonPack && !got.IsSeasonPack {
				t.Errorf("IsSeasonPack mismatch: got false, want true")
			}
			if tt.expected.Resolution != "" && got.Resolution != tt.expected.Resolution {
				t.Errorf("Resolution mismatch: got %v, want %v", got.Resolution, tt.expected.Resolution)
			}
			if tt.expected.Source != "" && got.Source != tt.expected.Source {
				t.Errorf("Source mismatch: got %v, want %v", got.Source, tt.expected.Source)
			}
			if tt.expected.VideoCodec != "" && got.VideoCodec != tt.expected.VideoCodec {
				t.Errorf("VideoCodec mismatch: got %v, want %v", got.VideoCodec, tt.expected.VideoCodec)
			}
			if tt.expected.AudioCodec != "" && got.AudioCodec != tt.expected.AudioCodec {
				t.Errorf("AudioCodec mismatch: got %v, want %v", got.AudioCodec, tt.expected.AudioCodec)
			}
			if tt.expected.ReleaseGroup != "" && got.ReleaseGroup != tt.expected.ReleaseGroup {
				t.Errorf("ReleaseGroup mismatch: got %v, want %v", got.ReleaseGroup, tt.expected.ReleaseGroup)
			}
		})
	}
}
