package auth

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/config"
	"erp/internal/middleware"
	"erp/internal/security"
)

type Handler struct {
	DB  *sql.DB
	Cfg *config.Config
}

func Register(r *gin.RouterGroup, h *Handler) {
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.GET("/auth/me", h.Me)
}

type loginReq struct {
	LoginName  string `json:"login_name"`
	Password   string `json:"password"`
	ClientType string `json:"client_type"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.LoginName == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	clientType := security.NormalizeClientType(req.ClientType)
	var id int64
	var hash, userType, status, name string
	var empID sql.NullInt64
	err := h.DB.QueryRow(`
		SELECT u.id, u.password_hash, u.user_type, u.status, u.employee_id, COALESCE(e.name,'')
		FROM iam_user u
		LEFT JOIN hr_employee e ON e.id = u.employee_id
		WHERE u.login_name = ? AND u.is_deleted = 0`, req.LoginName).
		Scan(&id, &hash, &userType, &status, &empID, &name)
	if err == sql.ErrNoRows || !security.CheckPassword(hash, req.Password) {
		api.FailJSON(c, "INVALID_CREDENTIAL")
		return
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	if status == "frozen" {
		api.FailJSON(c, "USER_FROZEN")
		return
	}
	roles, perms := h.loadRolesPerms(id)
	claims := security.Claims{
		UserID:      id,
		LoginName:   req.LoginName,
		UserType:    userType,
		ClientType:  clientType,
		Roles:       roles,
		Permissions: perms,
	}
	access, err := security.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.AccessTTLMin, claims)
	if err != nil {
		api.FailJSON(c, "TOKEN_ERROR")
		return
	}
	refresh, _ := security.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.RefreshTTLMin, claims)
	h.saveSession(c, id, access, clientType, h.Cfg.JWT.AccessTTLMin)
	_, _ = h.DB.Exec(`UPDATE iam_user SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	api.OK(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.Cfg.JWT.AccessTTLMin * 60,
		"client_type":   clientType,
		"user": gin.H{
			"id":          id,
			"login_name":  req.LoginName,
			"user_type":   userType,
			"employee_id": empID.Int64,
			"name":        name,
			"status":      status,
		},
		"roles":       roles,
		"permissions": perms,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	claims, err := security.ParseToken(h.Cfg.JWT.Secret, req.RefreshToken)
	if err != nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return
	}
	claims.ClientType = security.NormalizeClientType(claims.ClientType)
	access, err := security.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.AccessTTLMin, *claims)
	if err != nil {
		api.FailJSON(c, "TOKEN_ERROR")
		return
	}
	h.saveSession(c, claims.UserID, access, claims.ClientType, h.Cfg.JWT.AccessTTLMin)
	api.OK(c, gin.H{
		"access_token":  access,
		"refresh_token": req.RefreshToken,
		"expires_in":    h.Cfg.JWT.AccessTTLMin * 60,
		"client_type":   claims.ClientType,
		"roles":         claims.Roles,
		"permissions":   claims.Permissions,
	})
}

func (h *Handler) Me(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	menus := h.loadMenus(claims.UserID)
	fields := h.loadFieldPolicies(claims.UserID)
	api.OK(c, gin.H{
		"user": gin.H{
			"id":         claims.UserID,
			"login_name": claims.LoginName,
			"user_type":  claims.UserType,
		},
		"client_type":    claims.ClientType,
		"roles":          claims.Roles,
		"permissions":    claims.Permissions,
		"menus":          menus,
		"field_policies": fields,
	})
}

func (h *Handler) saveSession(c *gin.Context, userID int64, accessToken, clientType string, ttlMin int) {
	expireAt := time.Now().Add(time.Duration(ttlMin) * time.Minute).UTC().Format("2006-01-02 15:04:05.000")
	_, _ = h.DB.Exec(`
		INSERT INTO iam_user_session (user_id, token_hash, client_type, ip, user_agent, expire_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID,
		security.HashToken(accessToken),
		clientType,
		c.ClientIP(),
		truncateUA(c.Request.UserAgent()),
		expireAt,
	)
}

func truncateUA(ua string) string {
	if len(ua) > 512 {
		return ua[:512]
	}
	return ua
}

func (h *Handler) loadRolesPerms(userID int64) ([]string, []string) {
	roles := []string{}
	rows, err := h.DB.Query(`
		SELECT r.code FROM iam_user_role ur
		JOIN iam_role r ON r.id = ur.role_id
		WHERE ur.user_id = ?`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			_ = rows.Scan(&code)
			roles = append(roles, code)
		}
	}
	perms := []string{}
	prows, err := h.DB.Query(`
		SELECT DISTINCT p.code FROM iam_user_role ur
		JOIN iam_role_permission rp ON rp.role_id = ur.role_id
		JOIN iam_permission p ON p.id = rp.permission_id
		WHERE ur.user_id = ?`, userID)
	if err == nil {
		defer prows.Close()
		for prows.Next() {
			var code string
			_ = prows.Scan(&code)
			perms = append(perms, code)
		}
	}
	return roles, perms
}

func (h *Handler) loadMenus(userID int64) []gin.H {
	out := []gin.H{}
	rows, err := h.DB.Query(`
		SELECT m.domain, m.module, m.menu_key, m.visible, m.sort_no
		FROM iam_menu_custom m
		JOIN iam_user_role ur ON ur.role_id = m.role_id
		WHERE ur.user_id = ?
		ORDER BY m.sort_no`, userID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var domain, module, key string
		var visible, sortNo int
		_ = rows.Scan(&domain, &module, &key, &visible, &sortNo)
		out = append(out, gin.H{
			"domain": domain, "module": module, "menu_key": key,
			"visible": visible == 1, "sort_no": sortNo,
		})
	}
	return out
}

func (h *Handler) loadFieldPolicies(userID int64) []gin.H {
	out := []gin.H{}
	rows, err := h.DB.Query(`
		SELECT fp.role_id, fp.field_key, COALESCE(fp.field_name,''), fp.visible, fp.editable
		FROM iam_field_policy fp
		JOIN iam_user_role ur ON ur.role_id = fp.role_id
		WHERE ur.user_id = ?`, userID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var roleID int64
		var key, name string
		var visible, editable int
		_ = rows.Scan(&roleID, &key, &name, &visible, &editable)
		out = append(out, gin.H{
			"role_id": roleID, "field_key": key, "field_name": name,
			"visible": visible == 1, "editable": editable == 1,
		})
	}
	return out
}
