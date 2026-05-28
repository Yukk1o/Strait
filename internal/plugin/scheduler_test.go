package plugin

import (
	"context"
	"testing"

	"strait/api"
	"strait/internal/testutil"
)

func newTestManager(router api.Router, adapter api.ChatAdapter, auths ...api.Authenticator) *Manager {
	m := NewManager()
	m.SetRouter(router)
	if adapter != nil {
		_ = m.AddAdapter(adapter)
	}
	for _, a := range auths {
		m.AddAuthenticator(a)
	}
	return m
}

// ── ExecuteChat ──

func TestExecuteChat_Success(t *testing.T) {
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "hello")
	m := newTestManager(router, adapter)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model", Messages: []api.Message{{Role: "user", Content: "hi"}}}
	resp, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Choices[0].Content != "hello" {
		t.Fatalf("expected 'hello', got '%s'", resp.Choices[0].Content)
	}
}

func TestExecuteChat_AuthFailed(t *testing.T) {
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "hello")
	auth := testutil.NewMockAuthenticator(map[string]string{"valid-token": "admin"})
	m := newTestManager(router, adapter, auth)
	s := NewScheduler(m)

	ctx := api.WithAuthToken(context.Background(), "wrong-token")
	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(ctx, payload)
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestExecuteChat_AuthSuccess(t *testing.T) {
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	auth := testutil.NewMockAuthenticator(map[string]string{"valid-token": "admin"})
	m := newTestManager(router, adapter, auth)
	s := NewScheduler(m)

	ctx := api.WithAuthToken(context.Background(), "valid-token")
	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteChat_NoAdapter(t *testing.T) {
	router := testutil.NewMockRouter("unknown-protocol", "http://test", "sk-test")
	m := newTestManager(router, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err == nil {
		t.Fatal("expected adapter_not_found error")
	}
}

func TestExecuteChat_RouterError(t *testing.T) {
	router := testutil.NewMockRouterWithError(&api.PluginError{Code: "NO_ROUTE", Message: "no route"})
	adapter := testutil.NewMockAdapter("mock", "hello")
	m := newTestManager(router, adapter)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "unknown"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err == nil {
		t.Fatal("expected router error")
	}
}

func TestExecuteChat_ModelOverride(t *testing.T) {
	router := &testutil.MockRouter{
		Decision: &api.RouteDecision{
			Protocol: "mock",
			BaseURL:  "http://test",
			APIKey:   "sk-test",
			Model:    "overridden-model",
		},
	}
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManager(router, adapter)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "original-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Model != "overridden-model" {
		t.Fatalf("expected model override to 'overridden-model', got '%s'", payload.Model)
	}
}

// ── ExecuteChatStream ──

func TestExecuteChatStream_Success(t *testing.T) {
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "stream-ok")
	m := newTestManager(router, adapter)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model", Stream: true}
	ch, err := s.ExecuteChatStream(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chunk, ok := <-ch
	if !ok {
		t.Fatal("expected at least one chunk")
	}
	if chunk.Choices[0].Content != "mock" {
		t.Fatalf("expected 'mock', got '%s'", chunk.Choices[0].Content)
	}
}

// ── ReloadManager ──

func TestReloadManager(t *testing.T) {
	router1 := testutil.NewMockRouter("mock", "http://v1", "sk-v1")
	adapter1 := testutil.NewMockAdapter("mock", "v1")
	m1 := newTestManager(router1, adapter1)
	s := NewScheduler(m1)

	// v1
	payload := &api.ChatRequest{Model: "test"}
	resp, _ := s.ExecuteChat(context.Background(), payload)
	if resp.Choices[0].Content != "v1" {
		t.Fatalf("expected v1, got %s", resp.Choices[0].Content)
	}

	// reload to v2
	router2 := testutil.NewMockRouter("mock", "http://v2", "sk-v2")
	adapter2 := testutil.NewMockAdapter("mock", "v2")
	m2 := newTestManager(router2, adapter2)
	s.ReloadManager(m2)

	resp, _ = s.ExecuteChat(context.Background(), payload)
	if resp.Choices[0].Content != "v2" {
		t.Fatalf("expected v2 after reload, got %s", resp.Choices[0].Content)
	}
}
