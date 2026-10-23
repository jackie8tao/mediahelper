package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

func TestJellyfinPlugin(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Library/Refresh" && r.Method == http.MethodPost {
			if r.URL.Query().Get("api_key") == "test_key" {
				called = true
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	p := New()
	cfg := map[string]interface{}{
		"server_url": server.URL,
		"api_key":    "test_key",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := p.Init(ctx, cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	event := plugin.OrganizeEvent{
		Metadata: &scraper.MediaMetadata{Title: "Oppenheimer"},
	}

	if err := p.OnPostOrganize(ctx, event); err != nil {
		t.Fatalf("OnPostOrganize failed: %v", err)
	}

	if !called {
		t.Errorf("expected Jellyfin refresh endpoint to be called")
	}
}
