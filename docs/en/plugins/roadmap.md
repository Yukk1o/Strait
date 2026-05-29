# Plugin Roadmap

English | [中文](../../zh/plugins/roadmap.md)

> Everything in Strait is a plugins. Registered via `api.Register()`, loaded via `plugins.yaml`.

## Type Overview

| Type | Interface | Count | Description |
|------|-----------|-------|-------------|
| Router | `api.Router` | 1 | Single routing decision, swappable rules |
| Adapter | `api.ChatAdapter` | Multiple | Found by protocol, one per AI provider |
| Authenticator | `api.Authenticator` | Multiple | Chain auth, any pass = allow |
| Guard | `api.Guard` | Multiple | Request interception (rate limit, budget, compliance) |
| PreProcessor | `api.PreProcessor` | Multiple | Transform before AI (redaction, injection) |
| PostProcessor | `api.PostProcessor` | Multiple | Transform before client (audit, cost) |

## Implemented

| Plugin | Type | Description |
|--------|------|-------------|
| router-yaml | Router | YAML route matching |
| adapter-openai | Adapter | OpenAI-compatible protocol adapter |
| adapter-ollama | Adapter | Ollama local model adapter |
| auth-static-token | Authenticator | Static token authentication |
| prompt-injector | PreProcessor | System prompt auto-injection |

## Planned

| Plugin | Type | Prerequisite | Description |
|--------|------|-------------|-------------|
| adapter-anthropic | Adapter | P4 multi-provider | Claude Messages API adapter |
| adapter-gemini | Adapter | P4 multi-provider | Google Gemini API adapter |
| rate-limiter | Guard | Guard interface | Token bucket / sliding window |
| ip-blocker | Guard | Guard interface | IP whitelist/blacklist |
| budget-check | Guard | Guard interface | Per-team/model budget circuit breaker |
| pii-mask | PreProcessor | PreProcessor interface | Sensitive data redaction |
| audit-logger | PostProcessor | PostProcessor interface | Full request-response audit |
| cost-tracker | PostProcessor | PostProcessor interface | Token usage and cost tracking |

## Dependencies

```
P3 Agent Protocol ──→ Adapter interface unchanged
P4 Multi-provider  ──→ adapter-anthropic / adapter-gemini
P5 Pipeline Extension ──→ Guard / PreProcessor / PostProcessor interfaces ✅
                        ──→ StreamPostProcessor interface ✅
                        ──→ Plugin descriptor + fail strategies ✅
```
