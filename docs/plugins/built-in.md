# 内置插件

Strait 提供 4 个内置插件，启动即用。

## 列表

| 插件 | 类型 | 目录 | 说明 |
|------|------|------|------|
| router-yaml | Router | `internal/router/` | 基于 YAML 配置的路由规则匹配 |
| adapter-openai | Adapter | `plugins/adapter-openai/` | OpenAI 兼容协议适配 |
| adapter-ollama | Adapter | `plugins/adapter-ollama/` | Ollama 本地模型适配 |
| auth-static-token | Authenticator | `plugins/auth-static-token/` | 静态 Token 鉴权 |

---

## router-yaml

根据 `routes.yaml` 中的路由规则，将请求匹配到对应的 provider。

```yaml
# plugins.yaml
plugins:
  - id: router-yaml
    type: router
```

```yaml
# routes.yaml
routes:
  - id: deepseek-chat
    match:
      model: deepseek-chat           # 匹配请求中的 model 字段
    target:
      provider: deepseek-main        # 转发到 providers.yaml 中定义的 provider
      model: deepseek-chat           # 实际请求上游的模型名
```

- **无需额外配置** — 直接声明 `type: router` 即可加载
- **支持热重载** — 修改 `routes.yaml` 后自动生效

---

## adapter-openai

将请求转为 OpenAI 兼容格式，发送到上游 provider，再将响应转回内部模型。

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
    protocol: openai                 # 通过 protocol 字段匹配 adapter-openai
    base_url: https://api.deepseek.com/v1
    api_key_env: DEEPSEEK_API_KEY
    models:
      - deepseek-chat
```

- 向上游发送 POST `{base_url}/chat/completions`
- 支持流式 (SSE) 和非流式
- 支持 Function Calling 透传（`tools` → `tool_calls`）

---

## adapter-ollama

适配 Ollama 本地模型的 chat API。

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
    protocol: ollama                 # 通过 protocol 字段匹配 adapter-ollama
    base_url: http://localhost:11434
    api_key_env: DEEPSEEK_API_KEY    # Ollama 本地不需要 key，但字段必填
    models:
      - qwen2.5:0.5b
```

- 向上游发送 POST `{base_url}/api/chat`
- 支持流式 (NDJSON) 和非流式
- API Key 配置项对 Ollama 无实际作用，填写占位值即可

---

## auth-static-token

基于静态 Token 的简单鉴权。适合开发、内网部署场景。

```yaml
# plugins.yaml
plugins:
  - id: auth-static-token
    type: authenticator
    config:
      token: sk-admin-init            # 服务端校验的 token
      subject: admin                  # 鉴权通过后赋予的调用方标识
```

请求需携带 `Authorization: Bearer sk-admin-init` 头。鉴权失败返回 401。

- **开发场景**：配置一个固定 token 即可，无需外部认证服务
- **生产场景**：建议替换为 JWT / OAuth2 等自定义认证插件

## 协议匹配机制

Adapter 的选择是通过 `protocol` 字段串起来的：

```
Adapter.Protocol()  →  providers.yaml 的 protocol 字段
```

例如：provider 声明 `protocol: openai`，Scheduler 会找到 `adapter-openai`（其 `Protocol()` 返回 `"openai"`）来执行请求。