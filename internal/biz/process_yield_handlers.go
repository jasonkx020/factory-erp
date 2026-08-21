package biz

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func yieldLossRate(inputKg, lossKg float64) float64 {
	if inputKg <= kgEps {
		return 0
	}
	return roundKg(lossKg / inputKg)
}

// snapshotTraceYield writes per-process yield for a whole trace (keyed by trace_code only).
// Called when foreman completes trace production — not on board close.
func (s *Services) snapshotTraceYield(trace string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return
	}

	inRows, err := s.DB.Query(`SELECT i.process_id, COALESCE(SUM(i.issue_kg - i.returned_kg),0)
		FROM pd_process_issue i
		WHERE UPPER(COALESCE(i.trace_code,''))=UPPER(?) AND COALESCE(i.worker_id,0)>0
		GROUP BY i.process_id`, trace)
	if err != nil {
		return
	}
	type agg struct {
		inKg, outKg float64
	}
	byProc := map[int64]*agg{}
	for inRows.Next() {
		var pid int64
		var inKg float64
		if err := inRows.Scan(&pid, &inKg); err != nil {
			continue
		}
		byProc[pid] = &agg{inKg: roundKg(inKg)}
	}
	inRows.Close()

	outRows, err := s.DB.Query(`SELECT m.from_process_id, COALESCE(SUM(m.kg),0)
		FROM pd_process_move m
		WHERE UPPER(COALESCE(m.trace_code,''))=UPPER(?) AND COALESCE(m.from_process_id,0)>0
		GROUP BY m.from_process_id`, trace)
	if err != nil {
		return
	}
	for outRows.Next() {
		var pid int64
		var outKg float64
		if err := outRows.Scan(&pid, &outKg); err != nil {
			continue
		}
		a := byProc[pid]
		if a == nil {
			a = &agg{}
			byProc[pid] = a
		}
		a.outKg = roundKg(outKg)
	}
	outRows.Close()

	boardCount := 0
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)`, trace).Scan(&boardCount)

	for pid, a := range byProc {
		inKg, outKg := roundKg(a.inKg), roundKg(a.outKg)
		if inKg <= kgEps && outKg <= kgEps {
			continue
		}
		lossKg := roundKg(inKg - outKg)
		rate := yieldLossRate(inKg, lossKg)
		_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_yield(trace_code, process_id, input_kg, output_kg, loss_kg, loss_rate, board_count)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT (trace_code, process_id) DO UPDATE SET
			input_kg=EXCLUDED.input_kg, output_kg=EXCLUDED.output_kg, loss_kg=EXCLUDED.loss_kg,
			loss_rate=EXCLUDED.loss_rate, board_count=EXCLUDED.board_count`,
			trace, pid, inKg, outKg, lossKg, rate, boardCount)
	}
}

func (s *Services) handleProcessYields(c *gin.Context, method, openapiPath, action string) bool {
	if method != "GET" {
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
	_ = action
	if strings.Contains(openapiPath, "/traces") {
		return s.listTraceProcessYields(c)
	}
	// Board-level yield is discontinued; keep route for compat with empty page.
	return s.listBoardProcessYields(c)
}

func (s *Services) listBoardProcessYields(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	api.PageOK(c, []gin.H{}, 0, pageNum, pageSize)
	return true
}

func (s *Services) listTraceProcessYields(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if v := strings.TrimSpace(c.Query("trace_code")); v != "" {
		where += ` AND y.trace_code=?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("process_id")); v != "" {
		var pid int64
		fmt.Sscanf(v, "%d", &pid)
		if pid > 0 {
			where += ` AND y.process_id=?`
			args = append(args, pid)
		}
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield y `+where, args...).Scan(&total)
	qargs := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT y.id, y.trace_code, y.process_id, COALESCE(p.name,''),
		y.input_kg, y.output_kg, y.loss_kg, y.loss_rate, y.board_count, CAST(y.created_at AS TEXT)
		FROM pd_trace_process_yield y
		LEFT JOIN pd_process p ON p.id=y.process_id
		`+where+` ORDER BY y.id DESC LIMIT ? OFFSET ?`, qargs...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, processID int64
		var trace, processName, created string
		var inKg, outKg, lossKg, rate float64
		var boards int
		if err := rows.Scan(&id, &trace, &processID, &processName, &inKg, &outKg, &lossKg, &rate, &boards, &created); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "trace_code": trace, "process_id": processID, "process_name": processName,
			"input_kg": inKg, "output_kg": outKg, "loss_kg": lossKg, "loss_rate": rate,
			"board_count": boards, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}
