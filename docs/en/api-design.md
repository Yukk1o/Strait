# API Design Document

English | [中文](../zh/api-design.md)

> All APIs follow RESTful conventions with unified response format, pagination, and filtering.

## 1. Unified Response Format

```json
// Success
{
  "code": 0,
  "message": "ok",
  "data": { ... },
  "request_id": "uuid"
}

// Paginated success
{
  "code": 0,
  "data": {
    "items": [...],
    "total": 150,
    "page": 1,
    "page_size": 20
  }
}

// Error
{
  "code": 40001,
  "message": "Invalid request parameter: model_name is required",
  "request_id": "uuid"
}
```

## 2. Error Codes

| Code | Meaning | HTTP Status |
|------|---------|-------------|
| 0 | Success | 200/201 |
| 40001 | Bad parameter | 400 |
| 40101 | Unauthenticated (Missing API Key) | 401 |
| 40102 | API Key invalid or expired | 401 |
| 40301 | Forbidden | 403 |
| 40401 | Resource not found | 404 |
| 40901 | Conflict (duplicate name etc.) | 409 |
| 42901 | Rate limited | 429 |
| 50001 | Internal error | 500 |
| 50201 | Upstream unavailable | 502 |
| 50301 | Service temporarily unavailable | 503 |

## 3. Admin API

### 3.1 Provider Management

```
GET    /api/v1/admin/providers          # List
POST   /api/v1/admin/providers          # Create
GET    /api/v1/admin/providers/:id      # Detail
PUT    /api/v1/admin/providers/:id      # Update
DELETE /api/v1/admin/providers/:id      # Soft delete
```

### 3.2 Route Rule Management

```
GET    /api/v1/admin/routes            # List
POST   /api/v1/admin/routes            # Create
GET    /api/v1/admin/routes/:id        # Detail
PUT    /api/v1/admin/routes/:id        # Update
DELETE /api/v1/admin/routes/:id        # Delete
```

### 3.3 Upstream Service Management

```
GET    /api/v1/admin/upstreams         # List
POST   /api/v1/admin/upstreams         # Create
GET    /api/v1/admin/upstreams/:id     # Detail
PUT    /api/v1/admin/upstreams/:id     # Update
DELETE /api/v1/admin/upstreams/:id     # Delete
PUT    /api/v1/admin/upstreams/:id/toggle  # Enable/disable
```

### 3.4 Model Group Management

```
GET    /api/v1/admin/groups                   # List groups
POST   /api/v1/admin/groups                   # Create group
GET    /api/v1/admin/groups/:id               # Group detail (with models)
PUT    /api/v1/admin/groups/:id               # Update group
DELETE /api/v1/admin/groups/:id               # Delete group
PUT    /api/v1/admin/groups/:id/models        # Update group models (full replace)
POST   /api/v1/admin/groups/:id/models        # Add model to group
DELETE /api/v1/admin/groups/:id/models/:mid   # Remove model from group
```

**Group routing strategies:**

| Strategy | Description | sort_order | weight |
|----------|-------------|-----------|--------|
| `fallback` | Try by sort_order, fail over | Priority (lower first) | Ignored |
| `least_cost` | Pick cheapest model in group | Ignored | Ignored |
| `best_quality` | Pick highest-rated model | Ignored | Ignored |
| `round_robin` | Rotate through all available | Ignored | Ignored |
| `weighted` | Distribute by weight ratio | Ignored | Traffic ratio |

### 3.5 MCP Tool Management

```
GET    /api/v1/admin/mcp/tools                # Tool list
POST   /api/v1/admin/mcp/tools                # Register tool
GET    /api/v1/admin/mcp/tools/:id            # Tool detail
PUT    /api/v1/admin/mcp/tools/:id            # Update tool
DELETE /api/v1/admin/mcp/tools/:id            # Delete tool
POST   /api/v1/admin/mcp/tools/:id/test       # Test call
```

### 3.6 MCP Service Registry (Nacos-like)

MCP Servers auto-register on startup with periodic heartbeat. Gateway acts as registry.

```
POST   /mcp/registry/register            # Register
POST   /mcp/registry/deregister          # Deregister
POST   /mcp/registry/heartbeat           # Heartbeat (every 10s)
POST   /mcp/registry/sync-tools          # Full tool sync
GET    /api/v1/admin/mcp/nodes           # List registered nodes
GET    /api/v1/admin/mcp/nodes/:id       # Node detail
DELETE /api/v1/admin/mcp/nodes/:id       # Manually remove node
```

### 3.7 Workflow Management

```
GET    /api/v1/admin/workflows              # List
POST   /api/v1/admin/workflows              # Create
GET    /api/v1/admin/workflows/:id          # Detail
PUT    /api/v1/admin/workflows/:id          # Update
DELETE /api/v1/admin/workflows/:id          # Delete
POST   /api/v1/admin/workflows/:id/publish  # Publish
POST   /api/v1/admin/workflows/:id/execute  # Test execute
GET    /api/v1/admin/workflows/:id/executions  # Execution history
```

### 3.8 API Key Management

```
GET    /api/v1/admin/apikeys              # List keys
POST   /api/v1/admin/apikeys              # Generate key (full key shown only once!)
DELETE /api/v1/admin/apikeys/:id          # Revoke key
```

## 4. AI Proxy API (Public)

### 4.1 Chat Completions (OpenAI Compatible)

```bash
POST /v1/chat/completions
Authorization: Bearer sk-aigw-xxx
Content-Type: application/json

{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant"},
    {"role": "user", "content": "Hello"}
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "stream": false
}
```

**Streaming response (stream=true):**
```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"}}]}
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"!"}}]}
data: [DONE]
```

### 4.2 Model List

```bash
GET /v1/models
Authorization: Bearer sk-aigw-xxx
```

### 4.3 Group Routing

Call by model group without specifying exact model name.

```bash
POST /v1/chat/completions
Authorization: Bearer sk-aigw-xxx

{
  "group": "review-models",
  "messages": [{"role": "user", "content": "Review this contract..."}]
}
```

If both `group` and `model` are provided, `group` takes precedence.

## 5. Playground API

Independent chat API for development/testing.

```
POST   /api/v1/playground/chat              # Send message
POST   /api/v1/playground/chat/stream       # Streaming (SSE)
GET    /api/v1/playground/conversations     # List conversations
POST   /api/v1/playground/conversations     # New conversation
GET    /api/v1/playground/conversations/:id  # Conversation detail
DELETE /api/v1/playground/conversations/:id  # Delete conversation
GET    /api/v1/playground/groups            # List available groups
GET    /api/v1/playground/models            # List available models
```

## 6. MCP Protocol Endpoints

Gateway exposes MCP Server endpoints:

```
POST /mcp                 # JSON-RPC 2.0 (stateless)
GET  /mcp/sse             # SSE streaming (stateful session)
```

## 7. Workflow Execution API

```bash
POST /api/v1/workflows/:id/execute
Authorization: Bearer sk-aigw-xxx

{
  "workflow_name": "smart-cs-router",
  "input": { "user_message": "I want to file a complaint!" },
  "stream": true
}
```

## 8. Monitoring & Ops API

```
GET /health                               # Health check
GET /ready                                # Readiness check
GET /metrics                              # Prometheus metrics
GET /api/v1/admin/stats                   # Overview stats
GET /api/v1/admin/stats/providers         # Provider stats
GET /api/v1/admin/stats/model-usage       # Model usage stats
GET /api/v1/admin/audit-logs              # Audit logs (with filters)
```

## 9. Auth Flow

```
Client                          AI Gateway
  │                                │
  │  POST /v1/chat/completions     │
  │  Authorization: Bearer sk-xxx  │
  │ ──────────────────────────────►│
  │                                │ 1. Extract API Key
  │                                │ 2. SHA-256(api_key) → key_hash
  │                                │ 3. Lookup api_keys table
  │                                │ 4. Check enabled + expires_at
  │                                │ 5. Update last_used_at
  │                                │ 6. Inject context
  │                                │ 7. Continue...
  │◄──────────────────────────────│
  │  200 OK / 401 Unauthorized     │
```

## 10. Pagination & Filtering

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `page_size` | int | 20 (max 100) | Items per page |
| `sort_by` | string | `created_at` | Sort field |
| `sort_order` | string | `desc` | asc / desc |
| `keyword` | string | — | Fuzzy search |
| `enabled` | bool | — | Enabled status filter |
| `from` | datetime | — | Time range start |
| `to` | datetime | — | Time range end |
