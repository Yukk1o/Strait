# Strait vs Alternatives

English | [中文](../zh/comparison.md)

> Based on public documentation and common usage patterns. Each tool has its strengths; Strait focuses on pipeline programmability and plugin depth.

## Comparison

| Capability | LiteLLM | Kong AI Gateway | Strait |
|------------|---------|-----------------|--------|
| Positioning | Multi-provider proxy | API governance + AI extensions | Programmable AI request pipeline |
| Multi-provider routing | Excellent (100+ providers) | Strong + semantic routing | Model-level YAML routing + plugin |
| Request interception | Hooks callback | Generic plugins | Native six-stage pipeline |
| Stream post-processing | post_call (heavy) | Generic transformer | Native StreamPostProcessor |
| Deep request/response processing | Pre/Post hooks | AI Request/Response Transformer | Independent pre + post stages |
| Plugin mechanism | Python hooks + callbacks | Lua plugin system | Go native interfaces (type-safe) |
| Hot reload | Config reload | Mature support | Atomic pointer swap, zero interruption |
| Plugin self-description | None | Partial (plugin config) | Auto-generated manifest.json |

## Strait Unique Capabilities

- **Request deep-copy isolation** — pipeline entry deep-copies the request; preprocessor mutations don't affect the original payload
- **Plugin failure strategies** — strict (abort), skip (ignore + continue), fallback (use default) — handled by the scheduler, not the plugin
- **TraceID full-chain tracing** — each request gets a unique trace ID that flows through all six stages
- **Plugin self-description** — plugins declare their type, priority, failure strategy, and config schema via `Descriptor()`; a manifest is auto-generated on startup

## When to Choose Strait

- You need deep control over the **full AI request lifecycle** (not just routing)
- You want to write plugins in Go, not Python hooks or Lua scripts
- You need stream-aware post-processing (auditing, token accounting, content filtering)
- You need per-plugin failure isolation and fallback strategies

## When Not to Choose Strait

- You only need simple multi-provider routing → LiteLLM is simpler
- You need enterprise API gateway (rate limiting, quotas, developer portal) → Kong is more mature
- You don't need custom plugin logic → just use OpenAI SDK with env var switching
