// Strait — AI 代理网关入口。
package main

import (
	"context"
	"log"
	"net/http"

	"strait/internal/metrics"

	"strait/internal/config"

	"strait/internal/hotreload"

	"strait/internal/app"
	"strait/internal/plugin"
	"strait/internal/router"
	_ "strait/plugins/adapter-ollama"
	_ "strait/plugins/adapter-openai"
	_ "strait/plugins/auth-static-token"
)

func main() {
	loader := plugin.NewLoader(config.PluginsPath)
	m, err := loader.Build()
	if err != nil {
		log.Fatal(err)
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

	server := app.NewServer(scheduler, met)
	log.Println("strait listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", server.Handler()))
}
