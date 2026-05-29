// Package api 定义 Strait 公共类型和接口——插件开发者唯一依赖
package api

import (
	"context"
	"time"
)

// Plugin 所有插件的基础接口
type Plugin interface {
	ID() string
	Init(cfg map[string]any) error
}

// Authenticator 认证插件
type Authenticator interface {
	Plugin
	Authenticate(ctx context.Context, token string) (*Subject, error)
}

// Guard 安全防护插件
type Guard interface {
	Plugin
	Guard(pctx *PipelineContext) error
}

// PreProcessor 请求预处理插件
type PreProcessor interface {
	Plugin
	PreProcess(pctx *PipelineContext) error
}

// PostProcessor 响应后处理插件
type PostProcessor interface {
	Plugin
	PostProcess(pctx *PipelineContext) error
}

// StreamPostProcessor 流式响应后处理插件，可选实现，直接操作 StreamChunk channel
type StreamPostProcessor interface {
	Plugin
	PostProcessStream(pctx *PipelineContext, in <-chan *StreamChunk) <-chan *StreamChunk
}

// Router 路由决策插件
type Router interface {
	Plugin
	Route(ctx context.Context, model string) (*RouteDecision, error)
}

// ChatAdapter 协议适配插件
type ChatAdapter interface {
	Plugin
	Protocol() string
	SendChat(ctx context.Context, payload *ChatRequest, route *RouteDecision) (*ChatResponse, error)
	SendChatStream(ctx context.Context, payload *ChatRequest, route *RouteDecision) (<-chan *StreamChunk, error)
}

// FailMode 插件失败策略
type FailMode string

const (
	FailStrict   FailMode = "strict"   // 立即中断
	FailSkip     FailMode = "skip"     // 跳过继续
	FailFallback FailMode = "fallback" // 使用默认值
)

// PluginDescriptor 插件描述符
type PluginDescriptor struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Description   string         `json:"description"`
	Version       string         `json:"version"`
	Priority      int            `json:"priority"`          // 越小越先执行，默认 100
	Timeout       time.Duration  `json:"timeout,omitempty"` // 单插件超时，0 表示不限
	FailMode      FailMode       `json:"failMode"`          // 默认 strict
	ConfigSchema  map[string]any `json:"configSchema,omitempty"`
	DefaultConfig map[string]any `json:"defaultConfig,omitempty"`
}

// Describable 可选接口，插件实现此接口可声明自身元信息
type Describable interface {
	Descriptor() PluginDescriptor
}

// Describe 获取插件描述符，未实现 Describable 时返回默认值
func Describe(p Plugin) PluginDescriptor {
	if d, ok := p.(Describable); ok {
		return d.Descriptor()
	}
	return PluginDescriptor{ID: p.ID(), Priority: 100, FailMode: FailStrict}
}

// Constructor 插件构造函数
type Constructor func() Plugin

var constructors = map[string]Constructor{}

// Register 注册插件构造函数
func Register(name string, fn func() Plugin) {
	constructors[name] = fn
}

// CreatePlugin 根据名称创建插件实例
func CreatePlugin(name string) (Plugin, bool) {
	fn, ok := constructors[name]
	if !ok {
		return nil, false
	}
	return fn(), true
}
