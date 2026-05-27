# Strait Roadmap

> AI 流经此处 — 插件驱动的 AI 网关

## 总体原则

- **内核优先**：先让核心管线跑通，再做周边
- **插件驱动扩展**：所有业务策略通过插件接口解耦
- **配置驱动**：YAML 驱动，不做 10 张表的企业级数据模型
- **先 Runtime 后 Persistence**：网关本质是运行时，不先建数据库

---

## 内核路线

> Strait 本体交付。内核只做路由、转发、鉴权、管线——其余能力以插件形式接入。

| 阶段 | 状态 | 交付内容 |
|------|------|---------|
| **P0 MVP 骨架** | ✅ | 项目结构 + API 接口定义 + YAML 路由 + OpenAI 适配 + SSE 流式 |
| **P1 插件加载器** | ✅ | Loader + 自动注册 + 配置驱动 + 启动日志 |
| **P2 生产可用性** | ✅ | 鉴权 + 错误模型 + 热重载 + 统一内部模型 + OpenAI 输出格式 |
| **P3 Agent 协议** | ✅ | Function Calling + Tool Use + Tool Calls 响应 |
| **P4 多供应商** | 🚧 | Ollama 适配 ✅ / Anthropic + Gemini 适配 ⏳ / 负载均衡 + 熔断 ⏳ |
| **P5 管道扩展** | ⏳ | PreProcessor / PostProcessor 接口 + 模型分组路由 |
| **P6 持久化** | ⏳ | SQLite + Repository 层 + 管理 API CRUD + Playground |
| **P7 生产部署** | ⏳ | Docker + 优雅关闭 + Prometheus + 限流 |
| **P8 扩展协议** | ⏳ | Embedding 透传 / MCP 端点 / Rerank 适配 |

### 内核关键里程碑

- **P0 完成** → 首个 curl 请求走通
- **P1 完成** → `go run ./cmd/strait/` 一键启动
- **P3 完成** → 请求可携带 tools，响应支持 tool_calls
- **P4 完成** → 一份 YAML 配多个供应商，自动故障转移
- **P7 完成** → 可生产部署，GitHub 开源

---

## 插件

Strait 所有组成部分都是插件——Router、Adapter、Authenticator、Guard、Pre/PostProcessor。统一由 `plugins.yaml` 加载，通过 `api` 包接口解耦。

### 内置插件

| 插件 | 类型 | 状态 | 说明 |
|------|------|------|------|
| router-yaml | Router | ✅ | YAML 配置路由匹配 |
| adapter-openai | Adapter | ✅ | OpenAI 兼容协议适配 |
| adapter-ollama | Adapter | ✅ | Ollama 本地模型适配 |
| auth-static-token | Authenticator | ✅ | 静态 Token 鉴权 |

### 开发中

| 插件 | 类型 | 计划 | 说明 |
|------|------|------|------|
| adapter-anthropic | Adapter | P4 | Claude 协议适配 |
| adapter-gemini | Adapter | P4 | Gemini 协议适配 |
| rate-limiter | Guard | P5 | 令牌桶 / 滑动窗口限流 |
| prompt-injector | PreProcessor | P5 | 系统提示词自动注入 |
| audit-logger | PostProcessor | P5 | 请求-响应审计记录 |
| cost-tracker | PostProcessor | P5 | Token 用量和成本统计 |

---

## 文档

| 文档 | 内容 |
|------|------|
| [内置插件](docs/plugins/built-in.md) | 所有内置插件的配置参考 |
| [插件路线](docs/plugins/roadmap.md) | 插件的完整演进路线 |
| [插件开发指南](docs/developer/plugin-dev.md) | 如何开发 Strait 插件 |