package hotreload

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ReloadFunc 重载回调，接收变更的文件名
type ReloadFunc func(filename string) error

// Watcher 热更新监听器
type Watcher struct {
	reload ReloadFunc // 重载回调
	paths  []string   // 路径
}

func New(reload ReloadFunc, paths ...string) *Watcher {
	return &Watcher{reload: reload, paths: paths}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()

	for _, p := range w.paths {
		if err := watcher.Add(p); err != nil {
			return err
		}
	}

	var timer *time.Timer // 定时器

	for {
		select {
		case event := <-watcher.Events:
			if !strings.HasSuffix(event.Name, ".yaml") {
				continue
			}
			name := event.Name
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(500*time.Millisecond, func() {
				if err := w.reload(filepath.Base(name)); err != nil {
					slog.Error("HotReload failed", "error", err)
				} else {
					slog.Info("config reloaded")
				}
			})

		case err := <-watcher.Errors:
			slog.Error("HotReload watcher error", "error", err)

		case <-ctx.Done():
			return nil
		}
	}
}
