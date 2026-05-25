// Package adapter_ollama 实现 Ollama API 协议适配。
package adapter_ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"strait/api"
)

// ollamaToolCall Ollama API 工具调用
type ollamaToolCall struct {
	Type     string                 `json:"type"`
	Function ollamaToolCallFunction `json:"function"`
}

// ollamaToolCallFunction Ollama API 工具调用函数
type ollamaToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ollamaReq Ollama API 请求体
type ollamaReq struct {
	Model    string               `json:"model"`
	Messages []api.Message        `json:"messages"`
	Stream   bool                 `json:"stream"`
	Tools    []api.ToolDefinition `json:"tools,omitempty"`
}

// ollamaResp Ollama API 响应体
type ollamaResp struct {
	Message         ollamaMsg `json:"message"`
	Done            bool      `json:"done"`
	EvalCount       int       `json:"eval_count"`
	PromptEvalCount int       `json:"prompt_eval_count"`
}

type ollamaMsg struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

// Adapter Ollama 协议适配器。
type Adapter struct {
	client *http.Client
}

func (a *Adapter) ID() string       { return "adapter-ollama" }
func (a *Adapter) Protocol() string { return "ollama" }
func init() {
	api.Register("adapter-ollama", func() api.Plugin { return &Adapter{} })
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

	var ollama ollamaResp
	if err := json.NewDecoder(resp.Body).Decode(&ollama); err != nil {
		return nil, err
	}

	usage := api.Usage{}
	if ollama.Done {
		usage = api.Usage{
			PromptTokens:     ollama.PromptEvalCount,
			CompletionTokens: ollama.EvalCount,
			TotalTokens:      ollama.PromptEvalCount + ollama.EvalCount,
		}
	}

	ollamaChoice := ollama.Message
	choice := api.Choice{
		Content:      ollamaChoice.Content,
		FinishReason: "stop",
	}
	if len(ollamaChoice.ToolCalls) > 0 {
		choice.ToolCalls = make([]api.ToolCall, len(ollamaChoice.ToolCalls))
		for i, tc := range ollamaChoice.ToolCalls {
			choice.ToolCalls[i] = api.ToolCall{
				Type: tc.Type,
				Function: api.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: string(tc.Function.Arguments),
				},
			}
		}
	}
	return &api.ChatResponse{
		Model:   payload.Model,
		Choices: []api.Choice{choice},
		Usage:   usage,
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
			if chunk, done := parseOllamaStreamLine(line, payload.Model); done {
				if chunk != nil {
					select {
					case ch <- chunk:
					case <-ctx.Done():
						return
					}
				}
				break
			} else if chunk != nil {
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
	body := ollamaReq{
		Model:    payload.Model,
		Messages: payload.Messages,
		Stream:   payload.Stream,
	}
	if len(payload.Tools) > 0 {
		body.Tools = payload.Tools
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", route.BaseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func parseOllamaStreamLine(line string, model string) (*api.StreamChunk, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}
	var streamResp ollamaResp
	if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
		return nil, false
	}

	chunk := &api.StreamChunk{
		Model:   model,
		Choices: []api.Choice{{Content: streamResp.Message.Content}},
	}
	if streamResp.Done {
		chunk.Choices[0].FinishReason = "stop"
		if len(streamResp.Message.ToolCalls) > 0 {
			chunk.Choices[0].ToolCalls = make([]api.ToolCall, len(streamResp.Message.ToolCalls))
			for i, tc := range streamResp.Message.ToolCalls {
				chunk.Choices[0].ToolCalls[i] = api.ToolCall{
					Type: tc.Type,
					Function: api.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: string(tc.Function.Arguments),
					},
				}
			}
		}
		if streamResp.EvalCount > 0 {
			chunk.Usage = &api.Usage{
				PromptTokens:     streamResp.PromptEvalCount,
				CompletionTokens: streamResp.EvalCount,
				TotalTokens:      streamResp.PromptEvalCount + streamResp.EvalCount,
			}
		}
		return chunk, true
	}
	return chunk, false
}
