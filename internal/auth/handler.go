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
	r.POST("/auth/oauth/token", h.OAuthToken)
	r.POST("/auth/password/change", h.ChangePassword)
	r.GET("/auth/oauth/bindings", h.ListOAuthBindings)
	r.POST("/auth/oauth/bind", h.BindOAuth)
	r.DELETE("/auth/oauth/bind/:provider", h.UnbindOAuth)
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
	var hash, userType, status, name, customerName string
	var empID, customerID sql.NullInt64
	err := h.DB.QueryRow(`
		SELECT u.id, u.password_hash, u.user_type, u.status, u.employee_id, COALESCE(u.customer_id,0),
			COALESCE(NULLIF(cu.name,''), COALESCE(e.name,'')), COALESCE(cu.name,'')
		FROM iam_user u
		LEFT JOIN hr_employee e ON e.id = u.employee_id
		LEFT JOIN crm_customer cu ON cu.id = u.customer_id
		WHERE u.login_name = ? AND u.is_deleted = 0`, req.LoginName).
		Scan(&id, &hash, &userType, &status, &empID, &customerID, &name, &customerName)
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
	if clientType == "customer" && (userType != "customer" || customerID.Int64 <= 0) {
		api.FailJSON(c, "CUSTOMER_ACCOUNT_REQUIRED")
		return
	}
	if userType == "customer" && clientType != "customer" {
		api.FailJSON(c, "CLIENT_TYPE_MISMATCH")
		return
	}
	roles, perms := security.LoadUserRolesPerms(h.DB, id)
	security.InvalidateUserRBAC(id)
	claims := security.SlimForToken(security.Claims{
		UserID:     id,
		LoginName:  req.LoginName,
		UserType:   userType,
		ClientType: clientType,
		Roles:      roles,
	})
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
			"id":            id,
			"login_name":    req.LoginName,
			"user_type":     userType,
			"employee_id":   empID.Int64,
			"customer_id":   customerID.Int64,
			"customer_name": customerName,
			"name":          name,
			"status":        status,
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
	roles, perms := security.LoadUserRolesPerms(h.DB, claims.UserID)
	security.InvalidateUserRBAC(claims.UserID)
	slim := security.SlimForToken(security.Claims{
		UserID:     claims.UserID,
		LoginName:  claims.LoginName,
		UserType:   claims.UserType,
		ClientType: claims.ClientType,
		Roles:      roles,
	})
	access, err := security.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.AccessTTLMin, slim)
	if err != nil {
		api.FailJSON(c, "TOKEN_ERROR")
		return
	}
	// Also rotate refresh to drop legacy fat tokens that embedded permissions.
	refresh, err := security.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.RefreshTTLMin, slim)
	if err != nil {
		refresh = req.RefreshToken
	}
	h.saveSession(c, claims.UserID, access, claims.ClientType, h.Cfg.JWT.AccessTTLMin)
	api.OK(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.Cfg.JWT.AccessTTLMin * 60,
		"client_type":   claims.ClientType,
		"roles":         roles,
		"permissions":   perms,
	})
}

type oauthTokenReq struct {
	Provider   string `json:"provider"`
	Code       string `json:"code"`
	ClientType string `json:"client_type"`
}

// OAuthToken 第三方登录 code 交换（默认未开通，返回 OAUTH_NOT_CONFIGURED）。
func (h *Handler) OAuthToken(c *gin.Context) {
	var req oauthTokenReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Provider == "" || req.Code == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	if h.Cfg == nil || !h.Cfg.OAuth.Enabled {
		api.FailJSON(c, "OAUTH_NOT_CONFIGURED")
		return
	}
	if len(h.Cfg.OAuth.Providers) == 0 || h.Cfg.OAuth.Providers[req.Provider] == "" {
		api.FailJSON(c, "OAUTH_PROVIDER_UNKNOWN")
		return
	}
	// 真实 provider 交换后续接入；启用配置后仍需实现，避免误放行。
	api.FailJSON(c, "OAUTH_NOT_IMPLEMENTED")
}

func (h *Handler) Me(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	roles, perms := security.CachedUserRolesPerms(h.DB, claims.UserID)
	menus := h.loadMenus(claims.UserID)
	fields := h.loadFieldPolicies(claims.UserID)
	var empID, customerID sql.NullInt64
	var empName, empNo, badgeCode, customerName, displayName string
	_ = h.DB.QueryRow(`SELECT u.employee_id, COALESCE(u.customer_id,0),
		COALESCE(e.name,''), COALESCE(e.emp_no,''), COALESCE(e.badge_code,''), COALESCE(cu.name,''),
		COALESCE(NULLIF(cu.name,''), COALESCE(e.name,''))
		FROM iam_user u
		LEFT JOIN hr_employee e ON e.id=u.employee_id
		LEFT JOIN crm_customer cu ON cu.id=u.customer_id
		WHERE u.id=?`, claims.UserID).
		Scan(&empID, &customerID, &empName, &empNo, &badgeCode, &customerName, &displayName)
	if displayName == "" {
		displayName = empName
	}
	api.OK(c, gin.H{
		"user": gin.H{
			"id":            claims.UserID,
			"login_name":    claims.LoginName,
			"user_type":     claims.UserType,
			"employee_id":   empID.Int64,
			"customer_id":   customerID.Int64,
			"customer_name": customerName,
			"name":          displayName,
			"employee_name": empName,
			"emp_no":        empNo,
			"badge_code":    badgeCode,
		},
		"client_type":    claims.ClientType,
		"roles":          roles,
		"permissions":    perms,
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
