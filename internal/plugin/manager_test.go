package plugin

import (
	"context"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

type testHookPlugin struct {
	id             string
	organizedCount int
	panics         bool
}

func (t *testHookPlugin) ID() string          { return t.id }
func (t *testHookPlugin) Name() string        { return "Test Plugin" }
func (t *testHookPlugin) Version() string     { return "1.0.0" }
func (t *testHookPlugin) Description() string { return "Test description" }
func (t *testHookPlugin) Init(ctx context.Context, cfg map[string]interface{}) error {
	return nil
}
func (t *testHookPlugin) Start(ctx context.Context) error { return nil }
func (t *testHookPlugin) Stop(ctx context.Context) error  { return nil }
func (t *testHookPlugin) OnPostOrganize(ctx context.Context, event OrganizeEvent) error {
	if t.panics {
		panic("intentional plugin panic")
	}
	t.organizedCount++
	return nil
}

func TestPluginManager(t *testing.T) {
	mgr := NewManager()

	p1 := &testHookPlugin{id: "p1"}
	p2 := &testHookPlugin{id: "p2", panics: true}

	if err := mgr.Register(p1); err != nil {
		t.Fatalf("Register p1 failed: %v", err)
	}
	if err := mgr.Register(p2); err != nil {
		t.Fatalf("Register p2 failed: %v", err)
	}

	// 重复注册应报错
	if err := mgr.Register(p1); err == nil {
		t.Errorf("expected error registering duplicate plugin")
	}

	cfg := config.PluginConfig{
		Enabled: true,
		Items: map[string]map[string]interface{}{
			"p1": {"enabled": true},
			"p2": {"enabled": true},
		},
	}

	ctx := context.Background()
	if err := mgr.InitAll(ctx, cfg); err != nil {
		t.Fatalf("InitAll failed: %v", err)
	}
	_ = mgr.StartAll(ctx)
	defer mgr.StopAll(ctx)

	// 测试广播与 Panic 隔离
	event := OrganizeEvent{
		Torrent:   downloader.Torrent{Name: "Test"},
		MediaInfo: &parser.MediaInfo{Title: "Test"},
		Metadata:  &scraper.MediaMetadata{Title: "Test"},
		Result:    &organizer.OrganizeResult{Success: true},
	}

	mgr.EmitPostOrganize(ctx, event)

	if p1.organizedCount != 1 {
		t.Errorf("expected p1.organizedCount = 1, got %d", p1.organizedCount)
	}

	// 检查 ListPlugins
	list := mgr.ListPlugins()
	if len(list) != 2 {
		t.Errorf("expected 2 plugins in list, got %d", len(list))
	}
	foundP1 := false
	for _, info := range list {
		if info.ID == "p1" {
			foundP1 = true
			if !info.Enabled {
				t.Errorf("expected p1 enabled to be true")
			}
			if len(info.Hooks) == 0 || info.Hooks[0] != "PostOrganize" {
				t.Errorf("expected PostOrganize hook in info: %+v", info.Hooks)
			}
		}
	}
	if !foundP1 {
		t.Errorf("p1 not found in list")
	}
}
