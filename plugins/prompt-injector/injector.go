// Package prompt_injector 提供提示语注入功能(目前实现较为简单，后续可考虑使用更复杂的注入方式)
package prompt_injector

import (
	"log/slog"

	"strait/api"
)

type PromptInjector struct {
	SystemPrompt string
}

func (p *PromptInjector) ID() string { return "prompt-injector" }
func init() {
	api.Register("prompt-injector", func() api.Plugin { return &PromptInjector{} })
}

func (p *PromptInjector) Init(cfg map[string]any) error {
	if v, ok := cfg["system_prompt"].(string); ok {
		p.SystemPrompt = v
	}
	return nil
}

func (p *PromptInjector) PreProcess(pctx *api.PipelineContext) error {
	req, ok := pctx.Request.(*api.ChatRequest)
	if !ok {
		return nil
	}
	req.Messages = append([]api.Message{{Role: "system", Content: p.SystemPrompt}}, req.Messages...)
	slog.Info("injecting system prompt", "prompt", p.SystemPrompt, "messages_before", len(req.Messages))
	return nil
}
