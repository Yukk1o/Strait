// Package plugin 插件加载、注册、调度。
package plugin

import (
	"strait/api"
	"strings"
)

// Registry 插件注册表
type Registry struct {
	plugin map[string]api.Plugin
}

func NewRegistry() *Registry {
	return &Registry{
		plugin: make(map[string]api.Plugin),
	}
}

func (r *Registry) Register(id string, p api.Plugin) {
	id = strings.ToLower(id)
	if _, ok := r.plugin[id]; ok {
		panic("plugin already registered: " + id)
	}
	r.plugin[id] = p
}

func (r *Registry) Get(id string) (api.Plugin, bool) {
	p, ok := r.plugin[strings.ToLower(id)]
	return p, ok
}

func (r *Registry) Range(fn func(id string, p api.Plugin) bool) {
	for id, p := range r.plugin {
		if !fn(id, p) {
			break
		}
	}
}
