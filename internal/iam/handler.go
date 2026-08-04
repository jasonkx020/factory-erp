package iam

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

type Handler struct {
	DB *sql.DB
}

func Register(r *gin.RouterGroup, h *Handler) {
	g := r.Group("/iam")
	g.GET("/users", h.ListUsers)
	g.GET("/roles", h.ListRoles)
	g.GET("/admin-groups", h.ListGroups)
	g.GET("/permissions", h.ListPermissions)
	g.GET("/login-policy", h.GetLoginPolicy)
	g.GET("/field-policies", h.ListFieldPolicies)
	g.GET("/menus", h.ListMenus)
	// 其余写操作骨架
	g.POST("/users", api.NotImplemented)
	g.PUT("/users/:id", api.NotImplemented)
	g.PUT("/users/:id/roles", api.NotImplemented)
	g.POST("/users/:id/freeze", api.NotImplemented)
	g.POST("/users/:id/unfreeze", api.NotImplemented)
	g.POST("/admin-groups", api.NotImplemented)
	g.PUT("/roles/:id/permissions", api.NotImplemented)
	g.PUT("/menus", api.NotImplemented)
	g.PUT("/field-policies", api.NotImplemented)
	g.PUT("/login-policy", api.NotImplemented)
}

func (h *Handler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT u.id, u.login_name, u.user_type, u.status, COALESCE(e.name,''), COALESCE(u.employee_id,0)
		FROM iam_user u LEFT JOIN hr_employee e ON e.id = u.employee_id
		WHERE u.is_deleted = 0 ORDER BY u.id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, empID int64
		var login, ut, status, name string
		_ = rows.Scan(&id, &login, &ut, &status, &name, &empID)
		list = append(list, gin.H{
			"id": id, "login_name": login, "user_type": ut, "status": status,
			"name": name, "employee_id": empID,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) ListRoles(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, code, name, data_scope_type, is_system, status FROM iam_role WHERE is_deleted = 0`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, scope, status string
		var isSys int
		_ = rows.Scan(&id, &code, &name, &scope, &isSys, &status)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "data_scope_type": scope,
			"is_system": isSys == 1, "status": status,
		})
	}
	api.OK(c, gin.H{"list": list})
}

func (h *Handler) ListGroups(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, code, name, COALESCE(remark,''), sort_no, status FROM iam_admin_group WHERE is_deleted = 0 ORDER BY sort_no`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, remark, status string
		var sortNo int
		_ = rows.Scan(&id, &code, &name, &remark, &sortNo, &status)
		list = append(list, gin.H{"id": id, "code": code, "name": name, "remark": remark, "sort_no": sortNo, "status": status})
	}
	api.OK(c, gin.H{"list": list})
}

func (h *Handler) ListPermissions(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, code, name, domain, module, action FROM iam_permission WHERE is_deleted = 0`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, domain, module, action string
		_ = rows.Scan(&id, &code, &name, &domain, &module, &action)
		list = append(list, gin.H{"id": id, "code": code, "name": name, "domain": domain, "module": module, "action": action})
	}
	api.OK(c, gin.H{"list": list})
}

func (h *Handler) GetLoginPolicy(c *gin.Context) {
	var maxFail, lockMin, ttl, minLen, hist int
	var reqLetter, reqDigit, reqSpecial, single int
	err := h.DB.QueryRow(`
		SELECT max_fail_count, lock_minutes, session_ttl_min, password_min_len,
		password_require_letter, password_require_digit, password_require_special,
		password_history, single_session FROM iam_login_policy ORDER BY id LIMIT 1`).
		Scan(&maxFail, &lockMin, &ttl, &minLen, &reqLetter, &reqDigit, &reqSpecial, &hist, &single)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	api.OK(c, gin.H{
		"max_fail_count": maxFail, "lock_minutes": lockMin, "session_ttl_min": ttl,
		"password_min_len": minLen, "password_require_letter": reqLetter == 1,
		"password_require_digit": reqDigit == 1, "password_require_special": reqSpecial == 1,
		"password_history": hist, "single_session": single == 1,
	})
}

func (h *Handler) ListFieldPolicies(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT role_id, field_key, COALESCE(field_name,''), visible, editable FROM iam_field_policy`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var roleID int64
		var key, name string
		var vis, edit int
		_ = rows.Scan(&roleID, &key, &name, &vis, &edit)
		list = append(list, gin.H{"role_id": roleID, "field_key": key, "field_name": name, "visible": vis == 1, "editable": edit == 1})
	}
	api.OK(c, gin.H{"list": list})
}

func (h *Handler) ListMenus(c *gin.Context) {
	roleID := c.Query("role_id")
	if roleID == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	rows, err := h.DB.Query(`SELECT domain, module, menu_key, visible, sort_no FROM iam_menu_custom WHERE role_id = ? ORDER BY sort_no`, roleID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var domain, module, key string
		var vis, sortNo int
		_ = rows.Scan(&domain, &module, &key, &vis, &sortNo)
		list = append(list, gin.H{"domain": domain, "module": module, "menu_key": key, "visible": vis == 1, "sort_no": sortNo})
	}
	api.OK(c, gin.H{"list": list})
}
