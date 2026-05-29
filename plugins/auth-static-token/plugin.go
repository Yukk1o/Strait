package auth_static_token

import (
	"context"
	"errors"

	"strait/api"
)

type Auth struct {
	token     string
	subjectID string
}

func (a *Auth) ID() string { return "auth-static-token" }
func init() {
	api.Register("auth-static-token", func() api.Plugin { return &Auth{} })
}

func (a *Auth) Descriptor() api.PluginDescriptor {
	return api.PluginDescriptor{
		ID:          "auth-static-token",
		Type:        "authenticator",
		Description: "静态令牌认证插件",
		Version:     "0.3.0",
		Priority:    10,
		FailMode:    api.FailStrict,
		ConfigSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"token": map[string]any{
					"type":        "string",
					"description": "静态认证令牌",
				},
				"subject": map[string]any{
					"type":        "string",
					"description": "认证主体标识",
				},
			},
			"required": []string{"token", "subject"},
		},
		DefaultConfig: map[string]any{
			"token":   "sk-admin-init",
			"subject": "admin",
		},
	}
}

func (a *Auth) Init(cfg map[string]any) error {
	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return errors.New("auth-static-token: token required")
	}
	a.token = token

	subject, ok := cfg["subject"].(string)
	if !ok || subject == "" {
		return errors.New("auth-static-token: subject required")
	}
	a.subjectID = subject
	return nil
}

func (a *Auth) Authenticate(ctx context.Context, token string) (*api.Subject, error) {
	if token != a.token {
		return nil, &api.PluginError{
			Code:      "AUTH_FAILED",
			Message:   "invalid token",
			Retryable: false,
		}
	}
	return &api.Subject{ID: a.subjectID}, nil
}
