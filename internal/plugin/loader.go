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
	ID     string         `yaml:"id"`     // 插件ID
	Type   string         `yaml:"type"`   // 插件类型
	Config map[string]any `yaml:"config"` // 插件配置
}

// pluginConfig 插件配置
type pluginConfig struct {
	Plugins []pluginEntry `yaml:"plugins"`
}

func NewLoader(path string) *Loader {
	return &Loader{configPath: path}
}

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
			return nil, err
		}
	}

	if router != nil {
		m.SetRouter(router)
	}

	slog.Info(
		"all plugins loaded successfully",
		"router_loaded", router != nil,
		"authenticator_count", len(m.authenticators),
		"adapter_count", len(m.adapters),
	)
	return m, nil
}

func (l *Loader) loadPlugin(entry pluginEntry, m *Manager, router *api.Router) error {
	slog.Info("loading plugin", "id", entry.ID, "type", entry.Type)

	p, ok := api.CreatePlugin(entry.ID)
	if !ok {
		return fmt.Errorf("plugin not registered: %s", entry.ID)
	}

	// 初始化插件
	if err := p.Init(entry.Config); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", entry.ID, err)
	}

	// 根据类型进行分类
	switch entry.Type {
	case "router":
		r, ok := p.(api.Router)
		if !ok {
			return fmt.Errorf("plugin %s is not a Router", entry.ID)
		}
		*router = r
		slog.Info("router plugin loaded", "id", entry.ID)
	case "adapter":
		a, ok := p.(api.ChatAdapter)
		if !ok {
			return fmt.Errorf("plugin %s is not a ChatAdapter", entry.ID)
		}
		if err := m.AddAdapter(a); err != nil {
			return err
		}
		slog.Info("adapter plugin loaded", "id", entry.ID, "protocol", a.Protocol())
	case "authenticator":
		auth, ok := p.(api.Authenticator)
		if !ok {
			return fmt.Errorf("plugin %s is not an Authenticator", entry.ID)
		}
		m.AddAuthenticator(auth)
		slog.Info("authenticator plugin loaded", "id", entry.ID)
	default:
		return fmt.Errorf("unknown plugin type: %s", entry.Type)
	}
	return nil
}
