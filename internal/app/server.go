// Package app 提供 HTTP 服务和请求上下文管理
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"strait/internal/metrics"

	"strait/api"
	"strait/internal/plugin"
)

// Server HTTP 服务，持有调度器引用
// （辅助理解）相当于 Java 的 @RestController + @Autowired
type Server struct {
	scheduler *plugin.Scheduler
	metrics   *metrics.Metrics
}

// NewServer 创建 HTTP 服务
func NewServer(s *plugin.Scheduler, m *metrics.Metrics) *Server {
	return &Server{scheduler: s, metrics: m}
}

// Handler 注册路由，返回 http.Handler（含 CORS 中间件）
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /ready", s.readyHandler)
	mux.HandleFunc("POST /v1/chat/completions", s.chatHandler)
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		if _, err := fmt.Fprint(w, s.metrics.Handler()); err != nil {
			slog.Error("write metrics failed", "error", err)
		}
	})
	return corsMiddleware(mux)
}

// corsMiddleware 包装 http.Handler，添加 CORS 头并处理 OPTIONS 预检。
// 通过 STRAIT_CORS_ORIGINS 环境变量配置允许的来源（逗号分隔）。
// 未设置时允许所有来源（仅限开发环境）。
func corsMiddleware(next http.Handler) http.Handler {
	raw := os.Getenv("STRAIT_CORS_ORIGINS")
	var allowed []string
	if raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(o); s != "" {
				allowed = append(allowed, s)
			}
		}
	} else {
		slog.Warn("CORS: STRAIT_CORS_ORIGINS not set, allowing all origins (dev only)")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if len(allowed) == 0 {
			// 开发模式：允许所有
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			for _, o := range allowed {
				if o == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					break
				}
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ready"})
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := generateReqID()
	s.metrics.IncRequests("POST", "/v1/chat/completions")

	var req api.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	slog.Info("chat request", "model", req.Model)

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	ctx := api.WithAuthToken(r.Context(), token)

	if req.Stream {
		// 1. 设置 SSE 头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// 2. 调用调度器
		ch, err := s.scheduler.ExecuteChatStream(ctx, &req)
		if err != nil {
			writeError(w, err)
			return
		}

		// 3. 遍历 channel, 逐块写 SSE data 并 flush
		first := true
		for chunk := range ch {
			if chunk.Err != nil {
				errData, _ := json.Marshal(map[string]any{
					"error": map[string]string{
						"message": chunk.Err.Error(),
						"type":    "server_error",
					},
				})
				if _, err := fmt.Fprintf(w, "data: %s\n\n", errData); err != nil {
					slog.Error("write chunk failed", "error", err)
				}
				flusher.Flush()
				return
			}

			slog.Debug("stream chunk", "chunk", chunk)
			data, _ := json.Marshal(toOpenAIStreamChunk(reqID, start.Unix(), first, chunk))
			first = false
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				slog.Error("write chunk failed", "error", err)
			}
			flusher.Flush()
		}

		// 4. 最后写 SSE 完成并 flush
		slog.Info("stream done", "model", req.Model, "duration_ms", time.Since(start).Milliseconds())
		if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
			slog.Error("write chunk failed", "error", err)
		}
		flusher.Flush()
		return
	}

	resp, err := s.scheduler.ExecuteChat(ctx, &req)
	if err != nil {
		writeError(w, err)
		return
	}

	slog.Info(
		"chat response",
		"model", resp.Model,
		"finish_reason", resp.Choices[0].FinishReason,
		"total_tokens", resp.Usage.TotalTokens,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	writeJSON(w, toOpenAIResp(reqID, start.Unix(), resp))
}

func writeError(w http.ResponseWriter, err error) {
	slog.Error("request failed", "error", err)
	var pe *api.PluginError
	if errors.As(err, &pe) {
		http.Error(w, pe.Message, http.StatusInternalServerError)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON failed", "error", err)
	}
}
