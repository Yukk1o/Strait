package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"sync"

	"strait/internal/config"

	"strait/api"

	"github.com/goccy/go-yaml"
)

// providerYAML 定义了提供者的 YAML 配置结构
type providerYAML struct {
	ID        string   `yaml:"id"`          // 唯一标识
	Protocol  string   `yaml:"protocol"`    // 调用协议：openAI / anthropic / deepseek / ollama
	BaseURL   string   `yaml:"base_url"`    // API 接口地址
	APIKeyEnv string   `yaml:"api_key_env"` // 存储 API Key 的环境变量名
	Models    []string `yaml:"models"`      // 支持的模型名称列表
}

// targetYAML 定义路由指向的目标提供者与模型
type targetYAML struct {
	Provider string `yaml:"provider"` // 目标提供者 ID
	Model    string `yaml:"model"`    // 目标模型名称
	Priority int    `yaml:"priority"` // 路由优先级
	Weight   int    `yaml:"weight"`   // 路由权重
}

// routeYAML 定义路由规则的 YAML 配置结构
type routeYAML struct {
	ID       string       `yaml:"id"`       // 路由唯一标识
	Match    matchYAML    `yaml:"match"`    // 匹配表达式（模型名或通配符）
	Target   targetYAML   `yaml:"target"`   // 路由目标
	Targets  []targetYAML `yaml:"targets"`  // 路由目标列表
	Strategy string       `yaml:"strategy"` // 路由策略(priority / weight)
}

// matchYAML 定义匹配规则的 YAML 配置结构
type matchYAML struct {
	Model string `yaml:"model"` // 请求匹配的模型名称
}

// Router 定义了路由器结构
type Router struct {
	mu            sync.RWMutex            // 读写锁
	providers     map[string]providerYAML // 提供者配置映射
	routes        []routeYAML             // 路由规则列表
	providersPath string                  // 提供者配置文件路径
	routesPath    string                  // 路由规则文件路径
}

func (r *Router) ID() string { return "router-yaml" }

func init() {
	api.Register("router-yaml", func() api.Plugin { return &Router{} })
}

func (r *Router) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{
		ID:          "router-yaml",
		Type:        "router",
		Description: "YAML 配置路由插件",
		Version:     "0.3.0",
		Priority:    100,
		FailMode:    api.FailStrict,
	}
}

func (r *Router) loadConfig() (map[string]providerYAML, []routeYAML, error) {
	// 1. 读取并解析提供者配置
	data, err := os.ReadFile(r.providersPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read providers.yaml failed: %w", err)
	}

	var ps struct {
		Providers []providerYAML `yaml:"providers"`
	}
	if err := yaml.Unmarshal(data, &ps); err != nil {
		return nil, nil, fmt.Errorf("unmarshal providers failed: %w", err)
	}

	// 2. 加载提供者数据并合法性校验
	r.providers = make(map[string]providerYAML)
	for _, p := range ps.Providers {
		if p.ID == "" {
			return nil, nil, errors.New("provider ID cannot be empty")
		}
		if _, exists := r.providers[p.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate provider ID: %s", p.ID)
		}
		r.providers[p.ID] = p
	}

	// 3. 读取并解析路由规则
	data, err = os.ReadFile(r.routesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read routes.yaml failed: %w", err)
	}

	var rs struct {
		Routes []routeYAML `yaml:"routes"`
	}
	if err := yaml.Unmarshal(data, &rs); err != nil {
		return nil, nil, fmt.Errorf("unmarshal routes failed: %w", err)
	}

	// 4. 校验路由关联有效性
	for _, route := range rs.Routes {
		targets := route.Targets
		if len(route.Targets) > 0 && route.Target.Provider != "" {
			slog.Warn("both target and targets defined, using targets", "route", route.ID)
		}
		if len(targets) == 0 {
			if route.Target.Provider == "" {
				return nil, nil, fmt.Errorf("route %s has no target", route.ID)
			}
			targets = []targetYAML{route.Target}
		}
		for _, target := range targets {
			providerID := target.Provider
			if _, exists := r.providers[providerID]; !exists {
				return nil, nil, fmt.Errorf("route %s references unknown provider: %s", route.ID, providerID)
			}
		}
	}
	return r.providers, rs.Routes, nil
}

func (r *Router) Init(cfg map[string]any) error {
	r.providersPath = config.ProvidersPath
	r.routesPath = config.RoutesPath

	if v, ok := cfg["providers_path"].(string); ok {
		r.providersPath = v
	}
	if v, ok := cfg["routes_path"].(string); ok {
		r.routesPath = v
	}

	return r.Reload()
}

func (r *Router) Reload() error {
	newProviders, newRouters, err := r.loadConfig()
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.providers = newProviders
	r.routes = newRouters
	r.mu.Unlock()
	return nil
}

func (r *Router) Route(ctx context.Context, model string) (*api.RouteDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rt := range r.routes {
		if rt.Match.Model == model {
			return r.resolveRoute(ctx, rt, model)
		}
	}

	for _, rt := range r.routes {
		if strings.Contains(rt.Match.Model, "*") && matchModel(rt.Match.Model, model) {
			return r.resolveRoute(ctx, rt, model)
		}
	}

	return nil, &api.PluginError{
		Code:      "NO_ROUTE",
		Message:   fmt.Sprintf("no route for model: %s", model),
		Retryable: false,
	}
}

func (r *Router) resolveRoute(ctx context.Context, rt routeYAML, model string) (*api.RouteDecision, error) {
	// 获取 targets 列表
	targets := rt.Targets
	if len(targets) == 0 {
		targets = []targetYAML{rt.Target}
	}

	// 按 strategy 选择
	selected := r.selectTarget(targets, rt.Strategy)

	if p, ok := r.providers[selected.Provider]; ok {
		apiKey := os.Getenv(p.APIKeyEnv)
		if apiKey == "" {
			return nil, &api.PluginError{
				Code:      "NO_ROUTE",
				Message:   fmt.Sprintf("provider %s API key not set in env %s", p.ID, p.APIKeyEnv),
				Retryable: false,
			}
		}

		slog.Info(
			"route selected",
			"trace_id", api.TraceIDFrom(ctx),
			"route", rt.ID,
			"model", model,
			"provider", selected.Provider,
			"strategy", rt.Strategy,
		)

		return &api.RouteDecision{
			Protocol: p.Protocol,
			BaseURL:  p.BaseURL,
			APIKey:   apiKey,
			Model:    selected.Model,
		}, nil
	}

	return nil, &api.PluginError{
		Code:      "NO_ROUTE",
		Message:   fmt.Sprintf("no route for model: %s", model),
		Retryable: false,
	}
}

func matchModel(pattern, model string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(model, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == model
}

func (r *Router) selectTarget(targets []targetYAML, strategy string) targetYAML {
	if len(targets) == 1 {
		return targets[0]
	}

	switch strategy {
	case "priority":
		// 按优先级选择
		best := targets[0]
		for _, t := range targets {
			if t.Priority < best.Priority {
				best = t
			}
		}
		return best
	case "weight":
		// 按权重选择
		return r.selectByWeight(targets)
	default:
		return r.selectTarget(targets, "priority")
	}
}

func (r *Router) selectByWeight(targets []targetYAML) targetYAML {
	totalWeight := 0
	for _, t := range targets {
		totalWeight += t.Weight
	}
	randWeight := rand.Intn(totalWeight)
	for _, t := range targets {
		randWeight -= t.Weight
		if randWeight < 0 {
			return t
		}
	}
	return targets[0]
}
