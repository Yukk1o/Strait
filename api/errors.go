package api

type ErrCode string

const (
	ErrCodeAuthFailed      ErrCode = "AUTH_FAILED"
	ErrCodeNoRoute         ErrCode = "NO_ROUTE"
	ErrCodeAdapterNotFound ErrCode = "ADAPTER_NOT_FOUND"
	ErrCodeRateLimited     ErrCode = "RATE_LIMITED"
	ErrCodeUpstreamError   ErrCode = "UPSTREAM_ERROR"
	ErrCodeInvalidRequest  ErrCode = "INVALID_REQUEST"
	ErrCodeGuardRejected   ErrCode = "GUARD_REJECTED"
	ErrCodeCanceled        ErrCode = "CANCELED"
)

type PluginError struct {
	Code      string // "AUTH_FAILED" | "RATE_LIMITED" | "UPSTREAM_ERROR" | "INVALID_REQUEST"
	Message   string // 可读的错误信息
	Retryable bool   // 是否可重试
	Err       error  // 原始错误
}

func (e *PluginError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + " (" + e.Err.Error() + ")"
	}
	return e.Code + ": " + e.Message
}

func (e *PluginError) Unwrap() error {
	return e.Err
}

func NewPluginError(code ErrCode, msg string, retryable bool) *PluginError {
	return &PluginError{Code: string(code), Message: msg, Retryable: retryable}
}
