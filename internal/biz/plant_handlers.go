package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) handlePlants(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, code, name, status, COALESCE(is_default,0), COALESCE(remark,''), created_at, updated_at
			FROM sys_plant WHERE COALESCE(is_deleted,0)=0 ORDER BY is_default DESC, id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var def int
			var code, name, st, remark, created, updated string
			_ = rows.Scan(&id, &code, &name, &st, &def, &remark, &created, &updated)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "status": st, "is_default": def == 1,
				"remark": remark, "created_at": created, "updated_at": updated,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		id := paramID(c)
		var def int
		var code, name, st, remark, created, updated string
		err := s.DB.QueryRow(`SELECT code, name, status, COALESCE(is_default,0), COALESCE(remark,''), created_at, updated_at
			FROM sys_plant WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
			Scan(&code, &name, &st, &def, &remark, &created, &updated)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "code": code, "name": name, "status": st, "is_default": def == 1,
			"remark": remark, "created_at": created, "updated_at": updated,
		})
		return true
	case "create":
		body := bindBody(c)
		code := strings.TrimSpace(strOr(body["code"]))
		name := strings.TrimSpace(strOr(body["name"]))
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		if code == "" {
			code = fmt.Sprintf("PLANT%s", time.Now().Format("060102150405"))
		}
		status := strOrDef(body["status"], "active")
		isDef := 0
		if asBool(body["is_default"]) {
			isDef = 1
			_, _ = s.DB.Exec(`UPDATE sys_plant SET is_default=0 WHERE COALESCE(is_deleted,0)=0`)
		}
		res, err := s.DB.Exec(`INSERT INTO sys_plant(code, name, status, is_default, remark) VALUES(?,?,?,?,?)`,
			code, name, status, isDef, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		var curCode, curName, curStatus, curRemark string
		var curDef int
		err := s.DB.QueryRow(`SELECT code, name, status, COALESCE(is_default,0), COALESCE(remark,'') FROM sys_plant WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
			Scan(&curCode, &curName, &curStatus, &curDef, &curRemark)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		code := strOrDef(body["code"], curCode)
		name := strOrDef(body["name"], curName)
		status := strOrDef(body["status"], curStatus)
		remark := strOrDef(body["remark"], curRemark)
		isDef := curDef
		if _, ok := body["is_default"]; ok {
			if asBool(body["is_default"]) {
				isDef = 1
				_, _ = s.DB.Exec(`UPDATE sys_plant SET is_default=0 WHERE id<>? AND COALESCE(is_deleted,0)=0`, id)
			} else {
				isDef = 0
			}
		}
		_, err = s.DB.Exec(`UPDATE sys_plant SET code=?, name=?, status=?, is_default=?, remark=?, updated_at=NOW() WHERE id=?`,
			code, name, status, isDef, remark, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE sys_plant SET is_deleted=1, updated_at=NOW() WHERE id=?`, id)
		api.OK(c, gin.H{"id": id})
		return true
	default:
		return false
	}
}

// UserAccessiblePlants returns plant ids allowed for current user (empty = all active).
func (s *Services) UserAccessiblePlants(c *gin.Context) []int64 {
	cl := middleware.Claims(c)
	if cl == nil {
		return nil
	}
	if ClaimsIsSysAdmin(cl.Roles, cl.Permissions) {
		return nil // unrestricted
	}
	ids := []int64{}
	rows, err := s.DB.Query(`SELECT DISTINCT plant_id FROM (
		SELECT plant_id FROM iam_user_plant_scope WHERE user_id=?
		UNION
		SELECT rps.plant_id FROM iam_user_role ur
		JOIN iam_role_plant_scope rps ON rps.role_id=ur.role_id
		WHERE ur.user_id=?
	) t`, cl.UserID, cl.UserID)
	if err != nil {
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *Services) AssertPlantAccess(c *gin.Context, plantID int64) bool {
	if plantID <= 0 {
		return true
	}
	allowed := s.UserAccessiblePlants(c)
	if allowed == nil {
		return true
	}
	if len(allowed) == 0 {
		// no explicit scope → default plant only
		def := EnsureDefaultPlant(s.DB)
		return plantID == def
	}
	for _, id := range allowed {
		if id == plantID {
			return true
		}
	}
	api.FailJSON(c, "PLANT_SCOPE_DENIED")
	return false
}
