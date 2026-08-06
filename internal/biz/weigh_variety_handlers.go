package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func (s *Services) handleWeighVarieties(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		status := strings.TrimSpace(c.Query("status"))
		q := `SELECT id, code, name, sort_no, status, COALESCE(default_product_id,0), COALESCE(remark,''), created_at, updated_at
			FROM pur_weigh_variety WHERE COALESCE(is_deleted,0)=0`
		args := []interface{}{}
		if status != "" {
			q += ` AND status=?`
			args = append(args, status)
		}
		q += ` ORDER BY sort_no ASC, id ASC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, sortNo, productID int64
			var code, name, st, remark, created, updated string
			_ = rows.Scan(&id, &code, &name, &sortNo, &st, &productID, &remark, &created, &updated)
			item := gin.H{
				"id": id, "code": code, "name": name, "sort_no": sortNo, "status": st,
				"remark": remark, "created_at": created, "updated_at": updated,
			}
			if productID > 0 {
				item["default_product_id"] = productID
			} else {
				item["default_product_id"] = nil
			}
			list = append(list, item)
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		return s.getWeighVariety(c, paramID(c))
	case "create":
		body := bindBody(c)
		code := strings.TrimSpace(strOr(body["code"]))
		name := strings.TrimSpace(strOr(body["name"]))
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		if code == "" {
			code = fmt.Sprintf("WV%s", time.Now().Format("060102150405"))
		}
		sortNo, _ := asInt64(body["sort_no"])
		status := strOrDef(body["status"], "active")
		productID, _ := asInt64(body["default_product_id"])
		res, err := s.DB.Exec(`INSERT INTO pur_weigh_variety(code, name, sort_no, status, default_product_id, remark)
			VALUES(?,?,?,?,?,?)`, code, name, sortNo, status, nullIf0(productID), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "sort_no": sortNo, "status": status})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		var curCode, curName, curStatus, curRemark string
		var curSort, curProduct int64
		err := s.DB.QueryRow(`SELECT code, name, sort_no, status, COALESCE(default_product_id,0), COALESCE(remark,'')
			FROM pur_weigh_variety WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
			Scan(&curCode, &curName, &curSort, &curStatus, &curProduct, &curRemark)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		if v := strings.TrimSpace(strOr(body["code"])); v != "" {
			curCode = v
		}
		if v := strings.TrimSpace(strOr(body["name"])); v != "" {
			curName = v
		}
		if _, ok := body["sort_no"]; ok {
			curSort, _ = asInt64(body["sort_no"])
		}
		if v := strings.TrimSpace(strOr(body["status"])); v != "" {
			curStatus = v
		}
		if _, ok := body["remark"]; ok {
			curRemark = strOr(body["remark"])
		}
		if _, ok := body["default_product_id"]; ok {
			curProduct, _ = asInt64(body["default_product_id"])
		}
		_, err = s.DB.Exec(`UPDATE pur_weigh_variety SET code=?, name=?, sort_no=?, status=?, default_product_id=?, remark=?, updated_at=datetime('now')
			WHERE id=? AND COALESCE(is_deleted,0)=0`,
			curCode, curName, curSort, curStatus, nullIf0(curProduct), curRemark, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		return s.getWeighVariety(c, id)
	case "delete":
		id := paramID(c)
		_, err := s.DB.Exec(`UPDATE pur_weigh_variety SET is_deleted=1, status='inactive', updated_at=datetime('now') WHERE id=?`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id, "deleted": true})
		return true
	}
	return true
}

func (s *Services) getWeighVariety(c *gin.Context, id int64) bool {
	var sortNo, productID int64
	var code, name, st, remark, created, updated string
	err := s.DB.QueryRow(`SELECT code, name, sort_no, status, COALESCE(default_product_id,0), COALESCE(remark,''), created_at, updated_at
		FROM pur_weigh_variety WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &name, &sortNo, &st, &productID, &remark, &created, &updated)
	if err == sql.ErrNoRows {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	out := gin.H{
		"id": id, "code": code, "name": name, "sort_no": sortNo, "status": st,
		"remark": remark, "created_at": created, "updated_at": updated,
	}
	if productID > 0 {
		out["default_product_id"] = productID
	} else {
		out["default_product_id"] = nil
	}
	api.OK(c, out)
	return true
}

// resolveWeighVariety fills variety name and default product from variety_id when provided.
func (s *Services) resolveWeighVariety(body map[string]interface{}, variety *string, productID *int64) {
	vid, _ := asInt64(body["variety_id"])
	if vid <= 0 {
		return
	}
	var name string
	var defPID int64
	err := s.DB.QueryRow(`SELECT name, COALESCE(default_product_id,0) FROM pur_weigh_variety
		WHERE id=? AND COALESCE(is_deleted,0)=0 AND status='active'`, vid).Scan(&name, &defPID)
	if err != nil {
		return
	}
	if name != "" {
		*variety = name
	}
	if *productID <= 0 && defPID > 0 {
		*productID = defPID
	}
}
