// Package api 定义 Strait 公共类型和接口——插件开发者唯一依赖
package api

import "context"

// Plugin 所有插件的基础接口
type Plugin interface {
	ID() string
	Init(cfg map[string]any) error // 由插件管理器调用
}

// Authenticator 认证插件
type Authenticator interface {
	Plugin
	Authenticate(ctx context.Context, token string) (*Subject, error)
}

// Router 根据请求内容选择目标后端
type Router interface {
	Plugin
	Route(ctx context.Context, model string) (*RouteDecision, error)
}

// ChatAdapter chat 协议适配器
type ChatAdapter interface {
	Plugin
	Protocol() string
	SendChat(ctx context.Context, payload *ChatRequest, route *RouteDecision) (*ChatResponse, error)
	SendChatStream(ctx context.Context, payload *ChatRequest, route *RouteDecision) (<-chan *StreamChunk, error)
}

type Guard interface {
	Plugin
	Guard(pctx *PipelineContext) error
}

type PreProcessor interface {
	Plugin
	PreProcess(pctx *PipelineContext) error
}

type PostProcessor interface {
	Plugin
	PostProcess(pctx *PipelineContext) error
}

// Constructor 插件构造函数
type Constructor func() Plugin

var constructors = map[string]Constructor{}

func Register(name string, fn func() Plugin) {
	constructors[name] = fn
}

func CreatePlugin(name string) (Plugin, bool) {
	fn, ok := constructors[name]
	if !ok {
		return nil, false
	}
	return fn(), true
}
