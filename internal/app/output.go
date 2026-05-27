package app

import (
	"fmt"
	"time"

	"strait/api"
)

// ─ OpenAI 非流式响应结构 ─

// openAIResp 提供OpenAI API的响应结构
type openAIResp struct {
	ID      string         `json:"id"`      // 唯一标识
	Object  string         `json:"object"`  // 对象类型
	Created int64          `json:"created"` // 创建时间
	Model   string         `json:"model"`   // 模型名称
	Choices []openAIChoice `json:"choices"` // 选择项
	Usage   api.Usage      `json:"usage"`   // 使用情况
}

// openAIChoice 提供OpenAI API的响应结构
type openAIChoice struct {
	Index        int       `json:"index"`         // 选择索引
	Message      openAIMsg `json:"message"`       // 消息
	FinishReason string    `json:"finish_reason"` // 结束原因
}

// openAIMsg 提供OpenAI API的响应结构
type openAIMsg struct {
	Role      string         `json:"role"`                 // 角色
	Content   string         `json:"content"`              // 内容
	ToolCalls []api.ToolCall `json:"tool_calls,omitempty"` // 工具调用
}

// ─ OpenAI 流式响应结构 ─

// openAIStreamChunk 提供OpenAI API的流式响应结构

type openAIStreamChunk struct {
	ID      string               `json:"id"`              // 唯一标识
	Object  string               `json:"object"`          // 对象类型
	Created int64                `json:"created"`         // 创建时间
	Model   string               `json:"model"`           // 模型名称
	Choices []openAIStreamChoice `json:"choices"`         // 选择项
	Usage   *api.Usage           `json:"usage,omitempty"` // 新增
}

// openAIStreamChoice 提供OpenAI API的流式响应结构
type openAIStreamChoice struct {
	Index        int               `json:"index"`         // 选择索引
	Delta        openAIStreamDelta `json:"delta"`         // 增量
	FinishReason *string           `json:"finish_reason"` // 结束原因
}

// openAIStreamDelta 提供OpenAI API的流式响应结构
type openAIStreamDelta struct {
	Role      string         `json:"role,omitempty"`       // 角色
	Content   string         `json:"content,omitempty"`    // 内容
	ToolCalls []api.ToolCall `json:"tool_calls,omitempty"` // 工具调用
}

func generateReqID() string {
	return fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
}

func toOpenAIResp(id string, now int64, resp *api.ChatResponse) *openAIResp {
	choices := make([]openAIChoice, len(resp.Choices))
	for i, c := range resp.Choices {
		role := c.Role
		if role == "" {
			role = "assistant"
		}
		msg := openAIMsg{Role: role, Content: c.Content}
		if len(c.ToolCalls) > 0 {
			msg.ToolCalls = c.ToolCalls
		}
		choices[i] = openAIChoice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}
	return &openAIResp{
		ID:      id,
		Object:  "chat.completion",
		Created: now,
		Model:   resp.Model,
		Choices: choices,
		Usage:   resp.Usage,
	}
}

func toOpenAIStreamChunk(id string, now int64, first bool, chunk *api.StreamChunk) *openAIStreamChunk {
	choices := make([]openAIStreamChoice, len(chunk.Choices))
	for i, c := range chunk.Choices {
		var finishReason *string
		if c.FinishReason != "" {
			finishReason = &c.FinishReason
		}
		delta := openAIStreamDelta{Content: c.Content}
		if len(c.ToolCalls) > 0 {
			delta.ToolCalls = c.ToolCalls
		}
		if first {
			delta.Role = "assistant"
		}
		choices[i] = openAIStreamChoice{
			Index:        c.Index,
			Delta:        delta,
			FinishReason: finishReason,
		}
	}
	var usage *api.Usage
	if chunk.Usage != nil {
		usage = chunk.Usage
	}
	return &openAIStreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: now,
		Model:   chunk.Model,
		Choices: choices,
		Usage:   usage,
	}
}
