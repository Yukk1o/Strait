// Package api 定义 Strait 公共类型和接口——插件开发者唯一依赖
package api

import (
	"context"
	"encoding/json"
)

// ─ 消息构建块 ─

// Message 对话消息
type Message struct {
	Role      string     `json:"role"`                 // 角色：system/user/assistant
	Content   string     `json:"content"`              // 消息内容
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用
}

// Choice 回复选项
type Choice struct {
	Index        int        `json:"index"`                // 选项索引
	Role         string     `json:"role"`                 // 角色：system/user/assistant
	Content      string     `json:"content"`              // 消息内容
	FinishReason string     `json:"finish_reason"`        // 结束原因："stop" | "length"
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"` // 工具调用列表
}

// ─ Chat 请求/响应 ─

// ChatRequest 标准化请求体
type ChatRequest struct {
	Model    string           `json:"model"`           // 模型名称，如 gpt-4o
	Messages []Message        `json:"messages"`        // 对话消息列表
	Stream   bool             `json:"stream"`          // 是否流式返回
	Tools    []ToolDefinition `json:"tools,omitempty"` // 工具列表
}

// ChatResponse 标准化响应体
type ChatResponse struct {
	ID      string   `json:"id"`      // 请求唯一标识
	Model   string   `json:"model"`   // 实际使用的模型
	Choices []Choice `json:"choices"` // 回复选项列表
	Usage   Usage    `json:"usage"`   // 用量信息
}

// StreamChunk 流式响应单条消息
type StreamChunk struct {
	Choices []Choice `json:"choices"` // 流式块内增量数据
	Model   string   `json:"model"`   // 实际使用的模型
	Usage   *Usage   `json:"usage"`   // 用量信息（仅最后一条携带）
}

// Usage Token 用量统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入 token 数
	CompletionTokens int `json:"completion_tokens"` // 输出 token 数
	TotalTokens      int `json:"total_tokens"`      // 总 token 数
}

// ─ 工具调用（Function Calling） ─

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string       `json:"type"`     // 工具类型
	Function ToolFunction `json:"function"` // 工具函数
}

// ToolFunction 工具函数
type ToolFunction struct {
	Name        string          `json:"name"`                  // 函数名称
	Description string          `json:"description,omitempty"` // 函数描述
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // 函数参数
}

// ToolCall 工具调用
type ToolCall struct {
	Index    int              `json:"index,omitempty"` // 调用索引
	ID       string           `json:"id,omitempty"`    // 调用 ID
	Type     string           `json:"type,omitempty"`  // 调用类型
	Function ToolCallFunction `json:"function"`        // 调用函数
}

// ToolCallFunction 工具调用函数
type ToolCallFunction struct {
	Name      string `json:"name"`      // 函数名称
	Arguments string `json:"arguments"` // 函数参数
}

// ─ 路由/鉴权 ─

// RouteDecision 路由决策
type RouteDecision struct {
	Protocol string `json:"protocol"` // 调用协议：openAI / anthropic / deepseek / ollama
	BaseURL  string `json:"base_url"` // 调用地址
	APIKey   string `json:"api_key"`  // 认证密钥
}

// Subject 认证后的调用方信息
type Subject struct {
	ID string `json:"id"` // 调用方唯一标识
}

// ─ Context ─

type ctxKey string

const authTokenKey ctxKey = "auth_token"

func WithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey, token)
}

func AuthTokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(authTokenKey).(string)
	return t
}
