package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type metricsStore struct {
	mu           sync.Mutex
	reqTotal     atomic.Int64
	errTotal     atomic.Int64
	permDenied   atomic.Int64
	latencySumMs atomic.Int64
	latencyCount atomic.Int64
	byStatus     map[int]*atomic.Int64
}

var globalMetrics = &metricsStore{byStatus: map[int]*atomic.Int64{}}

func (m *metricsStore) incStatus(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.byStatus[code]
	if !ok {
		c = &atomic.Int64{}
		m.byStatus[code] = c
	}
	c.Add(1)
}

// Metrics 记录请求量、延迟、错误与鉴权拒绝。
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		globalMetrics.reqTotal.Add(1)
		ms := time.Since(start).Milliseconds()
		globalMetrics.latencySumMs.Add(ms)
		globalMetrics.latencyCount.Add(1)
		status := c.Writer.Status()
		globalMetrics.incStatus(status)
		if status >= 500 {
			globalMetrics.errTotal.Add(1)
		}
		if status == http.StatusForbidden {
			globalMetrics.permDenied.Add(1)
		}
	}
}

// RegisterMetrics 挂载 /metrics（Prometheus 文本）。
func RegisterMetrics(r *gin.RouterGroup) {
	r.GET("/metrics", func(c *gin.Context) {
		req := globalMetrics.reqTotal.Load()
		errN := globalMetrics.errTotal.Load()
		perm := globalMetrics.permDenied.Load()
		sum := globalMetrics.latencySumMs.Load()
		cnt := globalMetrics.latencyCount.Load()
		avg := float64(0)
		if cnt > 0 {
			avg = float64(sum) / float64(cnt)
		}
		var b strings.Builder
		b.WriteString("# HELP erp_http_requests_total Total HTTP requests\n")
		b.WriteString("# TYPE erp_http_requests_total counter\n")
		fmt.Fprintf(&b, "erp_http_requests_total %d\n", req)
		b.WriteString("# HELP erp_http_errors_total Total HTTP 5xx responses\n")
		b.WriteString("# TYPE erp_http_errors_total counter\n")
		fmt.Fprintf(&b, "erp_http_errors_total %d\n", errN)
		b.WriteString("# HELP erp_http_perm_denied_total Total HTTP 403 responses\n")
		b.WriteString("# TYPE erp_http_perm_denied_total counter\n")
		fmt.Fprintf(&b, "erp_http_perm_denied_total %d\n", perm)
		b.WriteString("# HELP erp_http_latency_ms_avg Average request latency in ms\n")
		b.WriteString("# TYPE erp_http_latency_ms_avg gauge\n")
		fmt.Fprintf(&b, "erp_http_latency_ms_avg %.3f\n", avg)
		globalMetrics.mu.Lock()
		for code, a := range globalMetrics.byStatus {
			fmt.Fprintf(&b, "erp_http_responses_total{code=\"%d\"} %d\n", code, a.Load())
		}
		globalMetrics.mu.Unlock()
		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
	})
}
