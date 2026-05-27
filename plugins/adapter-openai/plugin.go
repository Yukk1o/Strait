// Package adapter_openai 实现 OpenAI API 协议适配。
package adapter_openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"strait/api"
)

// openAIReq OpenAI API 请求体
type openAIReq struct {
	Model    string               `json:"model"`
	Messages []api.Message        `json:"messages"`
	Stream   bool                 `json:"stream"`
	Tools    []api.ToolDefinition `json:"tools,omitempty"`
}

// openAIStreamDelta SSE 流式响应中的增量数据
type openAIStreamDelta struct {
	Content   string                 `json:"content"`
	ToolCalls []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

// openAIStreamToolCall SSE 流式响应中的工具调用
type openAIStreamToolCall struct {
	Index    int                  `json:"index"`
	ID       string               `json:"id,omitempty"`
	Type     string               `json:"type,omitempty"`
	Function openAIStreamToolFunc `json:"function,omitempty"`
}

// openAIStreamToolFunc SSE 流式响应中的工具函数
type openAIStreamToolFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// openAIResp OpenAI API 响应体
type openAIResp struct {
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage"`
}

// openAIStreamResp SSE 流式响应中的单行数据
type openAIStreamResp struct {
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

// openAIStreamChoice SSE 流式响应中的选择体
type openAIStreamChoice struct {
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

// openAIChoice OpenAI API 选择体
type openAIChoice struct {
	Message      api.Message `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// openAIUsage OpenAI API 使用情况
type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Adapter OpenAI 协议适配器。
type Adapter struct {
	client *http.Client
}

func (a *Adapter) ID() string       { return "adapter-openai" }
func (a *Adapter) Protocol() string { return "openai" }
func init() {
	api.Register("adapter-openai", func() api.Plugin { return &Adapter{} })
}

func (a *Adapter) Init(config map[string]any) error {
	a.client = &http.Client{}
	return nil
}

func (a *Adapter) SendChat(ctx context.Context, payload *api.ChatRequest,
	route *api.RouteDecision,
) (*api.ChatResponse, error) {
	req, err := a.buildRequest(ctx, payload, route)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var openai openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&openai); err != nil {
		return nil, err
	}
	if len(openai.Choices) == 0 {
		return nil, &api.PluginError{
			Code:      "UPSTREAM_ERROR",
			Message:   "openAI response: choices is empty",
			Retryable: false,
		}
	}

	usage := api.Usage{}
	if openai.Usage != nil {
		usage = api.Usage{
			PromptTokens:     openai.Usage.PromptTokens,
			CompletionTokens: openai.Usage.CompletionTokens,
			TotalTokens:      openai.Usage.TotalTokens,
		}
	}

	choice := openai.Choices[0]
	return &api.ChatResponse{
		Model: openai.Model,
		Choices: []api.Choice{{
			Role:         choice.Message.Role,
			Content:      choice.Message.Content,
			FinishReason: choice.FinishReason,
			ToolCalls:    choice.Message.ToolCalls,
		}},
		Usage: usage,
	}, nil
}

func (a *Adapter) SendChatStream(ctx context.Context, payload *api.ChatRequest,
	route *api.RouteDecision,
) (<-chan *api.StreamChunk, error) {
	req, err := a.buildRequest(ctx, payload, route)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *api.StreamChunk, 8)
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if chunk, done := parseOpenAIStreamLine(line, payload.Model); done {
				break
			} else if chunk != nil {
				if chunk.Choices[0].Content == "" && chunk.Choices[0].FinishReason == "" && len(chunk.Choices[0].ToolCalls) == 0 {
					continue
				}
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
				if err != nil {
					break
				}
			}
		}
	}()

	return ch, nil
}

func (a *Adapter) buildRequest(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (*http.Request, error) {
	body := openAIReq{
		Model:    payload.Model,
		Messages: make([]api.Message, len(payload.Messages)),
		Stream:   payload.Stream,
	}
	for i, msg := range payload.Messages {
		body.Messages[i] = api.Message{Role: msg.Role, Content: msg.Content, ToolCalls: msg.ToolCalls}
	}
	if len(payload.Tools) > 0 {
		body.Tools = payload.Tools
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", route.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+route.APIKey)
	return req, nil
}

func parseOpenAIStreamLine(line string, model string) (*api.StreamChunk, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return nil, false
	}
	if line == "data: [DONE]" {
		return nil, true
	}
	var streamResp openAIStreamResp
	if err := json.Unmarshal([]byte(line[6:]), &streamResp); err != nil {
		return nil, false
	}

	if len(streamResp.Choices) == 0 {
		return nil, false
	}
	chunk := &api.StreamChunk{
		Model:   model,
		Choices: []api.Choice{{Content: streamResp.Choices[0].Delta.Content}},
	}
	if len(streamResp.Choices[0].Delta.ToolCalls) > 0 {
		chunk.Choices[0].ToolCalls = make([]api.ToolCall, len(streamResp.Choices[0].Delta.ToolCalls))
		for j, tc := range streamResp.Choices[0].Delta.ToolCalls {
			chunk.Choices[0].ToolCalls[j] = api.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: api.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}
	if streamResp.Choices[0].FinishReason != nil {
		chunk.Choices[0].FinishReason = *streamResp.Choices[0].FinishReason
	}
	if streamResp.Usage != nil {
		chunk.Usage = &api.Usage{
			PromptTokens:     streamResp.Usage.PromptTokens,
			CompletionTokens: streamResp.Usage.CompletionTokens,
			TotalTokens:      streamResp.Usage.TotalTokens,
		}
	}
	return chunk, false
}
