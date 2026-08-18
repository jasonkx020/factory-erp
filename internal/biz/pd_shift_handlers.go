package biz

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func (s *Services) handleProductionShifts(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/members") {
		return s.handleShiftMembers(c, method, openapiPath)
	}
	if strings.Contains(openapiPath, "/close") && method == "POST" {
		return s.closeProductionShift(c)
	}
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM (
			SELECT s.*, w.name AS workshop_name,
			(SELECT COUNT(1) FROM pd_shift_member m WHERE m.shift_id=s.id) AS member_count
			FROM pd_shift s LEFT JOIN sys_department w ON w.id=s.workshop_dept_id
		)`)
	case "create":
		body := bindBody(c)
		workshopID := asInt64Or0(body["workshop_dept_id"])
		if workshopID <= 0 {
			workshopID = s.defaultWorkshopDeptID()
		}
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("SH%s", time.Now().Format("060102150405")))
		bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		status := strOrDef(body["status"], "open")
		res, err := s.DB.Exec(`INSERT INTO pd_shift(doc_no, workshop_dept_id, biz_date, status, remark) VALUES(?,?,?,?,?)`,
			docNo, workshopID, bizDate, status, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "workshop_dept_id": workshopID, "biz_date": bizDate, "status": status})
		return true
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT s.*, w.name AS workshop_name FROM pd_shift s
			LEFT JOIN sys_department w ON w.id=s.workshop_dept_id WHERE s.id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		mrows, _ := s.DB.Query(`SELECT m.id, m.employee_id, m.process_id, e.name AS employee_name, p.name AS process_name
			FROM pd_shift_member m
			LEFT JOIN hr_employee e ON e.id=m.employee_id
			LEFT JOIN pd_process p ON p.id=m.process_id
			WHERE m.shift_id=? ORDER BY m.id`, id)
		members := []gin.H{}
		if mrows != nil {
			defer mrows.Close()
			for mrows.Next() {
				var mid, eid, pid int64
				var ename, pname string
				_ = mrows.Scan(&mid, &eid, &pid, &ename, &pname)
				members = append(members, gin.H{
					"id": mid, "employee_id": eid, "process_id": pid,
					"employee_name": ename, "process_name": pname,
				})
			}
		}
		row := list[0]
		row["members"] = members
		api.OK(c, row)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_shift SET remark=COALESCE(NULLIF(?,''),remark),
			status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			strOr(body["remark"]), strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

func (s *Services) closeProductionShift(c *gin.Context) bool {
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE pd_shift SET status='closed' WHERE id=? AND status='open'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, gin.H{"id": id, "status": "closed"})
	return true
}

func (s *Services) handleShiftMembers(c *gin.Context, method, openapiPath string) bool {
	shiftID := paramID(c)
	if shiftID <= 0 {
		api.FailJSON(c, "SHIFT_ID_REQUIRED")
		return true
	}
	if method == "POST" {
		body := bindBody(c)
		eid := asInt64Or0(body["employee_id"])
		if eid <= 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		pid := asInt64Or0(body["process_id"])
		res, err := s.DB.Exec(`INSERT INTO pd_shift_member(shift_id, employee_id, process_id) VALUES(?,?,?)`,
			shiftID, eid, pid)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		mid, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": mid, "shift_id": shiftID, "employee_id": eid, "process_id": pid})
		return true
	}
	if method == "DELETE" {
		memberID, _ := strconv.ParseInt(c.Param("memberId"), 10, 64)
		if memberID <= 0 {
			api.FailJSON(c, "MEMBER_ID_REQUIRED")
			return true
		}
		_, _ = s.DB.Exec(`DELETE FROM pd_shift_member WHERE id=? AND shift_id=?`, memberID, shiftID)
		api.OK(c, gin.H{})
		return true
	}
	return true
}
