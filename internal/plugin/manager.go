// Package plugin 插件加载、注册、调度。
package plugin

import (
	"fmt"
	"sort"
	"strings"

	"strait/api"
)

type taggedPlugin[T any] struct {
	plugin T
	desc   api.PluginDescriptor
}

// Manager 插件管理器
type Manager struct {
	router         api.Router                        // 路由
	registry       *Registry                         // 插件注册表
	authenticators []taggedPlugin[api.Authenticator] // 认证器
	adapters       map[string]api.ChatAdapter        // 适配器

	guardSteps []pipelineStep
	preSteps   []pipelineStep
	postSteps  []pipelineStep

	guardDescs []api.PluginDescriptor
	preDescs   []api.PluginDescriptor
	postDescs  []api.PluginDescriptor

	Warnings []string // 启动警告
}

func NewManager() *Manager {
	return &Manager{
		registry: NewRegistry(),
		adapters: make(map[string]api.ChatAdapter),
	}
}

func (m *Manager) SetRouter(r api.Router) { m.router = r }
func (m *Manager) AddAuthenticator(a api.Authenticator, d api.PluginDescriptor) {
	m.authenticators = append(m.authenticators, taggedPlugin[api.Authenticator]{plugin: a, desc: d})
}

func (m *Manager) AddGuard(g api.Guard, d api.PluginDescriptor) {
	m.guardSteps = append(m.guardSteps, pipelineStep{
		exec:     g.Guard,
		failMode: d.FailMode,
		timeout:  d.Timeout,
	})
	m.guardDescs = append(m.guardDescs, d)
}

func (m *Manager) AddPreProcessor(p api.PreProcessor, d api.PluginDescriptor) {
	m.preSteps = append(m.preSteps, pipelineStep{
		exec:     p.PreProcess,
		failMode: d.FailMode,
		timeout:  d.Timeout,
	})
	m.preDescs = append(m.preDescs, d)
}

func (m *Manager) AddAdapter(a api.ChatAdapter) error {
	protocol := strings.ToLower(a.Protocol())
	if _, ok := m.adapters[protocol]; ok {
		return fmt.Errorf("adapter already registered for protocol: %s", protocol)
	}
	m.adapters[protocol] = a
	return nil
}

func (m *Manager) AddPostProcessor(p api.PostProcessor, d api.PluginDescriptor) {
	step := pipelineStep{
		exec:     p.PostProcess,
		failMode: d.FailMode,
		timeout:  d.Timeout,
	}
	if sp, ok := p.(api.StreamPostProcessor); ok {
		step.streamExec = sp.PostProcessStream
	}
	m.postSteps = append(m.postSteps, step)
	m.postDescs = append(m.postDescs, d)
}

func (m *Manager) Router() api.Router {
	return m.router
}

func (m *Manager) Authenticators() []taggedPlugin[api.Authenticator] {
	return m.authenticators
}
func (m *Manager) GuardSteps() []pipelineStep { return m.guardSteps }
func (m *Manager) PreSteps() []pipelineStep   { return m.preSteps }
func (m *Manager) Adapter(protocol string) (api.ChatAdapter, bool) {
	a, ok := m.adapters[strings.ToLower(protocol)]
	return a, ok
}
func (m *Manager) PostSteps() []pipelineStep { return m.postSteps }

func (m *Manager) Sort() {
	sortPairs(m.guardSteps, m.guardDescs)
	sortPairs(m.preSteps, m.preDescs)
	sortPairs(m.postSteps, m.postDescs)
	sort.SliceStable(m.authenticators, func(i, j int) bool {
		return m.authenticators[i].desc.Priority < m.authenticators[j].desc.Priority
	})
}

func sortPairs(steps []pipelineStep, descs []api.PluginDescriptor) {
	idx := make([]int, len(steps))
	for i := range steps {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		return descs[idx[i]].Priority < descs[idx[j]].Priority
	})
	sortedSteps := make([]pipelineStep, len(steps))
	sortedDescs := make([]api.PluginDescriptor, len(descs))
	for i, k := range idx {
		sortedSteps[i] = steps[k]
		sortedDescs[i] = descs[k]
	}
	copy(steps, sortedSteps)
	copy(descs, sortedDescs)
}

// Manifest 为已加载的插件生成清单
func (m *Manager) Manifest() []api.PluginDescriptor {
	descs := make([]api.PluginDescriptor, 0, len(m.authenticators)+len(m.guardDescs)+len(m.preDescs)+len(m.postDescs)+len(m.adapters)+1)
	if m.router != nil {
		descs = append(descs, api.Describe(m.router))
	}
	for _, a := range m.adapters {
		descs = append(descs, api.Describe(a))
	}
	for _, a := range m.authenticators {
		descs = append(descs, a.desc)
	}
	descs = append(descs, m.guardDescs...)
	descs = append(descs, m.preDescs...)
	descs = append(descs, m.postDescs...)
	return descs
}

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
		fmt.Fprintf(&b, "   ● %s (authenticator)\n", a.plugin.ID())
	}
	for _, d := range m.guardDescs {
		fmt.Fprintf(&b, "   ● %s (guard)\n", d.ID)
	}
	for _, d := range m.preDescs {
		fmt.Fprintf(&b, "   ● %s (preprocessor)\n", d.ID)
	}
	for _, d := range m.postDescs {
		fmt.Fprintf(&b, "   ● %s (postprocessor)\n", d.ID)
	}
	return b.String()
}
