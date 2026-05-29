package plugin

import (
	"context"
	"testing"
	"time"

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
		m.AddAuthenticator(a, api.Describe(a))
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
	// 深拷贝隔离：原始 payload 不应被修改
	if payload.Model != "original-model" {
		t.Fatalf("expected original payload unchanged, got '%s'", payload.Model)
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

func TestExecuteChatStream_StreamPostProcessor(t *testing.T) {
	transformCalled := false
	sp := &testutil.MockStreamPostProcessor{
		Transform: func(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk {
			transformCalled = true
			out := make(chan *api.StreamChunk, 4)
			go func() {
				defer close(out)
				for c := range in {
					modified := *c
					if len(modified.Choices) > 0 {
						modified.Choices[0].Content = "transformed:" + modified.Choices[0].Content
					}
					out <- &modified
				}
			}()
			return out
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "hello")
	m := newTestManagerFull(router, adapter, nil, nil, nil, []api.PostProcessor{sp})
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
	if chunk.Choices[0].Content != "transformed:mock" {
		t.Fatalf("expected 'transformed:mock', got '%s'", chunk.Choices[0].Content)
	}
	if !transformCalled {
		t.Fatal("expected StreamPostProcessor transform to be called")
	}
}

func TestExecuteChatStream_RegularPostProcessorSkipped(t *testing.T) {
	postCalled := false
	post := &testutil.MockPostProcessor{
		Fn: func(pctx *api.PipelineContext) error {
			postCalled = true
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, nil, []api.PostProcessor{post})
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model", Stream: true}
	ch, err := s.ExecuteChatStream(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 消费所有 chunk
	for range ch {
	}
	if postCalled {
		t.Fatal("regular PostProcessor should not be called in stream mode")
	}
}

func TestExecuteChatStream_NoPostProcessor(t *testing.T) {
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, nil, nil)
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

// ── 管线增强测试 ──

func newTestManagerFull(
	router api.Router,
	adapter api.ChatAdapter,
	auths []api.Authenticator,
	guards []api.Guard,
	pres []api.PreProcessor,
	posts []api.PostProcessor,
) *Manager {
	m := NewManager()
	m.SetRouter(router)
	if adapter != nil {
		_ = m.AddAdapter(adapter)
	}
	for _, a := range auths {
		m.AddAuthenticator(a, api.Describe(a))
	}
	for _, g := range guards {
		m.AddGuard(g, api.Describe(g))
	}
	for _, p := range pres {
		m.AddPreProcessor(p, api.Describe(p))
	}
	for _, p := range posts {
		m.AddPostProcessor(p, api.Describe(p))
	}
	m.Sort()
	return m
}

func TestExecuteChat_TraceID(t *testing.T) {
	var capturedTraceID string
	pre := &testutil.MockPreProcessor{
		Fn: func(pctx *api.PipelineContext) error {
			capturedTraceID = pctx.TraceID
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, []api.PreProcessor{pre}, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTraceID == "" {
		t.Fatal("expected non-empty TraceID")
	}
}

func TestExecuteChat_PriorityOrder(t *testing.T) {
	var order []int
	guard1 := &testutil.MockGuard{Priority: 200, Fn: func(pctx *api.PipelineContext) error {
		order = append(order, 200)
		return nil
	}}
	guard2 := &testutil.MockGuard{Priority: 10, Fn: func(pctx *api.PipelineContext) error {
		order = append(order, 10)
		return nil
	}}
	guard3 := &testutil.MockGuard{Priority: 100, Fn: func(pctx *api.PipelineContext) error {
		order = append(order, 100)
		return nil
	}}

	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard1, guard2, guard3}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 || order[0] != 10 || order[1] != 100 || order[2] != 200 {
		t.Fatalf("expected priority order [10 100 200], got %v", order)
	}
}

func TestExecuteChat_FailModeSkip(t *testing.T) {
	guard := testutil.NewMockGuardWithError(
		api.NewPluginError(api.ErrCodeGuardRejected, "blocked", false),
		api.FailSkip,
	)
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	resp, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("FailSkip should continue, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestExecuteChat_FailModeStrict(t *testing.T) {
	guard := testutil.NewMockGuardWithError(
		api.NewPluginError(api.ErrCodeGuardRejected, "blocked", false),
		api.FailStrict,
	)
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error from FailStrict guard")
	}
}

func TestExecuteChat_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(ctx, payload)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestExecuteChat_DeepCopyIsolation(t *testing.T) {
	pre := &testutil.MockPreProcessor{
		Fn: func(pctx *api.PipelineContext) error {
			req := pctx.Request.(*api.ChatRequest)
			req.Model = "hacked"
			req.Messages = append(req.Messages, api.Message{Role: "system", Content: "injected"})
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, []api.PreProcessor{pre}, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "original", Messages: []api.Message{{Role: "user", Content: "hi"}}}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Model != "original" {
		t.Fatalf("expected original model unchanged, got '%s'", payload.Model)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("expected original 1 message, got %d", len(payload.Messages))
	}
}

func TestExecuteChat_MetaPassthrough(t *testing.T) {
	pre := &testutil.MockPreProcessor{
		Fn: func(pctx *api.PipelineContext) error {
			pctx.Meta["pre_key"] = "pre_value"
			return nil
		},
		Priority: 10,
	}
	var postMeta any
	post := &testutil.MockPostProcessor{
		Fn: func(pctx *api.PipelineContext) error {
			postMeta = pctx.Meta["pre_key"]
			return nil
		},
		Priority: 20,
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, nil, []api.PreProcessor{pre}, []api.PostProcessor{post})
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postMeta != "pre_value" {
		t.Fatalf("expected postprocessor to read meta 'pre_value', got '%v'", postMeta)
	}
}

// ── 插件超时测试 ──

func TestExecuteChat_PluginTimeout(t *testing.T) {
	guard := &testutil.MockGuard{
		Timeout: 50 * time.Millisecond,
		Fn: func(pctx *api.PipelineContext) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExecuteChat_PluginNoTimeout(t *testing.T) {
	guard := &testutil.MockGuard{
		Timeout: 1 * time.Second,
		Fn: func(pctx *api.PipelineContext) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	_, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteChat_PluginTimeoutSkip(t *testing.T) {
	guard := &testutil.MockGuard{
		Timeout:  50 * time.Millisecond,
		FailMode: api.FailSkip,
		Fn: func(pctx *api.PipelineContext) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}
	router := testutil.NewMockRouter("mock", "http://test", "sk-test")
	adapter := testutil.NewMockAdapter("mock", "ok")
	m := newTestManagerFull(router, adapter, nil, []api.Guard{guard}, nil, nil)
	s := NewScheduler(m)

	payload := &api.ChatRequest{Model: "test-model"}
	resp, err := s.ExecuteChat(context.Background(), payload)
	if err != nil {
		t.Fatalf("FailSkip should continue, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
