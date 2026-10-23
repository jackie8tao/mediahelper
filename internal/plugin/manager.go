package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackie8tao/mediahelper/internal/config"
)

// Manager 插件管理器中枢
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	enabled map[string]bool
}

// NewManager 创建插件管理器
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		enabled: make(map[string]bool),
	}
}

// Register 注册一个插件
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := p.ID()
	if id == "" {
		return fmt.Errorf("plugin ID cannot be empty")
	}
	if _, exists := m.plugins[id]; exists {
		return fmt.Errorf("plugin %s already registered", id)
	}

	m.plugins[id] = p
	slog.Debug("Plugin registered", "id", id, "name", p.Name(), "version", p.Version())
	return nil
}

// InitAll 根据全局配置初始化所有插件
func (m *Manager) InitAll(ctx context.Context, cfg config.PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !cfg.Enabled {
		slog.Info("Plugin mechanism is globally disabled")
		return nil
	}

	for id, p := range m.plugins {
		pCfg := cfg.Items[id]
		if pCfg == nil {
			pCfg = make(map[string]interface{})
		}

		// 检查该插件是否显式开启 (默认若在配置中出现且 enabled != false 则启用)
		isEnabled := true
		if val, ok := pCfg["enabled"]; ok {
			if b, ok := val.(bool); ok {
				isEnabled = b
			}
		} else {
			// 未配置项默认不自动开启
			isEnabled = false
		}

		if !isEnabled {
			m.enabled[id] = false
			continue
		}

		if err := p.Init(ctx, pCfg); err != nil {
			slog.Error("Failed to initialize plugin", "id", id, "error", err)
			m.enabled[id] = false
			continue
		}

		m.enabled[id] = true
		slog.Info("Plugin initialized and enabled", "id", id, "name", p.Name())
	}

	return nil
}

// StartAll 启动所有已启用的插件
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, p := range m.plugins {
		if m.enabled[id] {
			if err := p.Start(ctx); err != nil {
				slog.Error("Failed to start plugin", "id", id, "error", err)
			}
		}
	}
	return nil
}

// StopAll 优雅关停所有插件
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, p := range m.plugins {
		if m.enabled[id] {
			if err := p.Stop(ctx); err != nil {
				slog.Error("Failed to stop plugin", "id", id, "error", err)
			}
		}
	}
	return nil
}

// EmitTorrentCompleted 广播种子完成事件 (带 Panic 隔离保护)
func (m *Manager) EmitTorrentCompleted(ctx context.Context, event TorrentEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, p := range m.plugins {
		if !m.enabled[id] {
			continue
		}

		if hook, ok := p.(TorrentCompletedHook); ok {
			func(pluginID string, h TorrentCompletedHook) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Plugin panic in OnTorrentCompleted", "id", pluginID, "panic", r)
					}
				}()
				if err := h.OnTorrentCompleted(ctx, event); err != nil {
					slog.Warn("Plugin error in OnTorrentCompleted", "id", pluginID, "error", err)
				}
			}(id, hook)
		}
	}
}

// EmitPostScrape 广播元数据刮削完成事件 (带 Panic 隔离保护)
func (m *Manager) EmitPostScrape(ctx context.Context, event ScrapeEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, p := range m.plugins {
		if !m.enabled[id] {
			continue
		}

		if hook, ok := p.(PostScrapeHook); ok {
			func(pluginID string, h PostScrapeHook) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Plugin panic in OnPostScrape", "id", pluginID, "panic", r)
					}
				}()
				if err := h.OnPostScrape(ctx, event); err != nil {
					slog.Warn("Plugin error in OnPostScrape", "id", pluginID, "error", err)
				}
			}(id, hook)
		}
	}
}

// EmitPostOrganize 广播文件整理完成事件 (带 Panic 隔离保护)
func (m *Manager) EmitPostOrganize(ctx context.Context, event OrganizeEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, p := range m.plugins {
		if !m.enabled[id] {
			continue
		}

		if hook, ok := p.(PostOrganizeHook); ok {
			func(pluginID string, h PostOrganizeHook) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Plugin panic in OnPostOrganize", "id", pluginID, "panic", r)
					}
				}()
				if err := h.OnPostOrganize(ctx, event); err != nil {
					slog.Warn("Plugin error in OnPostOrganize", "id", pluginID, "error", err)
				}
			}(id, hook)
		}
	}
}

// ListPlugins 列出所有已注册插件及其元数据
func (m *Manager) ListPlugins() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []Info
	for id, p := range m.plugins {
		var hooks []string
		if _, ok := p.(TorrentCompletedHook); ok {
			hooks = append(hooks, "TorrentCompleted")
		}
		if _, ok := p.(PostScrapeHook); ok {
			hooks = append(hooks, "PostScrape")
		}
		if _, ok := p.(PostOrganizeHook); ok {
			hooks = append(hooks, "PostOrganize")
		}

		list = append(list, Info{
			ID:          id,
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			Enabled:     m.enabled[id],
			Hooks:       hooks,
		})
	}
	return list
}
