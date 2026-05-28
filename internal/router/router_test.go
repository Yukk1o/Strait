package router

import (
	"context"
	"os"
	"testing"

	"strait/api"
)

// ── matchModel ──

func TestMatchModel_Exact(t *testing.T) {
	if !matchModel("deepseek-chat", "deepseek-chat") {
		t.Fatal("expected exact match")
	}
	if matchModel("deepseek-chat", "gpt-4") {
		t.Fatal("expected no match")
	}
}

func TestMatchModel_Wildcard(t *testing.T) {
	if !matchModel("deepseek-*", "deepseek-chat") {
		t.Fatal("expected wildcard match")
	}
	if !matchModel("deepseek-*", "deepseek-reasoner") {
		t.Fatal("expected wildcard match")
	}
	if matchModel("deepseek-*", "gpt-4") {
		t.Fatal("expected wildcard no match")
	}
}

func TestMatchModel_MatchAll(t *testing.T) {
	if !matchModel("*", "anything") {
		t.Fatal("expected * to match anything")
	}
}

// ── selectTarget ──

func TestSelectTarget_Single(t *testing.T) {
	r := &Router{}
	targets := []targetYAML{{Provider: "a", Priority: 1}}
	got := r.selectTarget(targets, "priority")
	if got.Provider != "a" {
		t.Fatalf("expected a, got %s", got.Provider)
	}
}

func TestSelectTarget_Priority(t *testing.T) {
	r := &Router{}
	targets := []targetYAML{
		{Provider: "low", Priority: 2},
		{Provider: "high", Priority: 1},
		{Provider: "mid", Priority: 3},
	}
	got := r.selectTarget(targets, "priority")
	if got.Provider != "high" {
		t.Fatalf("expected high, got %s", got.Provider)
	}
}

func TestSelectTarget_DefaultStrategy(t *testing.T) {
	r := &Router{}
	targets := []targetYAML{
		{Provider: "b", Priority: 2},
		{Provider: "a", Priority: 1},
	}
	got := r.selectTarget(targets, "")
	if got.Provider != "a" {
		t.Fatalf("expected a (default to priority), got %s", got.Provider)
	}
}

func TestSelectTarget_Weight(t *testing.T) {
	r := &Router{}
	targets := []targetYAML{
		{Provider: "heavy", Weight: 90},
		{Provider: "light", Weight: 10},
	}
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		got := r.selectTarget(targets, "weight")
		counts[got.Provider]++
	}
	if counts["heavy"] < 700 || counts["heavy"] > 950 {
		t.Fatalf("expected ~90%% heavy, got heavy=%d light=%d", counts["heavy"], counts["light"])
	}
}

// ── Route (端到端，绕过文件 I/O) ──

func newTestRouter() *Router {
	return &Router{
		providers: map[string]providerYAML{
			"deepseek-main": {
				ID:        "deepseek-main",
				Protocol:  "openai",
				BaseURL:   "https://api.deepseek.com/v1",
				APIKeyEnv: "TEST_DEEPSEEK_KEY",
			},
			"ollama-local": {
				ID:        "ollama-local",
				Protocol:  "ollama",
				BaseURL:   "http://localhost:11434",
				APIKeyEnv: "TEST_OLLAMA_KEY",
			},
		},
		routes: []routeYAML{
			{
				ID:       "exact-route",
				Match:    matchYAML{Model: "deepseek-chat"},
				Strategy: "priority",
				Targets: []targetYAML{
					{Provider: "deepseek-main", Model: "deepseek-chat", Priority: 1, Weight: 1},
				},
			},
			{
				ID:       "wildcard-route",
				Match:    matchYAML{Model: "ollama-*"},
				Strategy: "priority",
				Targets: []targetYAML{
					{Provider: "ollama-local", Model: "qwen2.5:0.5b", Priority: 1, Weight: 1},
				},
			},
			{
				ID:    "legacy-route",
				Match: matchYAML{Model: "deepseek-reasoner"},
				Target: targetYAML{
					Provider: "deepseek-main",
					Model:    "deepseek-reasoner",
				},
			},
		},
	}
}

func TestRoute_ExactMatch(t *testing.T) {
	r := newTestRouter()
	os.Setenv("TEST_DEEPSEEK_KEY", "sk-test")
	defer os.Unsetenv("TEST_DEEPSEEK_KEY")

	decision, err := r.Route(context.Background(), "deepseek-chat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Protocol != "openai" {
		t.Fatalf("expected openai, got %s", decision.Protocol)
	}
	if decision.Model != "deepseek-chat" {
		t.Fatalf("expected deepseek-chat, got %s", decision.Model)
	}
}

func TestRoute_WildcardMatch(t *testing.T) {
	r := newTestRouter()
	os.Setenv("TEST_OLLAMA_KEY", "sk-test")
	defer os.Unsetenv("TEST_OLLAMA_KEY")

	decision, err := r.Route(context.Background(), "ollama-llama3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Protocol != "ollama" {
		t.Fatalf("expected ollama, got %s", decision.Protocol)
	}
	if decision.Model != "qwen2.5:0.5b" {
		t.Fatalf("expected qwen2.5:0.5b, got %s", decision.Model)
	}
}

func TestRoute_NoMatch(t *testing.T) {
	r := newTestRouter()

	_, err := r.Route(context.Background(), "unknown-model")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
	pe, ok := err.(*api.PluginError)
	if !ok {
		t.Fatalf("expected PluginError, got %T", err)
	}
	if pe.Code != "NO_ROUTE" {
		t.Fatalf("expected NO_ROUTE, got %s", pe.Code)
	}
}

func TestRoute_MissingAPIKey(t *testing.T) {
	r := newTestRouter()
	os.Unsetenv("TEST_DEEPSEEK_KEY")

	_, err := r.Route(context.Background(), "deepseek-chat")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestRoute_LegacyTarget(t *testing.T) {
	r := newTestRouter()
	os.Setenv("TEST_DEEPSEEK_KEY", "sk-test")
	defer os.Unsetenv("TEST_DEEPSEEK_KEY")

	decision, err := r.Route(context.Background(), "deepseek-reasoner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Model != "deepseek-reasoner" {
		t.Fatalf("expected deepseek-reasoner, got %s", decision.Model)
	}
}
