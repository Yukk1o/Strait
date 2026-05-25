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

func (s *Scheduler) executePipeline(ctx context.Context, model string) (*api.RouteDecision, error) {
	if err := s.executeAuth(ctx); err != nil {
		return nil, err
	}
	return s.manager.Load().Router().Route(ctx, model)
}

// ExecuteChat 执行完整 chat 管线：鉴权 → 路由 → 协议适配。
func (s *Scheduler) ExecuteChat(ctx context.Context, payload *api.ChatRequest) (*api.ChatResponse, error) {
	decision, err := s.executePipeline(ctx, payload.Model)
	if err != nil {
		return nil, err
	}
	a, ok := s.manager.Load().Adapter(decision.Protocol)
	if !ok {
		return nil, &api.PluginError{
			Code:      "adapter_not_found",
			Message:   fmt.Sprintf("adapter not found: %s", decision.Protocol),
			Retryable: false,
		}
	}
	return a.SendChat(ctx, payload, decision)
}

// ExecuteChatStream 执行流式 chat 管线，返回响应 channel。
func (s *Scheduler) ExecuteChatStream(ctx context.Context, payload *api.ChatRequest) (<-chan *api.StreamChunk, error) {
	decision, err := s.executePipeline(ctx, payload.Model)
	if err != nil {
		return nil, err
	}
	a, ok := s.manager.Load().Adapter(decision.Protocol)
	if !ok {
		return nil, &api.PluginError{
			Code:      "adapter_not_found",
			Message:   fmt.Sprintf("adapter not found: %s", decision.Protocol),
			Retryable: false,
		}
	}
	return a.SendChatStream(ctx, payload, decision)
}
