# Strait

> AI 流经此处 — 插件驱动的 AI 网关

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)

## 功能

| 功能 | 状态 |
|------|------|
| OpenAI 兼容 API 入口 | ✅ |
| YAML 配置路由 | ✅ |
| 流式响应 (SSE) | ✅ |
| 插件系统 (Router / Adapter / Authenticator) | ✅ |
| 插件加载器 | ✅ |
| 鉴权 | ✅ |
| 热重载 | ✅ |
| 统一错误模型 | ✅ |
| 统一内部模型 + OpenAI 输出格式 | ✅ |
| Function Calling 透传 | ✅ |
| Ollama 适配 | ✅ |
| 多供应商 + 负载均衡 | ⏳ 规划中 |
| 管理 API + Playground | ⏳ 规划中 |
| 插件生态 (限流 / 审计 / 成本管控 ...) | ⏳ 规划中 |

详见 [Roadmap](docs/roadmap.md)。

## 快速开始

```bash
export DEEPSEEK_API_KEY=sk-xxx
go run ./cmd/strait/
```

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-admin-init" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"hello"}]}'
```

```bash
# 流式
curl -N http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-admin-init" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"hello"}],"stream":true}'
```

## 配置

编辑 `configs/` 下的 YAML 文件：

- `plugins.yaml` — 插件列表及配置（Router、Adapter、Authenticator 等）
- `providers.yaml` — AI 供应商定义（URL、API Key 环境变量名、模型列表）
- `routes.yaml` — 路由规则（model → provider 映射）

修改 `providers.yaml` 或 `routes.yaml` 后保存，服务自动热加载。

## 文档

| 文档 | 内容 |
|------|------|
| [Roadmap](docs/roadmap.md) | 项目演进路线 |
| [内置插件](docs/plugins/built-in.md) | 开箱即用的内置插件介绍与配置 |
| [插件路线](docs/plugins/roadmap.md) | 插件体系演进路线 |
| [插件开发指南](docs/developer/plugin-dev.md) | 如何开发 Strait 插件 |

## License

Apache License 2.0
