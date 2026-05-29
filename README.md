# Strait

> Not a proxy. An operating system for your AI traffic.

English | [中文](README.zh.md)

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)

Strait is a plugin-driven AI request pipeline framework. It exposes an OpenAI-compatible API and routes requests through a six-stage plugin pipeline — authentication, guard, preprocessing, routing, adaptation, and post-processing — where every stage is programmable through Go plugins.

## Why Strait?

- **Not a thin proxy** — full six-stage request pipeline with guard, pre/post-processing stages
- **Not config-only** — write real Go logic for complex behaviors (PII detection, content filtering, cost tracking)
- **Not fragile** — plugin failure strategies (strict / skip / fallback), request deep-copy isolation, TraceID tracing
- **Multi-provider** — route to OpenAI, Ollama, or any backend via YAML config, with priority/weight failover

## Architecture

```
  Request ──▶ [Auth] ──▶ [Guard] ──▶ [Pre] ──▶ [Route] ──▶ [Adapter] ──▶ [Post] ──▶ Response
                                                    │
                                              ┌─────┴─────┐
                                              │  Router    │
                                              └─────┬─────┘
                                         ┌──────────┼──────────┐
                                      OpenAI    Ollama      ...
```

| Stage | Purpose | Example |
|-------|---------|---------|
| **Auth** | Token validation | `auth-static-token` |
| **Guard** | Rate limiting, IP blacklist, PII detection | — |
| **PreProcess** | Prompt injection, context enrichment | `prompt-injector` |
| **Route** | Model routing, A/B testing, failover | `router-yaml` |
| **Adapter** | Protocol conversion, backend calls | `adapter-openai`, `adapter-ollama` |
| **PostProcess** | Logging, token accounting, content filtering | — |

Plugins are sorted by priority (lower first). Failure is handled by each plugin's declared strategy — no boilerplate needed.

## Quick Start

```bash
go build -o strait ./cmd/strait/
export DEEPSEEK_API_KEY=sk-xxx
./strait
```

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-admin-init" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"hello"}]}'
```

## Write a Plugin

Implement `Plugin` + one stage interface, declare a `Descriptor()`, register via `api.Register()`. ~30 lines total.

```go
func (r *RateLimiter) Guard(pctx *api.PipelineContext) error {
    if r.count++; r.count > r.limit {
        return api.NewPluginError(api.ErrCodeRateLimited, "too many requests", true)
    }
    return nil
}
```

See [Plugin Development Guide](docs/en/developer/plugin-dev.md) for the full API reference.

## Examples

- **Weighted routing** — split `deepseek-chat` requests across Deepseek (75%) and local Ollama (25%) via YAML config
- **System prompt injection** — prepend a unified system prompt to every request, zero client changes
- **Static token auth** — protect endpoints with a fixed bearer token for internal deployments

See [examples/](examples/) for working configs and code.

## Tools UI

Browser-based Playground and Pipeline Editor: https://yukk1o.github.io/Strait/

## Documentation

| Document | Content |
|----------|---------|
| [Comparison](docs/en/comparison.md) | Strait vs LiteLLM / Kong |
| [Deployment](docs/en/deployment.md) | Docker, K8s, environment variables |
| [Built-in Plugins](docs/en/plugins/built-in.md) | Included plugins and configuration |
| [Plugin Dev Guide](docs/en/developer/plugin-dev.md) | How to write Strait plugins |
| [Roadmap](docs/en/roadmap.md) | Project evolution plan |
| [API Design](docs/en/api-design.md) | Public types and interface definitions |

## License

[Apache License 2.0](LICENSE)
