// Package app 提供 HTTP 服务和请求上下文管理
package app

import (
	"context"

	"strait/api"
)

type requestContextKey struct{}

var ctxKey = requestContextKey{}

// RequestContext 单次请求的全生命周期状态
type RequestContext struct {
	RequestID string       // 请求唯一标识
	Subject   *api.Subject // 认证后的调用方信息
	Token     string       // 认证令牌
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, ctxKey, rc)
}

func FromRequestContext(ctx context.Context) *RequestContext {
	rc, _ := ctx.Value(ctxKey).(*RequestContext)
	return rc
}
