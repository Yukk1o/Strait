// Package plugin 提供插件注册、加载和调度功能
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

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

func (s *Scheduler) Manifest() []api.PluginDescriptor {
	return s.manager.Load().Manifest()
}

func (s *Scheduler) executeAuth(ctx context.Context) error {
	auths := s.manager.Load().Authenticators()
	if len(auths) == 0 {
		return nil
	}
	token := api.AuthTokenFrom(ctx)
	for _, a := range auths {
		if _, err := a.plugin.Authenticate(ctx, token); err == nil {
			return nil
		}
	}
	return api.NewPluginError(
		api.ErrCodeAuthFailed,
		"unauthorized",
		false,
	)
}

func (s *Scheduler) executePipeline(ctx context.Context, model string, req any) (*api.PipelineContext, *api.RouteDecision, string, error) {
	if ctx.Err() != nil {
		return nil, nil, "", api.NewPluginError(
			api.ErrCodeCanceled,
			"context canceled",
			false,
		)
	}

	// auth
	if err := s.executeAuth(ctx); err != nil {
		return nil, nil, "", err
	}

	copiedReq := req
	if chatReq, ok := req.(*api.ChatRequest); ok {
		copiedReq = api.CopyChatRequest(chatReq)
	}

	pctx := &api.PipelineContext{
		Context: ctx,
		TraceID: fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		Request: copiedReq,
		Meta:    make(map[string]any),
	}
	ctx = api.WithTraceID(ctx, pctx.TraceID)
	slog.Info("pipeline start", "trace_id", pctx.TraceID, "model", model)

	// guard
	if err := s.executeSteps(ctx, pctx, s.manager.Load().GuardSteps()); err != nil {
		return nil, nil, "", err
	}

	// preprocess
	if err := s.executeSteps(ctx, pctx, s.manager.Load().PreSteps()); err != nil {
		return nil, nil, "", err
	}

	// route
	decision, err := s.manager.Load().Router().Route(ctx, model)
	if err != nil {
		return nil, nil, "", err
	}
	pctx.Route = decision

	if decision.Model != "" {
		return pctx, decision, decision.Model, nil
	}
	return pctx, decision, model, nil
}

// ExecuteChat 执行完整 chat 管线：鉴权 → 路由 → 协议适配。
func (s *Scheduler) ExecuteChat(ctx context.Context, payload *api.ChatRequest) (*api.ChatResponse, error) {
	pctx, decision, actualModel, a, err := s.resolveAdapter(ctx, payload.Model, payload)
	if err != nil {
		return nil, err
	}
	req := pctx.Request.(*api.ChatRequest)
	req.Model = actualModel
	resp, err := a.SendChat(ctx, req, decision)
	if err != nil {
		return nil, err
	}
	pctx.Response = resp
	if err := s.executePostProcess(pctx); err != nil {
		return nil, err
	}
	return resp, nil
}

// ExecuteChatStream 执行流式 chat 管线，返回响应 channel。
func (s *Scheduler) ExecuteChatStream(ctx context.Context, payload *api.ChatRequest) (<-chan *api.StreamChunk, error) {
	pctx, decision, actualModel, a, err := s.resolveAdapter(ctx, payload.Model, payload)
	if err != nil {
		return nil, err
	}
	req := pctx.Request.(*api.ChatRequest)
	req.Model = actualModel
	ch, err := a.SendChatStream(ctx, req, decision)
	if err != nil {
		return nil, err
	}

	steps := s.manager.Load().PostSteps()
	if len(steps) > 0 {
		ch = s.wrapStreamPostProcess(pctx, ch, steps)
	}
	return ch, nil
}

func (s *Scheduler) resolveAdapter(ctx context.Context, model string, req any) (*api.PipelineContext, *api.RouteDecision, string, api.ChatAdapter, error) {
	pctx, decision, actualModel, err := s.executePipeline(ctx, model, req)
	if err != nil {
		return nil, nil, "", nil, err
	}
	a, ok := s.manager.Load().Adapter(decision.Protocol)
	if !ok {
		return nil, nil, "", nil, api.NewPluginError(
			api.ErrCodeAdapterNotFound,
			fmt.Sprintf("adapter not found: %s", decision.Protocol),
			false,
		)
	}
	return pctx, decision, actualModel, a, nil
}

func (s *Scheduler) executePostProcess(pctx *api.PipelineContext) error {
	steps := s.manager.Load().PostSteps()
	if len(steps) == 0 {
		return nil
	}
	return s.executeSteps(pctx.Context, pctx, steps)
}

func (s *Scheduler) wrapStreamPostProcess(
	pctx *api.PipelineContext,
	in <-chan *api.StreamChunk,
	steps []pipelineStep,
) <-chan *api.StreamChunk {
	ch := in
	for _, step := range steps {
		if step.streamExec == nil {
			continue
		}
		ch = step.streamExec(pctx, ch)
	}
	return ch
}

type pipelineStep struct {
	exec       func(pctx *api.PipelineContext) error
	streamExec func(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk
	failMode   api.FailMode
	timeout    time.Duration
}

func (s *Scheduler) executeSteps(ctx context.Context, pctx *api.PipelineContext, steps []pipelineStep) error {
	for _, step := range steps {
		if ctx.Err() != nil {
			return api.NewPluginError(api.ErrCodeCanceled, "context canceled", false)
		}

		var cancel context.CancelFunc
		var stepCtx context.Context
		if step.timeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, step.timeout)
			pctx.Context = stepCtx
		}

		err := step.exec(pctx)

		if err == nil && stepCtx != nil && stepCtx.Err() != nil {
			err = api.NewPluginError(
				api.ErrCodeCanceled,
				stepCtx.Err().Error(),
				false,
			)
		}

		if cancel != nil {
			pctx.Context = ctx
			cancel()
		}

		if err != nil {
			if step.failMode == api.FailSkip || step.failMode == api.FailFallback {
				continue
			}
			return err
		}
	}
	return nil
}
