package biz

import (
	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// GetPlantLinePreview returns decorative plant-line steps from config (no hardcoded chain).
// Preference: latest in-progress trace session routing → RT-CASSAVA-FRESH → latest active routing.
func (s *Services) GetPlantLinePreview(c *gin.Context) {
	var routingID int64
	source := "none"
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_trace_production
		WHERE status='in_progress' AND COALESCE(routing_id,0)>0 ORDER BY id DESC LIMIT 1`).Scan(&routingID)
	if routingID > 0 {
		source = "active_session"
	} else {
		_ = s.DB.QueryRow(`SELECT id FROM pd_routing
			WHERE code=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, freshCassavaRoutingCode).Scan(&routingID)
		if routingID > 0 {
			source = "fresh_cassava"
		} else {
			_ = s.DB.QueryRow(`SELECT id FROM pd_routing
				WHERE status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`).Scan(&routingID)
			if routingID > 0 {
				source = "active_routing"
			}
		}
	}
	if routingID <= 0 {
		api.OK(c, gin.H{
			"routing_id": 0, "routing_code": "", "routing_name": "",
			"source": source, "message": "未配置工艺", "steps": []gin.H{},
		})
		return
	}
	code, name, _ := s.loadRoutingMeta(routingID)
	raw := s.loadRoutingStepsByID(routingID)
	steps := make([]gin.H, 0, len(raw))
	for i, st := range raw {
		label := st.StepName
		if label == "" {
			label = st.ProcessName
		}
		steps = append(steps, gin.H{
			"name": label, "process_id": st.ProcessID, "seq_no": st.SeqNo,
			"run": i < len(raw)-1 || len(raw) == 1, "end": i == len(raw)-1,
		})
	}
	api.OK(c, gin.H{
		"routing_id": routingID, "routing_code": code, "routing_name": name,
		"source": source, "steps": steps,
	})
}
