# Strait Plugin Development Guide

English | [中文](../../zh/developer/plugin-dev.md)

> For plugin developers. You only need `strait/api` as a dependency.

## Plugin Types

| Type | Interface | Responsibility |
|------|-----------|---------------|
| Authenticator | `api.Authenticator` | Authentication |
| Guard | `api.Guard` | Request interception (rate limiting, IP blacklist, PII detection) |
| PreProcessor | `api.PreProcessor` | Request preprocessing (prompt injection, context enrichment, redaction) |
| Router | `api.Router` | Route requests to backends |
| ChatAdapter | `api.ChatAdapter` | Protocol conversion, backend communication |
| PostProcessor | `api.PostProcessor` | Response post-processing (auditing, cost tracking) |
| PostProcessor (stream) | `api.StreamPostProcessor` | Stream-aware post-processing, operates on StreamChunk channel |

## Base Interface

All plugins must implement `api.Plugin`:

```go
type Plugin interface {
    ID() string                       // Unique identifier
    Init(config map[string]any) error // Called by manager
}
```

## Plugin Descriptor

Plugins can optionally implement `api.Describable` to declare metadata. Defaults: Priority=100, FailMode=strict.

```go
type Describable interface {
    Descriptor() PluginDescriptor
}

type PluginDescriptor struct {
    ID            string         `json:"id"`
    Type          string         `json:"type"`
    Description   string         `json:"description"`
    Version       string         `json:"version"`
    Priority      int            `json:"priority"`          // Lower = first, default 100
    Timeout       time.Duration  `json:"timeout,omitempty"` // Per-plugin timeout, 0 = unlimited
    FailMode      FailMode       `json:"failMode"`          // Default strict
    ConfigSchema  map[string]any `json:"configSchema,omitempty"`
    DefaultConfig map[string]any `json:"defaultConfig,omitempty"`
}
```

On startup, the system calls `api.Describe(p)` to get descriptors and aggregates them into `manifest.json`. The Pipeline Editor tool reads this file to auto-generate config forms.

### Fail Strategies (FailMode)

```go
type FailMode string

const (
    FailStrict   FailMode = "strict"   // Abort pipeline immediately
    FailSkip     FailMode = "skip"     // Skip this plugin, continue
    FailFallback FailMode = "fallback" // Use default value, continue
)
```

Fail strategies are handled by the scheduler — plugin authors just return errors, no degradation logic needed.

### Priority and Config Override

`plugins.yaml` can override `priority` and `fail_mode` from the descriptor:

```yaml
plugins:
  - id: my-plugin
    type: guard
    priority: 20        # Overrides Descriptor() Priority
    fail_mode: skip     # Overrides Descriptor() FailMode
    config:
      limit: 100
```

`config` is merged with `DefaultConfig` (user config takes precedence).

## Registration

All plugins register via `api.Register()`, loaded by `plugins.yaml`:

```go
func init() {
    api.Register("my-plugin-id", func() api.Plugin {
        return &MyPlugin{}
    })
}
```

Adapters additionally implement `Protocol()` returning a protocol identifier (e.g. `"openai"`). The Scheduler matches against the `protocol` field in `providers.yaml`.

Router registers via the `api.Router` interface, auto-injected into Manager by the Loader on startup.

## Examples

### Guard Plugin

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
    // Implement sync request
}

func (a *Adapter) SendChatStream(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (<-chan *api.StreamChunk, error) {
    // Implement streaming request
}
```

### StreamPostProcessor

Implement `api.StreamPostProcessor` to operate directly on the chunk channel in streaming mode:

```go
type TokenCounter struct{}

func (t *TokenCounter) ID() string                       { return "token-counter" }
func (t *TokenCounter) Init(_ map[string]any) error      { return nil }
func (t *TokenCounter) PostProcess(pctx *api.PipelineContext) error { return nil } // Non-stream fallback

func (t *TokenCounter) PostProcessStream(pctx *api.PipelineContext, in <-chan *api.StreamChunk) <-chan *api.StreamChunk {
    out := make(chan *api.StreamChunk)
    go func() {
        defer close(out)
        for chunk := range in {
            // Count tokens, modify chunks, inject extra data
            out <- chunk
        }
    }()
    return out
}
```

> **Note**: If a PostProcessor does not implement `StreamPostProcessor`, it will be skipped in streaming mode with a warning log on startup.

## Error Handling

Use `api.NewPluginError` to create structured errors:

```go
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
// Args: error code, human-readable message, is retryable
```

## PipelineContext

During pipeline execution, all Guard / PreProcessor / PostProcessor plugins receive a `PipelineContext`:

```go
type PipelineContext struct {
    TraceID  string          // Unique request ID (auto-generated, for full-chain tracing)
    Context  context.Context // Request context (contains TraceID, AuthToken)
    Request  any             // *ChatRequest (deep copy, preprocessor mutations don't affect original)
    Response any             // *ChatResponse / nil
    Route    *RouteDecision  // Routing decision / nil
    Meta     map[string]any  // Shared data between plugins
}
```

- `TraceID` is auto-injected across all six stages, correlatable via `trace_id` in logs
- `Request` is auto-deep-copied at pipeline entry, preprocessors can safely modify it

## Lifecycle

```
Compile time: init() calls api.Register() to register plugin factory
Startup:      Loader reads plugins.yaml → creates instance via factory → Describe() → mergeConfig(DefaultConfig, userConfig) → Init()
Runtime:      Scheduler calls plugins by priority order per pipeline stage
Reload:       Loader reloads → ReloadManager() → atomic pointer swap → regenerate manifest.json
```

## Constraints

- Plugins must not import anything under `internal/`, only depend on `strait/api`
- Plugins cannot call other plugins
- Plugins must not directly access Manager or Scheduler
- Adapters must not assume request/response structures — use `api.ChatRequest` / `api.ChatResponse`
