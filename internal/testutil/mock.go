package testutil

import (
	"context"
	"fmt"
	"time"

	"strait/api"
)

// ── MockRouter ──

// MockRouter 测试用路由插件，返回预设的 RouteDecision
type MockRouter struct {
	Decision *api.RouteDecision
	Err      error
}

func (m *MockRouter) ID() string                  { return "mock-router" }
func (m *MockRouter) Init(_ map[string]any) error { return nil }
func (m *MockRouter) Route(_ context.Context, _ string) (*api.RouteDecision, error) {
	return m.Decision, m.Err
}

func (m *MockRouter) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{ID: "mock-router", Type: "router", Priority: 100, FailMode: api.FailStrict}
}

// NewMockRouter 创建返回指定 provider 的 MockRouter
func NewMockRouter(protocol, baseURL, apiKey string) *MockRouter {
	return &MockRouter{
		Decision: &api.RouteDecision{
			Protocol: protocol,
			BaseURL:  baseURL,
			APIKey:   apiKey,
		},
	}
}

// NewMockRouterWithError 创建返回错误的 MockRouter
func NewMockRouterWithError(err error) *MockRouter {
	return &MockRouter{Err: err}
}

// ── MockAdapter ──

// MockAdapter 测试用 Chat 适配器，返回预设响应
type MockAdapter struct {
	ProtocolName string
	Response     *api.ChatResponse
	Err          error
}

func (m *MockAdapter) ID() string                  { return "mock-adapter" }
func (m *MockAdapter) Init(_ map[string]any) error { return nil }
func (m *MockAdapter) Protocol() string            { return m.ProtocolName }
func (m *MockAdapter) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{ID: "mock-adapter", Type: "adapter", Priority: 100, FailMode: api.FailStrict}
}

func (m *MockAdapter) SendChat(_ context.Context, _ *api.ChatRequest, _ *api.RouteDecision) (*api.ChatResponse, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

func (m *MockAdapter) SendChatStream(_ context.Context, _ *api.ChatRequest, _ *api.RouteDecision) (<-chan *api.StreamChunk, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	ch := make(chan *api.StreamChunk, 1)
	ch <- &api.StreamChunk{
		Choices: []api.Choice{{Role: "assistant", Content: "mock", FinishReason: "stop"}},
		Model:   m.Response.Model,
	}
	close(ch)
	return ch, nil
}

// NewMockAdapter 创建返回简单文本响应的 MockAdapter
func NewMockAdapter(protocol, content string) *MockAdapter {
	return &MockAdapter{
		ProtocolName: protocol,
		Response: &api.ChatResponse{
			ID:    "mock-req-001",
			Model: "mock-model",
			Choices: []api.Choice{
				{Index: 0, Role: "assistant", Content: content, FinishReason: "stop"},
			},
		},
	}
}

// ── MockAuthenticator ──

// MockAuthenticator 测试用认证插件
type MockAuthenticator struct {
	ValidTokens map[string]string // token → subject
}

func (m *MockAuthenticator) ID() string                  { return "mock-auth" }
func (m *MockAuthenticator) Init(_ map[string]any) error { return nil }
func (m *MockAuthenticator) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{ID: "mock-auth", Type: "authenticator", Priority: 10, FailMode: api.FailStrict}
}

func (m *MockAuthenticator) Authenticate(_ context.Context, token string) (*api.Subject, error) {
	if subject, ok := m.ValidTokens[token]; ok {
		return &api.Subject{ID: subject}, nil
	}
	return nil, fmt.Errorf("invalid token: %s", token)
}

// NewMockAuthenticator 创建预设 token 的 MockAuthenticator
func NewMockAuthenticator(tokens map[string]string) *MockAuthenticator {
	return &MockAuthenticator{ValidTokens: tokens}
}

// ── MockGuard ──

// MockGuard 测试用守卫插件
type MockGuard struct {
	Err      error
	Fn       func(pctx *api.PipelineContext) error
	FailMode api.FailMode
	Priority int
	Timeout  time.Duration
	Called   bool
}

func (m *MockGuard) ID() string                  { return "mock-guard" }
func (m *MockGuard) Init(_ map[string]any) error { return nil }
func (m *MockGuard) Guard(pctx *api.PipelineContext) error {
	m.Called = true
	if m.Fn != nil {
		return m.Fn(pctx)
	}
	return m.Err
}

func (m *MockGuard) Descriptor() api.PluginDescriptor {
	fm := m.FailMode
	if fm == "" {
		fm = api.FailStrict
	}
	return api.PluginDescriptor{ID: "mock-guard", Type: "guard", Priority: m.Priority, Timeout: m.Timeout, FailMode: fm}
}

// NewMockGuard 创建默认的 MockGuard
func NewMockGuard() *MockGuard {
	return &MockGuard{Priority: 100}
}

// NewMockGuardWithError 创建返回错误的 MockGuard
func NewMockGuardWithError(err error, fm api.FailMode) *MockGuard {
	return &MockGuard{Err: err, FailMode: fm, Priority: 100}
}

// ── MockPreProcessor ──

// MockPreProcessor 测试用预处理插件
type MockPreProcessor struct {
	Fn       func(pctx *api.PipelineContext) error
	Priority int
	Timeout  time.Duration
	Called   bool
}

func (m *MockPreProcessor) ID() string                  { return "mock-preprocessor" }
func (m *MockPreProcessor) Init(_ map[string]any) error { return nil }
func (m *MockPreProcessor) PreProcess(pctx *api.PipelineContext) error {
	m.Called = true
	if m.Fn != nil {
		return m.Fn(pctx)
	}
	return nil
}

func (m *MockPreProcessor) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{ID: "mock-preprocessor", Type: "preprocessor", Priority: m.Priority, Timeout: m.Timeout, FailMode: api.FailStrict}
}

// NewMockPreProcessor 创建默认的 MockPreProcessor
func NewMockPreProcessor() *MockPreProcessor {
	return &MockPreProcessor{Priority: 100}
}

// ── MockPostProcessor ──

// MockPostProcessor 测试用后处理插件
type MockPostProcessor struct {
	Fn       func(pctx *api.PipelineContext) error
	Priority int
	Called   bool
}

func (m *MockPostProcessor) ID() string                  { return "mock-postprocessor" }
func (m *MockPostProcessor) Init(_ map[string]any) error { return nil }
func (m *MockPostProcessor) PostProcess(pctx *api.PipelineContext) error {
	m.Called = true
	if m.Fn != nil {
		return m.Fn(pctx)
	}
	return nil
}

func (m *MockPostProcessor) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{ID: "mock-postprocessor", Type: "postprocessor", Priority: m.Priority, FailMode: api.FailStrict}
}

// NewMockPostProcessor 创建默认的 MockPostProcessor
func NewMockPostProcessor() *MockPostProcessor {
	return &MockPostProcessor{Priority: 100}
}

type MockStreamPostProcessor struct {
	Transform func(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk
	Priority  int
}

func (m *MockStreamPostProcessor) ID() string                                  { return "mock-stream-postprocessor" }
func (m *MockStreamPostProcessor) Init(_ map[string]any) error                 { return nil }
func (m *MockStreamPostProcessor) PostProcess(pctx *api.PipelineContext) error { return nil }
func (m *MockStreamPostProcessor) PostProcessStream(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk {
	if m.Transform != nil {
		return m.Transform(pctx, in)
	}
	return in
}

func (m *MockStreamPostProcessor) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{
		ID:       "mock-stream-post",
		Type:     "postprocessor",
		Priority: m.Priority,
		FailMode: api.FailStrict,
	}
}
