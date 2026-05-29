# Strait Roadmap

English | [中文](../zh/roadmap.md)

> AI flows through here — a plugin-driven AI gateway

## Principles

- **Core first**: ship the pipeline, then build around it
- **Plugin-driven**: all business logic decoupled via plugin interfaces
- **Config-driven**: YAML-driven, no enterprise data model
- **Runtime before persistence**: gateway is a runtime, not a database

---

## Core Roadmap

> Strait core delivery. Only routing, forwarding, auth, and pipeline — everything else as plugins.

| Phase | Status | Deliverables |
|-------|--------|-------------|
| **P0 MVP Skeleton** | ✅ | Project structure + API interfaces + YAML routing + OpenAI adapter + SSE streaming |
| **P1 Plugin Loader** | ✅ | Loader + auto-registration + config-driven + startup logs |
| **P2 Production Readiness** | ✅ | Auth + error model + hot reload + unified internal model + OpenAI output format |
| **P3 Agent Protocol** | ✅ | Function Calling + Tool Use + Tool Calls response |
| **P4 Multi-provider** | 🚧 | Ollama adapter ✅ / Load balancing (priority/weight) ✅ / Anthropic + Gemini ⏳ / Circuit breaker ⏳ |
| **P5 Pipeline Extension** | 🚧 | Guard / PreProcessor / PostProcessor interfaces ✅ + Plugin descriptor ✅ + Fail strategies ✅ + StreamPostProcessor ✅ + Model group routing ⏳ |
| **P6 Persistence** | ⏳ | SQLite + Repository layer + Admin API CRUD + Playground |
| **P7 Production Deploy** | 🚧 | Docker ✅ + K8s ✅ + Prometheus /metrics ✅ + Graceful shutdown ✅ + CORS ✅ + Rate limiting ⏳ |
| **P8 Extended Protocols** | ⏳ | Embedding passthrough / MCP endpoint / Rerank adapter |

### Key Milestones

- **P0 done** → first curl request works
- **P1 done** → `go run ./cmd/strait/` one-command start
- **P3 done** → requests carry tools, responses support tool_calls
- **P4 done** → one YAML config for multiple providers, auto failover
- **P7 done** → production-ready, GitHub open source

---

## Plugins

Everything in Strait is a plugin — Router, Adapter, Authenticator, Guard, Pre/PostProcessor. Loaded via `plugins.yaml`, decoupled through `api` package interfaces.

### Built-in Plugins

| Plugin | Type | Status | Description |
|--------|------|--------|-------------|
| router-yaml | Router | ✅ | YAML route matching + multi-target + priority/weight strategy |
| adapter-openai | Adapter | ✅ | OpenAI-compatible protocol adapter |
| adapter-ollama | Adapter | ✅ | Ollama local model adapter |
| auth-static-token | Authenticator | ✅ | Static token authentication |
| prompt-injector | PreProcessor | ✅ | System prompt auto-injection |

### v0.3 New Capabilities

- **Plugin Descriptor** — plugins declare type, priority, fail strategy, and config schema via `Descriptor()`; `manifest.json` auto-generated on startup
- **Fail Strategies** — strict (abort) / skip (ignore) / fallback (default), handled by scheduler
- **StreamPostProcessor** — stream-aware post-processing interface, operates directly on StreamChunk channel
- **Request deep-copy isolation** — pipeline entry auto-deep-copies ChatRequest, preprocessor mutations don't affect original payload
- **TraceID full-chain tracing** — unique ID per request across all six stages, correlated via `trace_id` in logs
- **CORS middleware** — configure allowed origins via `STRAIT_CORS_ORIGINS` env var
- **Config merging** — plugin `DefaultConfig` auto-merged with user `config`

### In Progress

| Plugin | Type | Plan | Description |
|--------|------|------|-------------|
| adapter-anthropic | Adapter | P4 | Claude Messages API adapter |
| adapter-gemini | Adapter | P4 | Google Gemini API adapter |
| rate-limiter | Guard | P5 | Token bucket / sliding window rate limiting |
| audit-logger | PostProcessor | P5 | Request-response audit logging |
| cost-tracker | PostProcessor | P5 | Token usage and cost tracking |

---

## Documentation

| Document | Content |
|----------|---------|
| [Built-in Plugins](plugins/built-in.md) | All built-in plugins configuration reference |
| [Plugin Roadmap](plugins/roadmap.md) | Plugin ecosystem evolution |
| [Plugin Dev Guide](developer/plugin-dev.md) | How to write Strait plugins |
| [Comparison](comparison.md) | Strait vs LiteLLM / Kong |
| [Deployment](deployment.md) | Docker, K8s, environment variables |
| [API Design](api-design.md) | Public types and interface definitions |
