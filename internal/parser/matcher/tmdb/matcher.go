package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/parser"
)

// Config holds configuration parameters for the TMDB matcher.
type Config struct {
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Language string `json:"language"`
	Proxy    string `json:"proxy"`
}

// Matcher implements parser.Matcher by querying TMDB API to verify candidate metadata.
type Matcher struct {
	cfg        Config
	httpClient *http.Client
}

func init() {
	parser.RegisterMatcherFactory("tmdb", TMDBMatcherFactory)
}

// TMDBMatcherFactory initializes a new TMDB matcher instance from raw JSON config.
func TMDBMatcherFactory(cfgVal json.RawMessage) (parser.Matcher, error) {
	var cfgSt Config
	if len(cfgVal) > 0 {
		if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
			return nil, fmt.Errorf("unmarshal tmdb matcher config error: %w", err)
		}
	}
	return NewMatcher(cfgSt)
}

// NewMatcher creates a new TMDB Matcher instance with the given configuration.
func NewMatcher(cfg Config) (*Matcher, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb matcher api_key is required")
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL for tmdb matcher: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.themoviedb.org/3"
	}
	if cfg.Language == "" {
		cfg.Language = "zh-CN"
	}

	return &Matcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}, nil
}

// Name returns the matcher provider name.
func (m *Matcher) Name() string {
	return "TMDB"
}

type tmdbItem struct {
	ID            int
	Title         string
	OriginalTitle string
	ReleaseDate   string
	MediaType     parser.MediaType
}

// Match searches TMDB to find the most accurate matching media item for candidate.
func (m *Matcher) Match(ctx context.Context, candidate *parser.MediaInfo) (*parser.MatchResult, error) {
	if candidate == nil {
		return &parser.MatchResult{Matched: false}, nil
	}

	// 1. Prepare search queries
	var queries []string
	if t := strings.TrimSpace(candidate.Title); t != "" {
		queries = append(queries, t)
	}
	if t := strings.TrimSpace(candidate.CnTitle); t != "" && t != candidate.Title {
		queries = append(queries, t)
	}
	if t := strings.TrimSpace(candidate.EnTitle); t != "" && t != candidate.Title && t != candidate.CnTitle {
		queries = append(queries, t)
	}

	if len(queries) == 0 {
		return &parser.MatchResult{Matched: false}, nil
	}

	primaryType := candidate.MediaType
	if primaryType == "" {
		primaryType = parser.MediaTypeMovie
	}

	var bestItem *tmdbItem
	var bestScore float64

	// Search attempts: queries with year, then queries without year
	for _, q := range queries {
		items, err := m.search(ctx, q, candidate.Year, primaryType)
		if err == nil && len(items) > 0 {
			item, score := m.findBestItem(q, candidate.Year, primaryType, items)
			if score > bestScore {
				bestItem = item
				bestScore = score
			}
		}
		if bestScore >= 0.85 {
			break
		}
	}

	// Fallback without year if no confident match was found
	if bestScore < 0.75 && candidate.Year > 0 {
		for _, q := range queries {
			items, err := m.search(ctx, q, 0, primaryType)
			if err == nil && len(items) > 0 {
				item, score := m.findBestItem(q, candidate.Year, primaryType, items)
				if score > bestScore {
					bestItem = item
					bestScore = score
				}
			}
			if bestScore >= 0.80 {
				break
			}
		}
	}

	if bestItem == nil || bestScore < 0.70 {
		return &parser.MatchResult{
			Matched:    false,
			Provider:   "tmdb",
			Confidence: bestScore,
		}, nil
	}

	year := 0
	if len(bestItem.ReleaseDate) >= 4 {
		year, _ = strconv.Atoi(bestItem.ReleaseDate[:4])
	}

	return &parser.MatchResult{
		Matched:       true,
		Provider:      "tmdb",
		TMDBID:        bestItem.ID,
		Title:         bestItem.Title,
		OriginalTitle: bestItem.OriginalTitle,
		Year:          year,
		MediaType:     bestItem.MediaType,
		Confidence:    bestScore,
	}, nil
}

func (m *Matcher) search(ctx context.Context, query string, year int, preferredType parser.MediaType) ([]tmdbItem, error) {
	if preferredType == parser.MediaTypeTV || preferredType == parser.MediaTypeAnime {
		tvItems, err := m.searchTV(ctx, query, year)
		if err == nil && len(tvItems) > 0 {
			return tvItems, nil
		}
		// Fallback to movie search for OVA / movie anime
		return m.searchMovie(ctx, query, year)
	}

	// Preferred Movie
	movieItems, err := m.searchMovie(ctx, query, year)
	if err == nil && len(movieItems) > 0 {
		return movieItems, nil
	}
	// Fallback to TV search
	return m.searchTV(ctx, query, year)
}

func (m *Matcher) searchMovie(ctx context.Context, query string, year int) ([]tmdbItem, error) {
	u, _ := url.Parse(strings.TrimRight(m.cfg.BaseURL, "/") + "/search/movie")
	q := u.Query()
	q.Set("api_key", m.cfg.APIKey)
	q.Set("query", query)
	q.Set("language", m.cfg.Language)
	if year > 0 {
		q.Set("year", strconv.Itoa(year))
	}
	u.RawQuery = q.Encode()

	var resp struct {
		Results []struct {
			ID            int    `json:"id"`
			Title         string `json:"title"`
			OriginalTitle string `json:"original_title"`
			ReleaseDate   string `json:"release_date"`
		} `json:"results"`
	}

	if err := m.doGet(ctx, u.String(), &resp); err != nil {
		return nil, err
	}

	var items []tmdbItem
	for _, r := range resp.Results {
		items = append(items, tmdbItem{
			ID:            r.ID,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			ReleaseDate:   r.ReleaseDate,
			MediaType:     parser.MediaTypeMovie,
		})
	}
	return items, nil
}

func (m *Matcher) searchTV(ctx context.Context, query string, year int) ([]tmdbItem, error) {
	u, _ := url.Parse(strings.TrimRight(m.cfg.BaseURL, "/") + "/search/tv")
	q := u.Query()
	q.Set("api_key", m.cfg.APIKey)
	q.Set("query", query)
	q.Set("language", m.cfg.Language)
	if year > 0 {
		q.Set("first_air_date_year", strconv.Itoa(year))
	}
	u.RawQuery = q.Encode()

	var resp struct {
		Results []struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			OriginalName string `json:"original_name"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := m.doGet(ctx, u.String(), &resp); err != nil {
		return nil, err
	}

	var items []tmdbItem
	for _, r := range resp.Results {
		items = append(items, tmdbItem{
			ID:            r.ID,
			Title:         r.Name,
			OriginalTitle: r.OriginalName,
			ReleaseDate:   r.FirstAirDate,
			MediaType:     parser.MediaTypeTV,
		})
	}
	return items, nil
}

func (m *Matcher) findBestItem(query string, targetYear int, targetType parser.MediaType, items []tmdbItem) (*tmdbItem, float64) {
	var best *tmdbItem
	var maxScore = -1.0

	for i := range items {
		score := calculateScore(query, targetYear, targetType, &items[i], i)
		if score > maxScore {
			maxScore = score
			best = &items[i]
		}
	}
	return best, maxScore
}

func calculateScore(query string, targetYear int, targetType parser.MediaType, item *tmdbItem, rankIndex int) float64 {
	cleanQuery := normalizeString(query)
	cleanTitle := normalizeString(item.Title)
	cleanOrig := normalizeString(item.OriginalTitle)

	// 1. Title Similarity (0.0 - 0.60)
	titleScore := 0.0
	if cleanQuery == cleanTitle || cleanQuery == cleanOrig {
		titleScore = 0.60
	} else if strings.Contains(cleanTitle, cleanQuery) || strings.Contains(cleanQuery, cleanTitle) {
		titleScore = 0.50
	} else if strings.Contains(cleanOrig, cleanQuery) || strings.Contains(cleanQuery, cleanOrig) {
		titleScore = 0.45
	} else {
		sim := stringSimilarity(cleanQuery, cleanTitle)
		if origSim := stringSimilarity(cleanQuery, cleanOrig); origSim > sim {
			sim = origSim
		}
		titleScore = 0.60 * sim
		// Base score for items directly returned by TMDB's internal alias/transliteration search
		baseAliasScore := math.Max(0.20, 0.45-float64(rankIndex)*0.08)
		if titleScore < baseAliasScore {
			titleScore = baseAliasScore
		}
	}

	// 2. Year Proximity (0.0 - 0.30)
	yearScore := 0.15 // neutral score when year is unknown
	if targetYear > 0 && len(item.ReleaseDate) >= 4 {
		itemYear, err := strconv.Atoi(item.ReleaseDate[:4])
		if err == nil && itemYear > 0 {
			diff := int(math.Abs(float64(targetYear - itemYear)))
			switch diff {
			case 0:
				yearScore = 0.30
			case 1:
				yearScore = 0.20
			case 2:
				yearScore = 0.10
			default:
				yearScore = 0.0
			}
		}
	}

	// 3. Media Type Agreement (0.0 - 0.10)
	typeScore := 0.05
	if targetType != "" {
		if targetType == item.MediaType || (targetType == parser.MediaTypeAnime && item.MediaType == parser.MediaTypeTV) {
			typeScore = 0.10
		} else {
			typeScore = 0.0
		}
	}

	return titleScore + yearScore + typeScore
}

func (m *Matcher) doGet(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tmdb api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func normalizeString(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(":", "", "-", "", "_", "", ".", "", ",", "", " ", "")
	return replacer.Replace(s)
}

func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Dice coefficient on bigrams
	aGrams := make(map[string]int)
	for i := 0; i < len(a)-1; i++ {
		aGrams[a[i:i+2]]++
	}

	bMatches := 0
	for i := 0; i < len(b)-1; i++ {
		g := b[i : i+2]
		if count, ok := aGrams[g]; ok && count > 0 {
			aGrams[g]--
			bMatches++
		}
	}

	total := (len(a) - 1) + (len(b) - 1)
	if total <= 0 {
		return 0.0
	}
	return 2.0 * float64(bMatches) / float64(total)
}
