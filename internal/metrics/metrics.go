package metrics

import (
	"fmt"
	"strings"
	"sync"
)

type Metrics struct {
	mu       sync.Mutex       // 请求锁
	requests map[string]int64 // 请求次数(key: "method|path" -> count)
}

func NewMetrics() *Metrics {
	return &Metrics{
		requests: make(map[string]int64),
	}
}

func (m *Metrics) IncRequests(method, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + "|" + path
	m.requests[key]++
}

func (m *Metrics) Handler() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	b.WriteString("# HELP strait_requests_total Total HTTP requests\n")
	b.WriteString("# TYPE strait_requests_total counter\n")
	for key, count := range m.requests {
		parts := strings.SplitN(key, "|", 2)
		b.WriteString(fmt.Sprintf("strait_requests_total{method=\"%s\",path=\"%s\"} %d\n",
			parts[0], parts[1], count))
	}
	return b.String()
}
