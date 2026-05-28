// main Strait — AI 代理网关入口。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strait/internal/app"
	"strait/internal/config"
	"strait/internal/hotreload"
	"strait/internal/metrics"
	"strait/internal/plugin"
	"strait/internal/router"
	_ "strait/plugins/adapter-ollama"
	_ "strait/plugins/adapter-openai"
	_ "strait/plugins/auth-static-token"
	_ "strait/plugins/prompt-injector"
)

const banner = `
███████╗████████╗██████╗  █████╗ ██╗████████╗
██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║╚══██╔══╝
███████╗   ██║   ██████╔╝███████║██║   ██║
╚════██║   ██║   ██╔══██╗██╔══██║██║   ██║
███████║   ██║   ██║  ██║██║  ██║██║   ██║
╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝   ╚═╝  v0.2
`

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	loader := plugin.NewLoader(config.PluginsPath)
	m, err := loader.Build()
	if err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}

	if os.Getenv("STRAIT_BANNER") != "false" {
		fmt.Print(banner)
		fmt.Println(" Plugins:")
		fmt.Print(m.Summary())
		fmt.Println()
	}

	scheduler := plugin.NewScheduler(m)
	met := metrics.NewMetrics()

	reload := func(filename string) error {
		if filename == "plugins.yaml" {
			newM, err := loader.Build()
			if err != nil {
				return err
			}
			scheduler.ReloadManager(newM)
			m = newM
			return nil
		}
		if r, ok := m.Router().(*router.Router); ok {
			return r.Reload()
		}
		return nil
	}

	// 启动热更新监听器
	watcher := hotreload.New(reload, config.ConfigDir)
	go func() {
		_ = watcher.Start(context.Background())
	}()

	// 启动 HTTP 服务
	server := app.NewServer(scheduler, met)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: server.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("strait listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server crashed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
	}
	slog.Info("strait stopped")
}
