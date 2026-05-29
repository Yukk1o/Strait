// Package plugin 插件加载、注册、调度。
package plugin

import (
	"fmt"
	"log/slog"
	"os"

	"strait/api"

	"github.com/goccy/go-yaml"
)

// Loader 插件加载器
type Loader struct {
	configPath string // 配置文件路径
}

// pluginEntry 插件配置项
type pluginEntry struct {
	ID       string         `yaml:"id"`                  // 插件ID
	Type     string         `yaml:"type"`                // 插件类型
	Config   map[string]any `yaml:"config"`              // 插件配置
	Priority *int           `yaml:"priority,omitempty"`  // 插件优先级
	FailMode string         `yaml:"fail_mode,omitempty"` // 插件失败模式
}

// pluginConfig 插件配置
type pluginConfig struct {
	Plugins []pluginEntry `yaml:"plugins"`
}

func NewLoader(path string) *Loader {
	return &Loader{configPath: path}
}

// Build 构建插件管理器
func (l *Loader) Build() (*Manager, error) {
	slog.Info("starting plugin loader", "config_path", l.configPath)

	// 读取配置文件
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置文件
	var cfg pluginConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	m := NewManager()
	var router api.Router
	for _, entry := range cfg.Plugins {
		if err := l.loadPlugin(entry, m, &router); err != nil {
			if isCriticalType(entry.Type) {
				return nil, err
			}
			slog.Warn("optional plugin skipped", "id", entry.ID, "type", entry.Type, "error", err)
		}
	}

	if router != nil {
		m.SetRouter(router)
	}

	m.Sort()

	return m, nil
}

// isCriticalType 判断插件类型是否为关键类型，加载失败必须退出
func isCriticalType(t string) bool {
	switch t {
	case "router", "adapter", "authenticator":
		return true
	default:
		return false
	}
}

// loadPlugin 加载插件
func (l *Loader) loadPlugin(entry pluginEntry, m *Manager, router *api.Router) error {
	slog.Debug("loading plugin", "id", entry.ID, "type", entry.Type)

	p, ok := api.CreatePlugin(entry.ID)
	if !ok {
		return fmt.Errorf("plugin not registered: %s", entry.ID)
	}

	desc := api.Describe(p)
	if entry.Priority != nil {
		desc.Priority = *entry.Priority
	}
	if entry.FailMode != "" {
		desc.FailMode = api.FailMode(entry.FailMode)
	}

	cfg := mergeConfig(desc.DefaultConfig, entry.Config)

	if err := p.Init(cfg); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", entry.ID, err)
	}

	if err := registerPlugin(entry, p, m, router, desc); err != nil {
		return err
	}
	return nil
}

// registerPlugin 注册插件
func registerPlugin(entry pluginEntry, p api.Plugin, m *Manager, router *api.Router, desc api.PluginDescriptor) error {
	switch entry.Type {
	case "authenticator":
		auth, ok := p.(api.Authenticator)
		if !ok {
			return fmt.Errorf("plugin %s is not an Authenticator", entry.ID)
		}

		m.AddAuthenticator(auth, desc)
	case "guard":
		guard, ok := p.(api.Guard)
		if !ok {
			return fmt.Errorf("plugin %s is not a Guard", entry.ID)
		}
		m.AddGuard(guard, desc)
	case "preprocessor":
		pre, ok := p.(api.PreProcessor)
		if !ok {
			return fmt.Errorf("plugin %s is not a PreProcessor", entry.ID)
		}
		m.AddPreProcessor(pre, desc)
	case "router":
		r, ok := p.(api.Router)
		if !ok {
			return fmt.Errorf("plugin %s is not a Router", entry.ID)
		}
		*router = r
	case "postprocessor":
		post, ok := p.(api.PostProcessor)
		if !ok {
			return fmt.Errorf("plugin %s is not a PostProcessor", entry.ID)
		}
		if _, ok := p.(api.StreamPostProcessor); !ok {
			slog.Warn(
				"PostProcessor does not implement StreamPostProcessor, will be skipped in stream mode",
				"plugin", entry.ID,
			)
		}
		m.AddPostProcessor(post, desc)
	case "adapter":
		a, ok := p.(api.ChatAdapter)
		if !ok {
			return fmt.Errorf("plugin %s is not a ChatAdapter", entry.ID)
		}
		if err := m.AddAdapter(a); err != nil {
			return err
		}
		slog.Debug("adapter plugin loaded", "id", entry.ID, "protocol", a.Protocol())
		return nil
	default:
		return fmt.Errorf("unknown plugin type: %s", entry.Type)
	}
	slog.Debug(entry.Type+" plugin loaded", "id", entry.ID)
	return nil
}

func mergeConfig(base, override map[string]any) map[string]any {
	if len(base) == 0 {
		return override
	}
	if len(override) == 0 {
		return base
	}
	merged := make(map[string]any, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}
