# 插件路线

> Strait 所有组成部分都是插件。统一走 `api.Register()` 注册、`plugins.yaml` 加载。

## 类型一览

| 类型 | 接口 | 数量 | 说明 |
|------|------|------|------|
| Router | `api.Router` | 1 | 只有一个路由决策，路由规则可替换 |
| Adapter | `api.ChatAdapter` | 多个 | 按 protocol 查找，每个 AI 供应商一个 |
| Authenticator | `api.Authenticator` | 多个 | 链式鉴权，任一通过即放行 |
| Guard | `api.Guard` | 多个 | 请求拦截（限流、预算、合规检查） |
| PreProcessor | `api.PreProcessor` | 多个 | 请求到达 AI 前转换（脱敏、注入） |
| PostProcessor | `api.PostProcessor` | 多个 | 响应返回客户端前转换（审计、记成本） |

## 已实现

| 插件 | 类型 | 说明 |
|------|------|------|
| router-yaml | Router | YAML 配置路由匹配 |
| adapter-openai | Adapter | OpenAI 兼容协议适配 |
| adapter-ollama | Adapter | Ollama 本地模型适配 |
| auth-static-token | Authenticator | 静态 Token 鉴权 |

## 规划中

| 插件 | 类型 | 前置条件 | 说明 |
|------|------|---------|------|
| adapter-anthropic | Adapter | P4 多供应商 | Claude Messages API 适配 |
| adapter-gemini | Adapter | P4 多供应商 | Google Gemini API 适配 |
| rate-limiter | Guard | Guard 接口 | 令牌桶 / 滑动窗口限流 |
| ip-blocker | Guard | Guard 接口 | IP 白名单/黑名单 |
| budget-check | Guard | Guard 接口 | 按 team/model 的预算熔断 |
| prompt-injector | PreProcessor | PreProcessor 接口 | 系统提示词自动注入 |
| pii-mask | PreProcessor | PreProcessor 接口 | 敏感信息脱敏 |
| audit-logger | PostProcessor | PostProcessor 接口 | 全量请求-响应审计 |
| cost-tracker | PostProcessor | PostProcessor 接口 | Token 用量和成本统计 |

## 依赖关系

```
P3 Agent 协议 ──→ Adapter 接口保持不变
P4 多供应商  ──→ adapter-anthropic / adapter-gemini
P5 管道扩展  ──→ PreProcessor + PostProcessor 接口
内核重构 TODO ──→ Guard 接口
```
