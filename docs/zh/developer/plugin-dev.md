# Strait 插件开发文档

[English](../../en/developer/plugin-dev.md) | 中文

> 给插件开发者看的。你只需要 `strait/api` 一个依赖。

## 插件类型

| 类型 | 接口 | 职责 |
|------|------|------|
| Authenticator | `api.Authenticator` | 鉴权 |
| Guard | `api.Guard` | 请求拦截（限流、IP 黑名单、PII 检测） |
| PreProcessor | `api.PreProcessor` | 请求预处理（提示注入、上下文扩充、脱敏） |
| Router | `api.Router` | 根据请求选择后端 |
| ChatAdapter | `api.ChatAdapter` | 与后端通信，转换协议 |
| PostProcessor | `api.PostProcessor` | 响应后处理（审计、成本统计） |
| PostProcessor (流式) | `api.StreamPostProcessor` | 流式响应后处理，直接操作 StreamChunk channel |

## 基础接口

所有插件必须实现 `api.Plugin`：

```go
type Plugin interface {
    ID() string                       // 唯一标识
    Init(config map[string]any) error // 由管理器调用
}
```

## 插件描述符（Descriptor）

插件可选实现 `api.Describable` 接口，声明自身元信息。未实现时系统使用默认值（Priority=100, FailMode=strict）。

```go
type Describable interface {
    Descriptor() PluginDescriptor
}

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
```

启动时系统自动调用 `api.Describe(p)` 获取描述符，并在 `manifest.json` 中汇总输出。Pipeline Editor 工具可读取此文件自动生成插件配置表单。

### 失败策略（FailMode）

```go
type FailMode string

const (
    FailStrict   FailMode = "strict"   // 立即中断管线
    FailSkip     FailMode = "skip"     // 跳过此插件，继续执行
    FailFallback FailMode = "fallback" // 使用默认值继续
)
```

失败策略由调度器处理，插件作者只需返回 error，不需要自己实现降级逻辑。

### 优先级与配置覆盖

`plugins.yaml` 中可覆盖描述符中的 `priority` 和 `fail_mode`：

```yaml
plugins:
  - id: my-plugin
    type: guard
    priority: 20        # 覆盖 Descriptor() 中的 Priority
    fail_mode: skip     # 覆盖 Descriptor() 中的 FailMode
    config:
      limit: 100
```

`config` 会与 `DefaultConfig` 合并（用户配置优先）。

## 注册规范

所有插件通过 `api.Register()` 注册，`plugins.yaml` 加载：

```go
func init() {
    api.Register("my-plugin-id", func() api.Plugin {
        return &MyPlugin{}
    })
}
```

Adapter 额外实现 `Protocol()` 返回协议标识（如 `"openai"`），Scheduler 按 `providers.yaml` 中的 `protocol` 字段匹配。

Router 通过 `api.Router` 接口注册，启动时由 Loader 自动注入 Manager。

## 实现示例

### Guard 插件

```go
type RateLimiter struct {
    limit int
    count int
}

func (r *RateLimiter) ID() string { return "rate-limiter" }
func (r *RateLimiter) Init(cfg map[string]any) error {
    r.limit = int(cfg["limit"].(float64))
    return nil
}

func (r *RateLimiter) Guard(pctx *api.PipelineContext) error {
    r.count++
    if r.count > r.limit {
        return api.NewPluginError(api.ErrCodeRateLimited, "too many requests", true)
    }
    return nil
}

func (r *RateLimiter) Descriptor() api.PluginDescriptor {
    return api.PluginDescriptor{
        ID:       "rate-limiter",
        Type:     "guard",
        Priority: 20,
        FailMode: api.FailStrict,
        ConfigSchema: map[string]any{
            "limit": map[string]any{"type": "number", "description": "max requests"},
        },
        DefaultConfig: map[string]any{
            "limit": 100,
        },
    }
}
```

### ChatAdapter

```go
type Adapter struct {
    client *http.Client
}

func (a *Adapter) ID() string       { return "adapters-my-protocol" }
func (a *Adapter) Protocol() string { return "my-protocol" }

func (a *Adapter) Init(config map[string]any) error {
    a.client = &http.Client{}
    return nil
}

func (a *Adapter) Descriptor() api.PluginDescriptor {
    return api.PluginDescriptor{
        ID: "adapters-my-protocol", Type: "adapter",
        Description: "My Protocol Adapter", Version: "0.3.0",
        Priority: 100, FailMode: api.FailStrict,
    }
}

func (a *Adapter) SendChat(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (*api.ChatResponse, error) {
    // 实现同步请求
}

func (a *Adapter) SendChatStream(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (<-chan *api.StreamChunk, error) {
    // 实现流式请求
}
```

### StreamPostProcessor（流式后处理）

实现 `api.StreamPostProcessor` 可在流式模式下直接操作 chunk channel：

```go
type TokenCounter struct{}

func (t *TokenCounter) ID() string                       { return "token-counter" }
func (t *TokenCounter) Init(_ map[string]any) error      { return nil }
func (t *TokenCounter) PostProcess(pctx *api.PipelineContext) error { return nil } // 非流式兜底

func (t *TokenCounter) PostProcessStream(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk {
    out := make(chan *api.StreamChunk)
    go func() {
        defer close(out)
        for chunk := range in {
            // 可在此统计 token、修改 chunk、注入额外数据
            out <- chunk
        }
    }()
    return out
}
```

> **注意**：PostProcessor 如果未实现 `StreamPostProcessor`，在流式模式下会被跳过，启动时会打印警告日志。

## 错误处理

使用 `api.NewPluginError` 创建结构化错误：

```go
// 错误码常量
const (
    ErrCodeAuthFailed      ErrCode = "AUTH_FAILED"
    ErrCodeNoRoute         ErrCode = "NO_ROUTE"
    ErrCodeAdapterNotFound ErrCode = "ADAPTER_NOT_FOUND"
    ErrCodeRateLimited     ErrCode = "RATE_LIMITED"
    ErrCodeUpstreamError   ErrCode = "UPSTREAM_ERROR"
    ErrCodeInvalidRequest  ErrCode = "INVALID_REQUEST"
    ErrCodeGuardRejected   ErrCode = "GUARD_REJECTED"
    ErrCodeCanceled        ErrCode = "CANCELED"
)

return api.NewPluginError(api.ErrCodeRateLimited, "rate limit exceeded", true)
// 参数：错误码, 可读消息, 是否可重试
```

## PipelineContext

管线执行过程中，所有 Guard / PreProcessor / PostProcessor 收到的 `PipelineContext`：

```go
type PipelineContext struct {
    TraceID  string          // 请求唯一标识（自动生成，用于全链路追踪）
    Context  context.Context // 请求上下文（含 TraceID、AuthToken）
    Request  any             // *ChatRequest（深拷贝，预处理器修改不影响原始 payload）
    Response any             // *ChatResponse / nil
    Route    *RouteDecision  // 路由决策 / nil
    Meta     map[string]any  // 插件间共享数据
}
```

- `TraceID` 自动注入，贯穿全部六个阶段，日志中可通过 `trace_id` 字段关联
- `Request` 在管线入口自动深拷贝，预处理器可安全修改

## 生命周期

```
编译时：init() 调用 api.Register() 注册插件工厂
启动时：Loader 读取 plugins.yaml → 调用工厂创建实例 → Describe() → mergeConfig(DefaultConfig, userConfig) → Init()
运行时：Scheduler 按优先级排序后按管线阶段调用
重载时：Loader 重新加载 → ReloadManager() → 原子指针替换 → 重新生成 manifest.json
```

## 约束

- 插件不引用 `internal/` 下的任何包，只依赖 `strait/api`
- 插件不能调用其他插件
- 插件不直接访问 Manager 或 Scheduler
- Adapter 不假设请求/响应结构 —— 使用 `api.ChatRequest` / `api.ChatResponse`
