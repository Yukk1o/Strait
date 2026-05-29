# API 设计文档

[English](../en/api-design.md) | 中文

> 所有 API 遵循 RESTful 风格，统一响应格式，支持分页和过滤

## 1. 统一响应格式

```json
// 成功
{
  "code": 0,
  "message": "ok",
  "data": { ... },
  "request_id": "uuid"
}

// 分页成功
{
  "code": 0,
  "data": {
    "items": [...],
    "total": 150,
    "page": 1,
    "page_size": 20
  }
}

// 错误
{
  "code": 40001,
  "message": "Invalid request parameter: model_name is required",
  "request_id": "uuid"
}
```

## 2. 错误码规范

| 错误码 | 含义 | HTTP 状态码 |
|--------|------|-----------|
| 0 | 成功 | 200/201 |
| 40001 | 参数错误 | 400 |
| 40101 | 未认证 (Missing API Key) | 401 |
| 40102 | API Key 无效或已过期 | 401 |
| 40301 | 无权限 | 403 |
| 40401 | 资源不存在 | 404 |
| 40901 | 资源冲突 (名称重复等) | 409 |
| 42901 | 请求限流 | 429 |
| 50001 | 内部错误 | 500 |
| 50201 | 上游服务不可用 | 502 |
| 50301 | 服务暂时不可用 | 503 |

## 3. 管理 API (Admin)

### 3.1 供应商管理

```
GET    /api/v1/admin/providers          # 列表
POST   /api/v1/admin/providers          # 创建
GET    /api/v1/admin/providers/:id      # 详情
PUT    /api/v1/admin/providers/:id      # 更新
DELETE /api/v1/admin/providers/:id      # 软删除
```

**创建供应商:**
```bash
POST /api/v1/admin/providers
Authorization: Bearer sk-admin-xxx

{
  "name": "openai",
  "display_name": "OpenAI",
  "base_url": "https://api.openai.com",
  "provider_type": "openai",
  "api_key": "sk-xxxxxxxx",           // 传入明文，存储时加密
  "models": [
    {
      "model_name": "gpt-4o",
      "type": "chat",
      "max_tokens": 128000,
      "input_price": 2.50,
      "output_price": 10.00
    },
    {
      "model_name": "gpt-4o-mini",
      "type": "chat",
      "max_tokens": 128000,
      "input_price": 0.15,
      "output_price": 0.60
    }
  ]
}
```

**查询参数:** `?page=1&page_size=20&enabled=true&keyword=openai`

### 3.2 路由规则管理

```
GET    /api/v1/admin/routes            # 列表
POST   /api/v1/admin/routes            # 创建
GET    /api/v1/admin/routes/:id        # 详情
PUT    /api/v1/admin/routes/:id        # 更新
DELETE /api/v1/admin/routes/:id        # 删除
```

**创建路由规则:**
```json
{
  "name": "gpt4-route",
  "match_type": "prefix",
  "match_value": "gpt-4",
  "upstream_ids": ["up_001", "up_002"],
  "balancer_strategy": "weighted_round_robin",
  "priority": 10,
  "enabled": true,
  "config": {
    "timeout_ms": 30000,
    "retry_count": 2,
    "rate_limit_per_min": 100
  }
}
```

### 3.3 上游服务管理

```
GET    /api/v1/admin/upstreams         # 列表
POST   /api/v1/admin/upstreams         # 创建
GET    /api/v1/admin/upstreams/:id     # 详情
PUT    /api/v1/admin/upstreams/:id     # 更新
DELETE /api/v1/admin/upstreams/:id     # 删除
PUT    /api/v1/admin/upstreams/:id/toggle  # 启用/禁用
```

### 3.4 模型分组管理

```
GET    /api/v1/admin/groups                   # 分组列表
POST   /api/v1/admin/groups                   # 创建分组
GET    /api/v1/admin/groups/:id               # 分组详情 (含关联模型)
PUT    /api/v1/admin/groups/:id               # 更新分组
DELETE /api/v1/admin/groups/:id               # 删除分组
PUT    /api/v1/admin/groups/:id/models        # 更新组内模型 (全量替换)
POST   /api/v1/admin/groups/:id/models        # 添加模型到组
DELETE /api/v1/admin/groups/:id/models/:mid   # 从组内移除模型
```

**创建模型分组:**
```json
{
  "name": "review",
  "display_name": "审查模型组",
  "description": "用于内容安全审查、合同审查等场景",
  "strategy": "fallback",
  "enabled": true,
  "sort_order": 1,
  "models": [
    {"model_id": "mdl_001", "sort_order": 1, "weight": 100},
    {"model_id": "mdl_002", "sort_order": 2, "weight": 80},
    {"model_id": "mdl_003", "sort_order": 3, "weight": 50}
  ]
}
```

**分组路由策略说明:**

| strategy | 说明 | sort_order 含义 | weight 含义 |
|----------|------|----------------|-------------|
| `fallback` | 按 sort_order 顺序尝试，失败降级 | 优先级 (越小越先试) | 忽略 |
| `least_cost` | 选组内价格最低的模型 | 忽略 | 忽略 |
| `best_quality` | 选组内能力评分最高的模型 | 忽略 | 忽略 |
| `round_robin` | 轮询所有可用模型 | 忽略 | 忽略 |
| `weighted` | 按 weight 比例分发流量 | 忽略 | 流量比例 |

### 3.5 MCP 工具管理

```
GET    /api/v1/admin/mcp/tools                # 工具列表
POST   /api/v1/admin/mcp/tools                # 注册工具
GET    /api/v1/admin/mcp/tools/:id            # 工具详情
PUT    /api/v1/admin/mcp/tools/:id            # 更新工具
DELETE /api/v1/admin/mcp/tools/:id            # 删除工具
POST   /api/v1/admin/mcp/tools/:id/test       # 测试调用
```

**注册 MCP 工具:**
```json
{
  "name": "web_search",
  "display_name": "Web Search",
  "description": "Search the web for current information. Returns search results with titles, snippets, and URLs.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "The search query"
      },
      "num_results": {
        "type": "integer",
        "default": 5,
        "description": "Number of results to return"
      }
    },
    "required": ["query"]
  },
  "handler_type": "http",
  "handler_config": {
    "method": "GET",
    "url": "https://api.search.example.com/v1/search",
    "headers": {
      "Authorization": "Bearer ${env.SEARCH_API_KEY}"
    },
    "param_mapping": {
      "query": "q",
      "num_results": "count"
    }
  },
  "tags": ["search", "web"]
}
```

### 3.6 MCP 服务注册发现 (类 Nacos)

MCP Server 启动时自动注册到网关，定期心跳保活。网关作为注册中心维护所有在线节点。

```
POST   /mcp/registry/register            # MCP Server 注册
POST   /mcp/registry/deregister          # MCP Server 注销
POST   /mcp/registry/heartbeat           # 心跳 (10s 一次)
POST   /mcp/registry/sync-tools          # 全量同步工具列表
GET    /api/v1/admin/mcp/nodes           # 查看所有注册节点 (管理)
GET    /api/v1/admin/mcp/nodes/:id       # 节点详情
DELETE /api/v1/admin/mcp/nodes/:id       # 手动摘除节点
```

**注册请求:**
```bash
POST /mcp/registry/register
Content-Type: application/json

{
  "mcp_name": "web-search-svc",
  "display_name": "Web Search MCP Server",
  "host": "10.0.1.5",
  "port": 9001,
  "transport_type": "http",
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web...",
      "input_schema": { ... }
    }
  ],
  "tools_hash": "sha256:abc123def456",
  "metadata": {
    "version": "1.2.0",
    "region": "cn-east"
  }
}
```

**注册响应:**
```json
{
  "code": 0,
  "data": {
    "node_id": "node_abc123",
    "lease_seconds": 30,
    "sync_tools": false
  }
}
```

**心跳请求 (每 10s):**
```bash
POST /mcp/registry/heartbeat
Content-Type: application/json

{
  "node_id": "node_abc123",
  "tools_hash": "sha256:abc123def456"
}
```

**心跳响应 (tools_hash 已变更，触发同步):**
```json
{
  "code": 0,
  "data": {
    "lease_seconds": 30,
    "sync_tools": true
  }
}
```

**工具同步 (收到 sync_tools=true 后调用):**
```bash
POST /mcp/registry/sync-tools
Content-Type: application/json

{
  "node_id": "node_abc123",
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web for current information...",
      "input_schema": { ... }
    },
    {
      "name": "web_fetch",
      "description": "Fetch and parse a web page...",
      "input_schema": { ... }
    }
  ],
  "tools_hash": "sha256:newhash789"
}
```

**查看注册节点:**
```bash
GET /api/v1/admin/mcp/nodes?mcp_name=web-search-svc&status=online
Authorization: Bearer sk-admin-xxx

# 响应
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "node_abc123",
        "mcp_name": "web-search-svc",
        "host": "10.0.1.5",
        "port": 9001,
        "status": "online",
        "tool_count": 2,
        "last_heartbeat_at": "2026-05-24T10:05:30Z",
        "registered_at": "2026-05-24T09:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### 3.7 工作流管理

```
GET    /api/v1/admin/workflows              # 列表
POST   /api/v1/admin/workflows              # 创建
GET    /api/v1/admin/workflows/:id          # 详情
PUT    /api/v1/admin/workflows/:id          # 更新
DELETE /api/v1/admin/workflows/:id          # 删除
POST   /api/v1/admin/workflows/:id/publish  # 发布
POST   /api/v1/admin/workflows/:id/execute  # 测试执行
GET    /api/v1/admin/workflows/:id/executions  # 执行历史
```

**创建工作流:**
```json
{
  "name": "智能客服路由",
  "description": "先判断意图，投诉类升级到高级模型处理",
  "definition": {
    "nodes": [
      {
        "id": "node_1",
        "type": "ai_call",
        "config": {
          "provider_name": "openai",
          "model": "gpt-4o-mini",
          "system_prompt": "你是一个客服助手，分析用户意图...",
          "temperature": 0
        }
      },
      {
        "id": "node_2",
        "type": "condition",
        "config": {
          "expression": ".intent == 'complaint'",
          "true_node": "node_3",
          "false_node": "node_4"
        }
      },
      {
        "id": "node_3",
        "type": "ai_call",
        "config": {
          "provider_name": "openai",
          "model": "gpt-4o",
          "system_prompt": "你是高级客服专家...",
          "temperature": 0.3
        }
      },
      {
        "id": "node_4",
        "type": "response",
        "config": {}
      }
    ],
    "edges": [
      {"from": "node_1", "to": "node_2"},
      {"from": "node_2", "to": "node_3", "label": "true"},
      {"from": "node_2", "to": "node_4", "label": "false"},
      {"from": "node_3", "to": "node_4"}
    ]
  }
}
```

### 3.8 API Key 管理

```
GET    /api/v1/admin/apikeys              # 密钥列表
POST   /api/v1/admin/apikeys              # 生成密钥 (返回完整 key，仅此一次!)
DELETE /api/v1/admin/apikeys/:id          # 吊销密钥
```

**生成密钥响应 (仅创建时返回完整 key):**
```json
{
  "code": 0,
  "data": {
    "id": "key_001",
    "name": "生产环境密钥",
    "api_key": "sk-aigw-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",  // ← 仅此一次
    "key_prefix": "sk-aigw-x",
    "role": "admin",
    "rate_limit": 100,
    "created_at": "2026-05-24T10:00:00Z"
  },
  "message": "API Key created. Store it safely — the full key will not be shown again."
}
```

## 4. AI 代理 API (对外服务)

### 4.1 Chat Completions (OpenAI 兼容)

```bash
POST /v1/chat/completions
Authorization: Bearer sk-aigw-xxx
Content-Type: application/json

{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "你是一个有用的助手"},
    {"role": "user", "content": "你好"}
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "stream": false
}
```

**响应 (OpenAI 兼容格式):**
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1716940800,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！有什么可以帮助你的吗？"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 12,
    "total_tokens": 32
  }
}
```

**流式响应 (stream=true):**
```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"你"}}]}
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"好"}}]}
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"！"}}]}
data: [DONE]
```

### 4.2 模型列表

```bash
GET /v1/models
Authorization: Bearer sk-aigw-xxx

# 响应: OpenAI 兼容的模型列表
{
  "object": "list",
  "data": [
    {"id": "gpt-4o", "object": "model", "owned_by": "openai"},
    {"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
    {"id": "claude-sonnet-4-6", "object": "model", "owned_by": "anthropic"},
    {"id": "deepseek-v3", "object": "model", "owned_by": "deepseek"}
  ]
}
```

### 4.3 分组路由调用

支持按模型分组调用，无需指定具体模型名。

```bash
POST /v1/chat/completions
Authorization: Bearer sk-aigw-xxx
Content-Type: application/json

{
  "group": "审查模型组",
  "messages": [
    {"role": "user", "content": "请审查这份合同是否有法律风险..."}
  ],
  "temperature": 0.1,
  "stream": false
}
```

**分组路由响应 (与标准 Chat Completion 一致):**
- 网关根据分组的 strategy 自动选择组内模型
- 响应中 `model` 字段返回实际使用的模型名
- 如果 `group` 和 `model` 同时传，`group` 优先

## 5. Playground 对话服务

为方便开发测试，提供独立的对话 API (不依赖编排引擎)。

```
POST   /api/v1/playground/chat              # 发送消息 (支持 group/model)
POST   /api/v1/playground/chat/stream       # 流式发送 (SSE)
GET    /api/v1/playground/conversations     # 会话列表
POST   /api/v1/playground/conversations     # 新建会话
GET    /api/v1/playground/conversations/:id  # 会话详情 (含消息历史)
DELETE /api/v1/playground/conversations/:id  # 删除会话
GET    /api/v1/playground/groups            # 列出可用分组
GET    /api/v1/playground/models            # 列出可用模型
```

**发送消息 (分组模式):**
```bash
POST /api/v1/playground/chat
Authorization: Bearer sk-aigw-xxx

{
  "conversation_id": null,
  "group": "审查模型组",
  "messages": [
    {"role": "user", "content": "你好，请帮我审查这份合同..."}
  ],
  "temperature": 0.7,
  "max_tokens": 2048
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "conversation_id": "conv_abc123",
    "model": "gpt-4o",
    "message": {
      "role": "assistant",
      "content": "好的，我来帮你审查这份合同...",
      "token_count": 150
    },
    "usage": {
      "prompt_tokens": 50,
      "completion_tokens": 150,
      "total_tokens": 200
    }
  }
}
```

**流式发送 (SSE):**
```bash
POST /api/v1/playground/chat/stream
Authorization: Bearer sk-aigw-xxx

{
  "conversation_id": "conv_abc123",
  "group": "审查模型组",
  "messages": [
    {"role": "user", "content": "继续"}
  ],
  "stream": true
}

# SSE 响应
event: message
data: {"delta":{"content":"好"},"conversation_id":"conv_abc123"}

event: message
data: {"delta":{"content":"的"},"conversation_id":"conv_abc123"}

event: done
data: {"conversation_id":"conv_abc123","model":"gpt-4o","usage":{"prompt_tokens":80,"completion_tokens":200}}
```

**会话列表:**
```bash
GET /api/v1/playground/conversations?page=1&page_size=20

{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "conv_abc123",
        "title": "合同审查测试",
        "model_name": "",
        "group_name": "审查模型组",
        "message_count": 4,
        "total_tokens": 450,
        "created_at": "2026-05-24T10:00:00Z"
      }
    ],
    "total": 15
  }
}
```

## 6. MCP 协议端点

### 6.1 MCP Server 端点

根据 MCP 协议规范，网关作为 MCP Server 对外暴露：

```
POST /mcp                 # JSON-RPC 2.0 请求 (无状态)
GET  /mcp/sse             # SSE 流式传输 (有状态会话)
```

**初始化握手:**
```bash
POST /mcp
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "clientInfo": {
      "name": "my-ai-app",
      "version": "1.0.0"
    }
  }
}
```

**列出工具:**
```bash
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

**调用工具:**
```bash
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "web_search",
    "arguments": {
      "query": "Go language tutorial",
      "num_results": 5
    }
  }
}
```

## 7. 工作流执行 API

### 6.1 触发工作流

```bash
POST /api/v1/workflows/:id/execute
# 或使用名称
POST /api/v1/workflows/execute
Content-Type: application/json
Authorization: Bearer sk-aigw-xxx

{
  "workflow_name": "智能客服路由",
  "input": {
    "user_message": "我要投诉，产品有严重的质量问题！",
    "user_id": "user_12345"
  },
  "stream": true
}
```

### 6.2 流式工作流响应

```
event: node_start
data: {"node_id":"node_1","type":"ai_call","status":"running"}

event: node_chunk
data: {"node_id":"node_1","content":"正在分析用户意图..."}

event: node_complete
data: {"node_id":"node_1","output":{"intent":"complaint","sentiment":"angry"}}

event: node_start
data: {"node_id":"node_3","type":"ai_call","status":"running"}

event: node_chunk
data: {"node_id":"node_3","content":"非常抱歉给您带来不便..."}

event: workflow_complete
data: {"status":"success","output":"非常抱歉给您带来不便...","duration_ms":2500}
```

## 8. 监控和运维 API

```
GET /health                               # 健康检查
GET /ready                                # 就绪检查
GET /metrics                              # Prometheus 指标
GET /api/v1/admin/stats                   # 概览统计
GET /api/v1/admin/stats/providers         # 供应商维度统计
GET /api/v1/admin/stats/model-usage       # 模型用量统计
GET /api/v1/admin/audit-logs              # 审计日志 (带过滤)
```

**概览统计响应:**
```json
{
  "total_requests": 125430,
  "total_tokens": 50234000,
  "total_cost": 125.67,
  "active_workflows": 5,
  "active_mcp_tools": 12,
  "today": {
    "requests": 3420,
    "tokens": 1520000,
    "cost": 3.85,
    "avg_latency_ms": 850
  }
}
```

## 9. API 鉴权流程

```
客户端                          AI Gateway
  │                                │
  │  POST /v1/chat/completions     │
  │  Authorization: Bearer sk-xxx  │
  │ ──────────────────────────────►│
  │                                │ 1. 提取 API Key
  │                                │ 2. SHA-256(api_key) → key_hash
  │                                │ 3. 查 api_keys 表
  │                                │ 4. 校验 enabled + expires_at
  │                                │ 5. 更新 last_used_at
  │                                │ 6. 注入 context
  │                                │ 7. 继续处理...
  │◄──────────────────────────────│
  │  200 OK / 401 Unauthorized     │
```

## 10. 分页和过滤通用参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码 |
| `page_size` | int | 20 (max 100) | 每页条数 |
| `sort_by` | string | `created_at` | 排序字段 |
| `sort_order` | string | `desc` | asc / desc |
| `keyword` | string | — | 模糊搜索 |
| `enabled` | bool | — | 启用状态筛选 |
| `from` | datetime | — | 时间范围起始 |
| `to` | datetime | — | 时间范围结束 |
