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
	authenticators []api.Authenticator        // 认证器
	guards         []api.Guard                // 守卫
	preprocessors  []api.PreProcessor         // 预处理器
	adapters       map[string]api.ChatAdapter // 适配器
	postprocessors []api.PostProcessor        // 后置处理器
	Warnings       []string                   // 启动警告
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
func (m *Manager) AddGuard(g api.Guard)               { m.guards = append(m.guards, g) }
func (m *Manager) AddPreProcessor(p api.PreProcessor) { m.preprocessors = append(m.preprocessors, p) }

func (m *Manager) AddAdapter(a api.ChatAdapter) error {
	protocol := strings.ToLower(a.Protocol())
	if _, ok := m.adapters[protocol]; ok {
		return fmt.Errorf("adapter already registered for protocol: %s", protocol)
	}
	m.adapters[protocol] = a
	return nil
}

func (m *Manager) AddPostProcessor(p api.PostProcessor) {
	m.postprocessors = append(m.postprocessors, p)
}

func (m *Manager) Router() api.Router {
	return m.router
}

func (m *Manager) Authenticators() []api.Authenticator {
	return m.authenticators
}
func (m *Manager) Guards() []api.Guard               { return m.guards }
func (m *Manager) PreProcessors() []api.PreProcessor { return m.preprocessors }
func (m *Manager) Adapter(protocol string) (api.ChatAdapter, bool) {
	a, ok := m.adapters[strings.ToLower(protocol)]
	return a, ok
}
func (m *Manager) PostProcessors() []api.PostProcessor { return m.postprocessors }

// Summary 返回已加载插件的摘要信息
func (m *Manager) Summary() string {
	var b strings.Builder
	if m.router != nil {
		fmt.Fprintf(&b, "   ● %s (router)\n", m.router.ID())
	}
	for _, a := range m.adapters {
		fmt.Fprintf(&b, "   ● %s (adapter)\n", a.ID())
	}
	for _, a := range m.authenticators {
		fmt.Fprintf(&b, "   ● %s (authenticator)\n", a.ID())
	}
	for _, g := range m.guards {
		fmt.Fprintf(&b, "   ● %s (guard)\n", g.ID())
	}
	for _, p := range m.preprocessors {
		fmt.Fprintf(&b, "   ● %s (preprocessor)\n", p.ID())
	}
	for _, p := range m.postprocessors {
		fmt.Fprintf(&b, "   ● %s (postprocessor)\n", p.ID())
	}
	return b.String()
}
