// Package plugin 插件加载、注册、调度。
package plugin

import (
	"fmt"
	"strings"

	"strait/api"
)

// Manager 插件管理器
type Manager struct {
	router         api.Router                 // 路由
	registry       *Registry                  // 插件注册表
	adapters       map[string]api.ChatAdapter // 适配器
	authenticators []api.Authenticator        // 认证器
}

func NewManager() *Manager {
	return &Manager{
		registry: NewRegistry(),
		adapters: make(map[string]api.ChatAdapter),
	}
}

func (m *Manager) SetRouter(r api.Router) { m.router = r }
func (m *Manager) AddAuthenticator(a api.Authenticator) {
	m.authenticators = append(m.authenticators, a)
}

func (m *Manager) AddAdapter(a api.ChatAdapter) error {
	protocol := strings.ToLower(a.Protocol())
	if _, ok := m.adapters[protocol]; ok {
		return fmt.Errorf("adapter already registered for protocol: %s", protocol)
	}
	m.adapters[protocol] = a
	return nil
}

func (m *Manager) Router() api.Router {
	return m.router
}

func (m *Manager) Authenticators() []api.Authenticator {
	return m.authenticators
}

func (m *Manager) Adapter(protocol string) (api.ChatAdapter, bool) {
	a, ok := m.adapters[strings.ToLower(protocol)]
	return a, ok
}
