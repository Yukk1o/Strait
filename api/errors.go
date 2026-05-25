package api

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
