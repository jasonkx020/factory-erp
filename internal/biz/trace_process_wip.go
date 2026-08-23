package biz

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

var processIssueLocationColsOnce sync.Once

// traceProcessWipRow is per (trace, process) material balance.
type traceProcessWipRow struct {
	ProcessID    int64
	ProcessName  string
	AvailableKg  float64 // board weight at process
	PoolKg       float64 // worker_id=0 open issue remainder at process
	OccupiedKg   float64 // worker holdings at process
	WipKg        float64
	IssuableKg   float64 // available + pool (material not held by workers)
	CanIssueFrom bool
	CanStockIn   bool
}

// computeTraceProcessWip aggregates WIP for all processes on a trace code.
func (s *Services) computeTraceProcessWip(trace string) map[int64]*traceProcessWipRow {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	out := map[int64]*traceProcessWipRow{}
	ensure := func(pid int64) *traceProcessWipRow {
		if pid <= 0 {
			return nil
		}
		if r, ok := out[pid]; ok {
			return r
		}
		r := &traceProcessWipRow{ProcessID: pid, ProcessName: s.processName(pid)}
		out[pid] = r
		return r
	}
	brows, err := s.DB.Query(`SELECT b.id, COALESCE(b.current_process_id,0), COALESCE(b.weight, b.qty, 0),
		COALESCE((SELECT SUM(l.qty) FROM inv_balance l WHERE l.box_code_id=b.id),0)
		FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND UPPER(COALESCE(b.trace_code,''))=UPPER(?)
		  AND COALESCE(b.status,'') NOT IN ('destroyed','void')`, trace)
	if err == nil {
		for brows.Next() {
			var bid, pid int64
			var w, bal float64
			if brows.Scan(&bid, &pid, &w, &bal) != nil || pid <= 0 {
				continue
			}
			at := w
			if bal > at {
				at = bal
			}
			if at <= kgEps {
				continue
			}
			r := ensure(pid)
			r.AvailableKg = roundKg(r.AvailableKg + at)
		}
		brows.Close()
	}
	irows, _ := s.DB.Query(`SELECT process_id, COALESCE(worker_id,0), COALESCE(biz_status,''),
		COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND status='open'
		  AND COALESCE(biz_status,'') NOT IN ('issue_pending_warehouse','return_pending','issue_rejected')
		GROUP BY process_id, COALESCE(worker_id,0), COALESCE(biz_status,'')`, trace)
	if irows != nil {
		for irows.Next() {
			var pid, wid int64
			var biz string
			var rem float64
			if irows.Scan(&pid, &wid, &biz, &rem) != nil || pid <= 0 || rem <= kgEps {
				continue
			}
			r := ensure(pid)
			occ, pool := splitTraceIssueRemain(wid, biz, rem)
			r.OccupiedKg = roundKg(r.OccupiedKg + occ)
			r.PoolKg = roundKg(r.PoolKg + pool)
		}
		irows.Close()
	}
	for _, r := range out {
		r.WipKg = roundKg(r.AvailableKg + r.PoolKg + r.OccupiedKg)
		r.IssuableKg = roundKg(r.AvailableKg + r.PoolKg)
		r.CanIssueFrom = r.WipKg > kgEps
		r.CanStockIn = r.WipKg > kgEps
	}
	return out
}

// splitTraceIssueRemain classifies open issue remainder into occupied (worker in progress) vs pool (issuable).
// work_done output is treated as pool so downstream领料 can consume foreman-confirmed产量.
func splitTraceIssueRemain(workerID int64, bizStatus string, rem float64) (occupied, pool float64) {
	if rem <= kgEps {
		return 0, 0
	}
	biz := strings.TrimSpace(bizStatus)
	if workerID <= 0 || biz == "work_done" {
		return 0, rem
	}
	return rem, 0
}

// creditTraceProcessPoolKg adds unassigned WIP at a process (e.g. process-stop output) into the trace pool.
func (s *Services) creditTraceProcessPoolKg(trace string, processID int64, kg float64) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	kg = roundKg(kg)
	if trace == "" || processID <= 0 || kg <= kgEps {
		return
	}
	s.ensureProcessIssueLocationColumns()
	var id int64
	var cur float64
	err := s.DB.QueryRow(`SELECT id, issue_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open'
		  AND COALESCE(worker_id,0)=0 AND COALESCE(biz_status,'open') IN ('open','')
		ORDER BY id LIMIT 1`, trace, processID).Scan(&id, &cur)
	if err == nil && id > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_process_issue SET issue_kg=?, updated_at=NOW() WHERE id=?`, roundKg(cur+kg), id)
		return
	}
	_, _ = s.DB.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, worker_id,
		issue_kg, returned_kg, completed_kg, status, biz_status, source)
		VALUES(0,'',?,?,?, ?,0,0,'open','open','process_output')`, trace, processID, 0, kg)
}

func (s *Services) traceProcessWipRow(trace string, processID int64) *traceProcessWipRow {
	m := s.computeTraceProcessWip(trace)
	if r, ok := m[processID]; ok {
		return r
	}
	return &traceProcessWipRow{ProcessID: processID, ProcessName: s.processName(processID)}
}

// assertTraceProcessWip checks process has enough total WIP for outbound from process.
func (s *Services) assertTraceProcessWip(trace string, processID int64, kg float64, requirePositive bool) string {
	r := s.traceProcessWipRow(trace, processID)
	if requirePositive && r.WipKg <= kgEps {
		return "PROCESS_WIP_EMPTY"
	}
	if kg > kgEps && kg-r.WipKg > kgEps {
		return "PROCESS_WIP_EMPTY"
	}
	return ""
}

func (s *Services) traceWipStepsJSON(trace string) []gin.H {
	m := s.computeTraceProcessWip(trace)
	steps := make([]gin.H, 0, len(m))
	for _, r := range m {
		steps = append(steps, gin.H{
			"process_id": r.ProcessID, "process_name": r.ProcessName,
			"available_kg": r.AvailableKg, "pool_kg": r.PoolKg, "occupied_kg": r.OccupiedKg,
			"wip_kg": r.WipKg, "issuable_kg": r.IssuableKg, "source_limit_kg": r.WipKg,
			"can_issue_from": r.CanIssueFrom, "can_stock_in": r.CanStockIn,
		})
	}
	return steps
}

func (s *Services) getTraceMaterialLocations(c *gin.Context) bool {
	trace := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	if trace == "" {
		trace = strings.ToUpper(strings.TrimSpace(c.Query("trace_code")))
	}
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	wipMap := s.computeTraceProcessWip(trace)
	processes := []gin.H{}
	for _, r := range wipMap {
		if r.WipKg <= kgEps {
			continue
		}
		processes = append(processes, gin.H{
			"process_id": r.ProcessID, "process_name": r.ProcessName,
			"wip_kg": r.WipKg, "source_limit_kg": r.WipKg, "issuable_kg": r.IssuableKg,
			"available_kg": r.AvailableKg, "occupied_kg": r.OccupiedKg, "pool_kg": r.PoolKg,
		})
	}
	api.OK(c, gin.H{
		"trace_code": trace,
		"warehouse":  gin.H{"location_type": "warehouse", "label": "仓库", "selectable": true},
		"processes":  processes,
		"steps":      s.traceWipStepsJSON(trace),
	})
	return true
}

func (s *Services) ensureProcessIssueLocationColumns() {
	processIssueLocationColsOnce.Do(func() {
		if s == nil || s.DB == nil {
			return
		}
		_, _ = s.DB.Exec(`ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS from_location_type TEXT NOT NULL DEFAULT ''`)
		_, _ = s.DB.Exec(`ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS from_process_id BIGINT NOT NULL DEFAULT 0`)
		_, _ = s.DB.Exec(`ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS to_process_id BIGINT NOT NULL DEFAULT 0`)
		_, _ = s.DB.Exec(`ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS warehouse_id BIGINT NOT NULL DEFAULT 0`)
		_, _ = s.DB.Exec(`ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS assigned_board_code TEXT NOT NULL DEFAULT ''`)
	})
}
