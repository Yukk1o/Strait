// Package plugin 提供插件注册、加载和调度功能
package plugin

import (
	"context"
	"fmt"
	"sync/atomic"

	"strait/api"
)

// Scheduler 按阶段调用插件，处理单次请求
type Scheduler struct {
	manager atomic.Pointer[Manager] // 管理器
}

// NewScheduler 创建调度器
func NewScheduler(m *Manager) *Scheduler {
	s := &Scheduler{}
	s.manager.Store(m)
	return s
}

func (s *Scheduler) ReloadManager(m *Manager) {
	s.manager.Store(m)
}

func (s *Scheduler) executeAuth(ctx context.Context) error {
	auths := s.manager.Load().Authenticators()
	if len(auths) == 0 {
		return nil
	}
	token := api.AuthTokenFrom(ctx)
	for _, a := range auths {
		if _, err := a.Authenticate(ctx, token); err == nil {
			return nil
		}
	}
	return &api.PluginError{
		Code:      "AUTH_FAILED",
		Message:   "unauthorized",
		Retryable: false,
	}
}

func (s *Scheduler) executePipeline(ctx context.Context, model string, req any) (*api.RouteDecision, string, error) {
	// auth
	if err := s.executeAuth(ctx); err != nil {
		return nil, "", err
	}

	pctx := &api.PipelineContext{Context: ctx, Request: req, Meta: make(map[string]any)}

	// guard
	for _, g := range s.manager.Load().Guards() {
		if err := g.Guard(pctx); err != nil {
			return nil, "", err
		}
	}

	// preprocess
	for _, p := range s.manager.Load().PreProcessors() {
		if err := p.PreProcess(pctx); err != nil {
			return nil, "", err
		}
	}

	// route
	decision, err := s.manager.Load().Router().Route(ctx, model)
	if err != nil {
		return nil, "", err
	}
	pctx.Route = decision

	if decision.Model != "" {
		return decision, decision.Model, nil
	}
	return decision, model, nil
}

// ExecuteChat 执行完整 chat 管线：鉴权 → 路由 → 协议适配。
func (s *Scheduler) ExecuteChat(ctx context.Context, payload *api.ChatRequest) (*api.ChatResponse, error) {
	decision, actualModel, a, err := s.resolveAdapter(ctx, payload.Model, payload)
	if err != nil {
		return nil, err
	}
	payload.Model = actualModel
	resp, err := a.SendChat(ctx, payload, decision)
	if err != nil {
		return nil, err
	}
	if err := s.executePostProcess(ctx, payload, resp, decision); err != nil {
		return nil, err
	}
	return resp, nil
}

// ExecuteChatStream 执行流式 chat 管线，返回响应 channel。
func (s *Scheduler) ExecuteChatStream(ctx context.Context, payload *api.ChatRequest) (<-chan *api.StreamChunk, error) {
	decision, actualModel, a, err := s.resolveAdapter(ctx, payload.Model, payload)
	if err != nil {
		return nil, err
	}
	payload.Model = actualModel
	return a.SendChatStream(ctx, payload, decision)
}

func (s *Scheduler) resolveAdapter(ctx context.Context, model string, req any) (*api.RouteDecision, string, api.ChatAdapter,
	error,
) {
	decision, actualModel, err := s.executePipeline(ctx, model, req)
	if err != nil {
		return nil, "", nil, err
	}
	a, ok := s.manager.Load().Adapter(decision.Protocol)
	if !ok {
		return nil, "", nil, &api.PluginError{
			Code:      "adapter_not_found",
			Message:   fmt.Sprintf("adapter not found: %s", decision.Protocol),
			Retryable: false,
		}
	}
	return decision, actualModel, a, nil
}

func (s *Scheduler) executePostProcess(ctx context.Context, req any, resp any, decision *api.RouteDecision) error {
	postprocessors := s.manager.Load().PostProcessors()
	if len(postprocessors) == 0 {
		return nil
	}
	pctx := &api.PipelineContext{
		Context: ctx, Request: req, Response: resp, Route: decision,
		Meta: make(map[string]any),
	}
	for _, p := range postprocessors {
		if err := p.PostProcess(pctx); err != nil {
			return err
		}
	}
	return nil
}
