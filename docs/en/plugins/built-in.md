# Built-in Plugins

English | [中文](../../zh/plugins/built-in.md)

Strait ships with 5 built-in plugins, ready to use.

## List

| Plugin | Type | Directory | Description |
|--------|------|-----------|-------------|
| router-yaml | Router | `internal/router/` | YAML-based route rule matching |
| adapter-openai | Adapter | `plugins/adapter-openai/` | OpenAI-compatible protocol adapter |
| adapter-ollama | Adapter | `plugins/adapter-ollama/` | Ollama local model adapter |
| auth-static-token | Authenticator | `plugins/auth-static-token/` | Static token authentication |
| prompt-injector | PreProcessor | `plugins/prompt-injector/` | System prompt auto-injection |

---

## router-yaml

Routes requests to the matching provider based on rules in `routes.yaml`. Supports multi-target routing with priority/weight strategy.

```yaml
# plugins.yaml
plugins:
  - id: router-yaml
    type: router
```

### Single Target (legacy format)

```yaml
# routes.yaml
routes:
  - id: deepseek-chat
    match:
      model: deepseek-chat
    target:
      provider: deepseek-main
      model: deepseek-chat
```

### Multi-target Routing

One route can point to multiple providers, selected by strategy:

```yaml
routes:
  - id: deepseek-chat
    match:
      model: deepseek-chat
    strategy: priority
    targets:
      - provider: deepseek-main
        model: deepseek-chat
        priority: 1
        weight: 3
      - provider: deepseek-backup
        model: deepseek-chat
        priority: 2
        weight: 1
```

### Routing Strategies

| Strategy | Description |
|----------|-------------|
| `priority` | Select by priority, lower number = higher priority (default) |
| `weight` | Random selection by weight, higher weight = higher chance |

### Model Mapping

The `model` field in `targets` maps the request model to the upstream model. For example, `model: gpt-4` can map to `deepseek-chat`.

### Features

- **No extra config** — just declare `type: router` to load
- **Hot reload** — modify `routes.yaml` and it takes effect automatically
- **Legacy compatible** — single-target `target` field still works

---

## adapter-openai

Converts requests to OpenAI-compatible format, sends to upstream provider, converts response back.

```yaml
# plugins.yaml
plugins:
  - id: adapter-openai
    type: adapter
```

```yaml
# providers.yaml
providers:
  - id: deepseek-main
    protocol: openai
    base_url: https://api.deepseek.com/v1
    api_key_env: DEEPSEEK_API_KEY
    models:
      - deepseek-chat
```

- Sends POST `{base_url}/chat/completions` upstream
- Supports streaming (SSE) and non-streaming
- Supports Function Calling passthrough (`tools` → `tool_calls`)

---

## adapter-ollama

Adapts Ollama local model chat API.

```yaml
# plugins.yaml
plugins:
  - id: adapter-ollama
    type: adapter
```

```yaml
# providers.yaml
providers:
  - id: ollama-local
    protocol: ollama
    base_url: http://localhost:11434
    api_key_env: DEEPSEEK_API_KEY
    models:
      - qwen2.5:0.5b
```

- Sends POST `{base_url}/api/chat` upstream
- Supports streaming (NDJSON) and non-streaming
- API key config has no effect for Ollama, use a placeholder

---

## auth-static-token

Simple static token authentication. Suitable for development and internal deployments.

```yaml
# plugins.yaml
plugins:
  - id: auth-static-token
    type: authenticator
    priority: 10
    config:
      token: sk-admin-init
      subject: admin
```

Requests must carry `Authorization: Bearer sk-admin-init` header. Returns 401 on auth failure.

- **Development**: configure a fixed token, no external auth service needed
- **Production**: replace with JWT / OAuth2 custom auth plugin

---

## prompt-injector

Automatically injects a system prompt before requests reach the AI. All pipeline requests get the unified system prompt injected, zero client changes needed.

```yaml
# plugins.yaml
plugins:
  - id: prompt-injector
    type: preprocessor
    priority: 50
    config:
      system_prompt: "You are a helpful assistant."
```

- Prepends `{"role": "system", "content": "..."}` to messages
- Skips injection if messages already have a system message
- Hot reload on `system_prompt` change

---

## Protocol Matching

Adapter selection is connected via the `protocol` field:

```
Adapter.Protocol()  →  providers.yaml protocol field
```

Example: provider declares `protocol: openai`, Scheduler finds `adapter-openai` (whose `Protocol()` returns `"openai"`) to execute the request.

## Plugin Config Fields

Each plugin in `plugins.yaml` supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Plugin ID, matches `api.Register()` name |
| `type` | string | Yes | Plugin type: `authenticator` / `guard` / `preprocessor` / `router` / `adapter` / `postprocessor` |
| `priority` | int | No | Execution priority, lower = first. Defaults to `Descriptor()` value, or 100 |
| `fail_mode` | string | No | Fail strategy: `strict` (abort) / `skip` (ignore) / `fallback` (default value). Default `strict` |
| `config` | map | No | Plugin config, merged with `DefaultConfig` before passing to `Init()` |

## manifest.json

Auto-generated on startup, aggregates all loaded plugin descriptors. Used for:

- Pipeline Editor tool loads plugin metadata, auto-generates config forms
- External systems query available plugins and their config schemas

Regenerated on hot reload.
