package biz

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/biz/tablespec"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleTableCRUD(c *gin.Context, resourceKey, action string) bool {
	spec := tablespec.Registry[resourceKey]
	if spec == nil {
		return false
	}
	switch {
	case action == "list":
		s.tableList(c, spec)
	case action == "create":
		s.tableCreate(c, spec)
	case action == "get":
		s.tableGet(c, spec)
	case action == "update", action == "replace":
		s.tableUpdate(c, spec)
	case action == "delete":
		s.tableDelete(c, spec)
	case strings.HasPrefix(action, "action:"):
		s.tableAction(c, spec, strings.TrimPrefix(action, "action:"))
	default:
		return false
	}
	return true
}

func (s *Services) tableList(c *gin.Context, spec *tablespec.Spec) {
	pageNum, pageSize := sqlutil.Page(c)
	where := "1=1"
	if spec.SoftDelete {
		where += " AND COALESCE(is_deleted,0)=0"
	}
	var total int
	_ = s.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE %s`, spec.Table, where)).Scan(&total)
	offset := (pageNum - 1) * pageSize
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT * FROM %s WHERE %s ORDER BY id DESC LIMIT ? OFFSET ?`, spec.Table, where), pageSize, offset)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list, err := rowsToMaps(rows)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	api.PageOK(c, list, total, pageNum, pageSize)
}

func (s *Services) tableGet(c *gin.Context, spec *tablespec.Spec) {
	id := paramID(c)
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT * FROM %s WHERE id=?`, spec.Table), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list, err := rowsToMaps(rows)
	if err != nil || len(list) == 0 {
		api.FailJSON(c, "NOT_FOUND")
		return
	}
	m := list[0]
	if spec.Lines != nil {
		lrows, _ := s.DB.Query(fmt.Sprintf(`SELECT * FROM %s WHERE %s=? ORDER BY %s`, spec.Lines.Table, spec.Lines.FK, spec.Lines.OrderBy), id)
		if lrows != nil {
			defer lrows.Close()
			lines, _ := rowsToMaps(lrows)
			m["lines"] = lines
			m["steps"] = lines
		}
	}
	api.OK(c, m)
}

func (s *Services) tableCreate(c *gin.Context, spec *tablespec.Spec) {
	body := bindBody(c)
	if spec.DocNo != "" {
		if v, _ := body[spec.DocNo].(string); v == "" {
			body[spec.DocNo] = fmt.Sprintf("%s-%d", strings.ReplaceAll(spec.Table, "_", ""), time.Now().UnixNano()%1e12)
		}
	}
	if spec.Status != "" {
		if _, ok := body[spec.Status]; !ok {
			body[spec.Status] = "draft"
			if spec.Table == "pd_routing" || spec.Table == "inv_box_code" {
				body[spec.Status] = "active"
			}
			if spec.Table == "pd_flow_event" {
				body[spec.Status] = "ok"
			}
		}
	}
	cols := make([]string, 0)
	vals := make([]interface{}, 0)
	ph := make([]string, 0)
	for _, col := range spec.Cols {
		if v, ok := body[col.Name]; ok && v != nil {
			cols = append(cols, col.Name)
			vals = append(vals, coerce(col, v))
			ph = append(ph, "?")
		}
	}
	if len(cols) == 0 {
		api.FailJSON(c, "EMPTY_BODY")
		return
	}
	res, err := s.DB.Exec(fmt.Sprintf(`INSERT INTO %s(%s) VALUES(%s)`, spec.Table, strings.Join(cols, ","), strings.Join(ph, ",")), vals...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return
	}
	id, _ := res.LastInsertId()
	if spec.Lines != nil {
		if lines, ok := body["lines"].([]interface{}); ok {
			s.insertLines(spec, id, lines)
		}
		if steps, ok := body["steps"].([]interface{}); ok {
			s.insertLines(spec, id, steps)
		}
	}
	body["id"] = id
	api.OK(c, body)
}

func (s *Services) insertLines(spec *tablespec.Spec, parentID int64, lines []interface{}) {
	for _, ln := range lines {
		m, _ := ln.(map[string]interface{})
		if m == nil {
			continue
		}
		cols := []string{spec.Lines.FK}
		vals := []interface{}{parentID}
		ph := []string{"?"}
		for _, col := range spec.Lines.Cols {
			if v, ok := m[col.Name]; ok && v != nil {
				cols = append(cols, col.Name)
				vals = append(vals, coerce(col, v))
				ph = append(ph, "?")
			}
		}
		_, _ = s.DB.Exec(fmt.Sprintf(`INSERT INTO %s(%s) VALUES(%s)`, spec.Lines.Table, strings.Join(cols, ","), strings.Join(ph, ",")), vals...)
	}
}

func (s *Services) tableUpdate(c *gin.Context, spec *tablespec.Spec) {
	id := paramID(c)
	body := bindBody(c)
	sets := make([]string, 0)
	vals := make([]interface{}, 0)
	for _, col := range spec.Cols {
		if v, ok := body[col.Name]; ok {
			sets = append(sets, col.Name+"=?")
			vals = append(vals, coerce(col, v))
		}
	}
	hasLines := false
	if spec.Lines != nil {
		if _, ok := body["lines"].([]interface{}); ok {
			hasLines = true
		}
		if _, ok := body["steps"].([]interface{}); ok {
			hasLines = true
		}
	}
	if len(sets) == 0 && !hasLines {
		api.FailJSON(c, "EMPTY_BODY")
		return
	}
	if len(sets) > 0 {
		vals = append(vals, id)
		_, err := s.DB.Exec(fmt.Sprintf(`UPDATE %s SET %s WHERE id=?`, spec.Table, strings.Join(sets, ",")), vals...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
	}
	if spec.Lines != nil {
		if lines, ok := body["lines"].([]interface{}); ok {
			_, _ = s.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s=?`, spec.Lines.Table, spec.Lines.FK), id)
			s.insertLines(spec, id, lines)
		}
		if steps, ok := body["steps"].([]interface{}); ok {
			_, _ = s.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s=?`, spec.Lines.Table, spec.Lines.FK), id)
			s.insertLines(spec, id, steps)
		}
	}
	body["id"] = id
	api.OK(c, body)
}

func (s *Services) tableDelete(c *gin.Context, spec *tablespec.Spec) {
	// 闭环业务：禁止物理删除，仅允许软删/作废
	if !spec.SoftDelete {
		api.FailJSON(c, "DELETE_FORBIDDEN")
		return
	}
	id := paramID(c)
	var err error
	if spec.SoftDelete {
		_, err = s.DB.Exec(fmt.Sprintf(`UPDATE %s SET is_deleted=1 WHERE id=?`, spec.Table), id)
	} else {
		_, err = s.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id=?`, spec.Table), id)
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	api.OK(c, gin.H{"id": id})
}

func (s *Services) tableAction(c *gin.Context, spec *tablespec.Spec, name string) {
	id := paramID(c)
	if st, ok := spec.Actions[name]; ok && spec.Status != "" {
		_, err := s.DB.Exec(fmt.Sprintf(`UPDATE %s SET %s=? WHERE id=?`, spec.Table, spec.Status), st, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		api.OK(c, gin.H{"id": id, "status": st, "action": name})
		return
	}
	api.FailJSON(c, "UNKNOWN_ACTION")
}

func coerce(col tablespec.Col, v interface{}) interface{} {
	switch col.Type {
	case tablespec.TypeInt:
		n, _ := asInt64(v)
		return n
	case tablespec.TypeFloat:
		f, _ := asFloat(v)
		return f
	case tablespec.TypeBool:
		switch t := v.(type) {
		case bool:
			if t {
				return 1
			}
			return 0
		case float64:
			if t != 0 {
				return 1
			}
			return 0
		case string:
			if t == "1" || strings.EqualFold(t, "true") {
				return 1
			}
			return 0
		default:
			return 0
		}
	default:
		return fmt.Sprint(v)
	}
}

func rowsToMaps(rows interface {
	Columns() ([]string, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]interface{}{}
	for rows.Next() {
		raw := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := map[string]interface{}{}
		for i, col := range cols {
			m[col] = normalizeSQL(raw[i])
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func normalizeSQL(v interface{}) interface{} {
	switch t := v.(type) {
	case []byte:
		s := string(t)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil && strings.Contains(s, ".") {
			return f
		}
		return s
	default:
		return v
	}
}
