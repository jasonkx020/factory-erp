package alert

import (
	"log"
	"sync"
	"time"
)

// Channel 基础告警：进程/DB/MQTT 异常写入日志（可扩展 webhook）。
type Channel struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	minGap   time.Duration
}

// Default 全局通道。
var Default = &Channel{lastSent: map[string]time.Time{}, minGap: time.Minute}

func (a *Channel) Warn(kind, message string) {
	if a == nil {
		log.Printf("[ALERT] %s: %s", kind, message)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if t, ok := a.lastSent[kind]; ok && now.Sub(t) < a.minGap {
		return
	}
	a.lastSent[kind] = now
	log.Printf("[ALERT] %s: %s", kind, message)
}
