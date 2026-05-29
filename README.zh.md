# Strait

> 不只是网关，是 AI 请求的管线引擎。

[English](README.md) | 中文

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)

Strait 是一个插件驱动的 AI 请求管线框架。它对外暴露 OpenAI 兼容 API，将请求经过六个阶段的插件管线——认证、守卫、预处理、路由、适配器、后处理——每个阶段都通过 Go 插件编程扩展。

## 实际应用场景

- **多供应商路由** — 一份 YAML 配置，按模型名自动路由到 OpenAI 或 Ollama 本地模型
- **权重分流** — 同一模型配置多个供应商，按权重分配请求（如 OpenAI 70% + Ollama 30%）
- **统一入口** — 对外暴露 OpenAI 兼容 API，后端可以是任何供应商，客户端无感知
- **系统提示词注入** — 所有请求自动附加统一的系统提示词，无需客户端修改
- **静态 Token 鉴权** — 内网部署场景，一个固定 Token 即可保护 API 端点

## 为什么选 Strait？

- **不是薄代理** — 完整的六阶段请求管线，包含守卫、预处理、后处理阶段
- **不是纯配置** — 用 Go 写真实逻辑（PII 检测、内容过滤、成本追踪）
- **不会脆弱** — 插件失败策略（严格/跳过/降级）、请求深拷贝隔离、TraceID 全链路追踪
- **多供应商** — YAML 配置路由到 OpenAI、Ollama 或任意后端，支持优先级/权重故障转移

## 架构

```
  Request ──▶ [认证] ──▶ [守卫] ──▶ [预处理] ──▶ [路由] ──▶ [适配器] ──▶ [后处理] ──▶ Response
                                                    │
                                              ┌─────┴─────┐
                                              │   路由器    │
                                              └─────┬─────┘
                                         ┌──────────┼──────────┐
                                      OpenAI    Ollama      ...
```

| 阶段 | 用途 | 示例插件 |
|------|------|---------|
| **认证** | Token 验证 | `auth-static-token` |
| **守卫** | 速率限制、IP 黑名单、PII 检测 | — |
| **预处理** | 提示注入、上下文扩充 | `prompt-injector` |
| **路由** | 模型路由、A/B 测试、故障转移 | `router-yaml` |
| **适配器** | 协议转换、调用后端 | `adapter-openai`, `adapter-ollama` |
| **后处理** | 日志记录、Token 统计、内容过滤 | — |

插件按优先级从小到大执行，失败时由调度器根据插件声明的策略自动处理。

## 快速开始

```bash
go build -o strait ./cmd/strait/
export DEEPSEEK_API_KEY=sk-xxx
./strait
```

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-admin-init" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"你好"}]}'
```

## 编写插件

实现 `Plugin` + 一个阶段接口，声明 `Descriptor()`，通过 `api.Register()` 注册。约 30 行代码。

```go
func (r *RateLimiter) Guard(pctx *api.PipelineContext) error {
    if r.count++; r.count > r.limit {
        return api.NewPluginError(api.ErrCodeRateLimited, "too many requests", true)
    }
    return nil
}
```

详见 [插件开发指南](docs/zh/developer/plugin-dev.md)。

## 实例

- **加权路由** — 通过 YAML 将 `deepseek-chat` 请求按 3:1 分配到 Deepseek 和本地 Ollama
- **系统提示词注入** — 所有请求自动附加统一的 system prompt，客户端零改动
- **静态 Token 鉴权** — 内网部署场景，固定 Token 保护 API 端点

详见 [examples/](examples/)。

## Tools UI

浏览器端 Playground 和 Pipeline Editor：https://yukk1o.github.io/Strait/

## 文档

| 文档 | 内容 |
|------|------|
| [对比](docs/zh/comparison.md) | Strait vs LiteLLM / Kong |
| [部署](docs/zh/deployment.md) | Docker、K8s、环境变量 |
| [内置插件](docs/zh/plugins/built-in.md) | 开箱即用的插件介绍与配置 |
| [插件开发指南](docs/zh/developer/plugin-dev.md) | 如何开发 Strait 插件 |
| [路线图](docs/zh/roadmap.md) | 项目演进路线 |
| [API 设计](docs/zh/api-design.md) | 公共类型和接口定义 |

## License

[Apache License 2.0](LICENSE)
