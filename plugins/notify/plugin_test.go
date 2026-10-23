package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

func TestNotifyPlugin(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if body["title"] == "奥本海默 (2023)" {
					called = true
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}))
	defer server.Close()

	p := New()
	cfg := map[string]interface{}{
		"webhook_url": server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := p.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	event := plugin.OrganizeEvent{
		MediaInfo: &parser.MediaInfo{Title: "奥本海默"},
		Metadata: &scraper.MediaMetadata{
			Title:     "奥本海默",
			Year:      2023,
			MediaType: parser.MediaTypeMovie,
		},
		Result: &organizer.OrganizeResult{
			TargetDir: "/media/movies/Oppenheimer (2023)",
			LinkedFiles: []organizer.FilePair{
				{Source: "a", Target: "b"},
			},
		},
	}

	if err := p.OnPostOrganize(ctx, event); err != nil {
		t.Fatalf("OnPostOrganize failed: %v", err)
	}

	if !called {
		t.Errorf("expected webhook endpoint to be called with formatted payload")
	}
}
