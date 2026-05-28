package testutil

import (
	"context"
	"fmt"

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
