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

// snapshotBoardYield writes per-process yield once after 生管 closes the board (status=finished).
func (s *Services) snapshotBoardYield(boardID int64) {
	if boardID <= 0 {
		return
	}
	var code, trace, status string
	err := s.DB.QueryRow(`SELECT COALESCE(code,''), COALESCE(trace_code,''), COALESCE(status,'')
		FROM inv_box_code WHERE id=? AND COALESCE(is_deleted,0)=0`, boardID).Scan(&code, &trace, &status)
	if err != nil || status != "finished" || strings.TrimSpace(trace) == "" {
		return
	}
	var bad int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE board_id=? AND COALESCE(trace_code,'')=''`, boardID).Scan(&bad)
	if bad > 0 {
		return
	}
	for _, pid := range s.boardCloseProcessIDs(boardID) {
		s.snapshotBoardProcessYield(boardID, pid)
	}
	s.snapshotTraceYield(trace)
}

func (s *Services) snapshotBoardProcessYield(boardID, processID int64) {
	if boardID <= 0 || processID <= 0 {
		return
	}
	var code, trace, status string
	err := s.DB.QueryRow(`SELECT COALESCE(code,''), COALESCE(trace_code,''), COALESCE(status,'')
		FROM inv_box_code WHERE id=? AND COALESCE(is_deleted,0)=0`, boardID).Scan(&code, &trace, &status)
	if err != nil || status != "finished" || strings.TrimSpace(trace) == "" {
		return
	}
	var inKg, outKg float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND COALESCE(worker_id,0)>0`, boardID, processID).Scan(&inKg)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(kg),0)
		FROM pd_process_move WHERE board_id=? AND from_process_id=?`, boardID, processID).Scan(&outKg)
	inKg, outKg = roundKg(inKg), roundKg(outKg)
	if inKg <= kgEps && outKg <= kgEps {
		return
	}
	loss := roundKg(inKg - outKg)
	rate := yieldLossRate(inKg, loss)
	_, _ = s.DB.Exec(`INSERT INTO pd_board_process_yield(board_id, board_code, trace_code, process_id, input_kg, output_kg, loss_kg, loss_rate)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT (board_id, process_id) DO NOTHING`,
		boardID, code, trace, processID, inKg, outKg, loss, rate)
}

func (s *Services) snapshotTraceYield(trace string) {
	trace = strings.TrimSpace(trace)
	if trace == "" {
		return
	}
	var unfinished int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND trace_code=?
		  AND COALESCE(status,'') NOT IN ('finished','destroyed','void')`, trace).Scan(&unfinished)
	if unfinished > 0 {
		return
	}
	var finished int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND trace_code=? AND status='finished'`, trace).Scan(&finished)
	if finished <= 0 {
		return
	}
	rows, err := s.DB.Query(`SELECT process_id,
		COALESCE(SUM(input_kg),0), COALESCE(SUM(output_kg),0), COALESCE(SUM(loss_kg),0), COUNT(DISTINCT board_id)
		FROM pd_board_process_yield WHERE trace_code=?
		GROUP BY process_id`, trace)
	if err != nil {
		return
	}
	type row struct {
		pid                 int64
		inKg, outKg, lossKg float64
		boards              int
	}
	list := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.pid, &r.inKg, &r.outKg, &r.lossKg, &r.boards); err != nil {
			continue
		}
		list = append(list, r)
	}
	rows.Close()
	for _, r := range list {
		inKg, outKg, lossKg := roundKg(r.inKg), roundKg(r.outKg), roundKg(r.lossKg)
		rate := yieldLossRate(inKg, lossKg)
		_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_yield(trace_code, process_id, input_kg, output_kg, loss_kg, loss_rate, board_count)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT (trace_code, process_id) DO NOTHING`,
			trace, r.pid, inKg, outKg, lossKg, rate, r.boards)
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
	return s.listBoardProcessYields(c)
}

func (s *Services) listBoardProcessYields(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if v := strings.TrimSpace(c.Query("trace_code")); v != "" {
		where += ` AND y.trace_code=?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("board_code")); v != "" {
		where += ` AND y.board_code=?`
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
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield y `+where, args...).Scan(&total)
	qargs := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT y.id, y.board_id, y.board_code, y.trace_code, y.process_id, COALESCE(p.name,''),
		y.input_kg, y.output_kg, y.loss_kg, y.loss_rate, CAST(y.created_at AS TEXT)
		FROM pd_board_process_yield y
		LEFT JOIN pd_process p ON p.id=y.process_id
		`+where+` ORDER BY y.id DESC LIMIT ? OFFSET ?`, qargs...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, boardID, processID int64
		var boardCode, trace, processName, created string
		var inKg, outKg, lossKg, rate float64
		if err := rows.Scan(&id, &boardID, &boardCode, &trace, &processID, &processName, &inKg, &outKg, &lossKg, &rate, &created); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "board_id": boardID, "board_code": boardCode, "trace_code": trace,
			"process_id": processID, "process_name": processName,
			"input_kg": inKg, "output_kg": outKg, "loss_kg": lossKg, "loss_rate": rate, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
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
