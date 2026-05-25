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
