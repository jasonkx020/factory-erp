package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

// canCreateReportWorkBackfill allows direct report-work create only for sys_admin backfill.
func (s *Services) canCreateReportWorkBackfill(c *gin.Context, body map[string]interface{}) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		return false
	}
	if !ClaimsIsSysAdmin(cl.Roles, cl.Permissions) {
		return false
	}
	reason := strings.TrimSpace(strOr(body["backfill_reason"]))
	if reason == "" {
		reason = strings.TrimSpace(strOr(body["remark"]))
	}
	return len(reason) >= 4
}

func (s *Services) rejectReportWorkCreate(c *gin.Context) bool {
	api.FailJSON(c, "FIELD_INPUT_USE_APP")
	return true
}

// workerShiftAuthorized: when pd_shift has open rows for today, worker must be on shift for process.
func (s *Services) workerShiftAuthorized(workerID, processID int64) bool {
	if workerID <= 0 {
		return false
	}
	var openShifts int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_shift WHERE status='open' AND date(biz_date)=date('now')`).Scan(&openShifts)
	if openShifts == 0 {
		return true
	}
	var ok int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_shift_member m
		INNER JOIN pd_shift sh ON sh.id=m.shift_id
		WHERE sh.status='open' AND date(sh.biz_date)=date('now')
		  AND m.employee_id=? AND (COALESCE(m.process_id,0)=0 OR m.process_id=?)`,
		workerID, processID).Scan(&ok)
	return ok > 0
}

// advanceBoxToStep updates box routing pointer without spawning dispatch/work-order.
func (s *Services) advanceBoxToStep(boxID int64, step *routingStep) {
	if boxID <= 0 || step == nil {
		return
	}
	_, _ = s.DB.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, updated_at=datetime('now') WHERE id=?`,
		step.ProcessID, step.ID, boxID)
}
