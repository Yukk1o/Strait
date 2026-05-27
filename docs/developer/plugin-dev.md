# Strait 插件开发文档

> 给插件开发者看的。你只需要 `strait/api` 一个依赖。

## 插件类型

| 类型 | 接口 | 职责 |
|------|------|------|
| Router | `api.Router` | 根据请求选择后端 |
| ChatAdapter | `api.ChatAdapter` | 与后端通信，转换协议 |
| Authenticator | `api.Authenticator` | 鉴权 |
| Guard | `api.Guard` | 请求拦截（限流、预算）— 接口已定义，参考实现待完成 |
| PreProcessor | `api.PreProcessor` | 请求预处理 — 接口已定义，参考实现待完成 |
| PostProcessor | `api.PostProcessor` | 响应后处理 — 接口已定义，参考实现待完成 |

## 基础接口

所有插件必须实现 `api.Plugin`：

```go
type Plugin interface {
    ID() string                    // 唯一标识
    Init(config map[string]any) error  // 由管理器调用
}
```

## 注册规范

所有插件通过 `api.Register()` 注册，`plugins.yaml` 加载：

```go
func init() {
    api.Register("my-plugin-id", func() api.Plugin {
        return &MyPlugin{}
    })
}
```

Adapter 额外实现 `Protocol()` 返回协议标识（如 `"openai"`），Scheduler 按 `providers.yaml` 中的 `protocol` 字段匹配。

Router 通过 `api.Router` 接口注册，启动时由 Loader 自动注入 Manager。

## 实现示例

以 ChatAdapter 为例：

```go
package my_adapter

import (
    "context"
    "strait/api"
)

type Adapter struct {
    client *http.Client
}

func (a *Adapter) ID() string          { return "adapters-my-protocol" }
func (a *Adapter) Protocol() string    { return "my-protocol" }  // 协议标识

func (a *Adapter) Init(config map[string]any) error {
    a.client = &http.Client{}
    return nil
}

func (a *Adapter) SendChat(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (*api.ChatResponse, error) {
    // 实现同步请求
}

func (a *Adapter) SendChatStream(ctx context.Context, payload *api.ChatRequest, route *api.RouteDecision) (<-chan *api.StreamChunk, error) {
    // 实现流式请求
}
```

## 生命周期

```
编译时：init() 调用 api.Register() 注册插件工厂
启动时：Loader 读取 plugins.yaml → 调用工厂创建实例 → Init()
运行时：Scheduler 按管线阶段调用
```

## 约束

- 插件不引用 `internal/` 下的任何包，只依赖 `strait/api`
- 插件不能调用其他插件
- 插件不直接访问 Manager 或 Scheduler
- Adapter 不假设请求/响应结构 —— 使用 `api.ChatRequest` / `api.ChatResponse`
