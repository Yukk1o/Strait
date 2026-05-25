// Package app 提供 HTTP 服务和请求上下文管理
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"strait/api"
	"strait/internal/plugin"
)

// Server HTTP 服务，持有调度器引用
// （辅助理解）相当于 Java 的 @RestController + @Autowired
type Server struct {
	scheduler *plugin.Scheduler
}

// NewServer 创建 HTTP 服务
func NewServer(s *plugin.Scheduler) *Server {
	return &Server{scheduler: s}
}

// Handler 注册路由，返回 http.Handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /ready", s.readyHandler)
	mux.HandleFunc("POST /v1/chat/completions", s.chatHandler)
	return mux
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

		// 3. 遍历 channel, 逐块写 SEE data 并 flush
		first := true
		for chunk := range ch {
			slog.Debug("stream chunk", "chunk", chunk)
			data, _ := json.Marshal(toOpenAIStreamChunk(reqID, start.Unix(), first, chunk))
			first = false
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				slog.Error("write chunk failed", "error", err)
			}
			flusher.Flush()
		}

		// 4. 最后写 SEE 完成并 flush
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
