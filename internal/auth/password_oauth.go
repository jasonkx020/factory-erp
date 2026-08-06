package auth

import (
	"database/sql"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/security"
)

// EnsureAuthSchema creates oauth binding table.
func EnsureAuthSchema(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS iam_user_oauth (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  provider TEXT NOT NULL,
  open_id TEXT NOT NULL,
  union_id TEXT,
  bound_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(provider, open_id),
  UNIQUE(user_id, provider)
)`)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	if msg := validateNewPassword(h.DB, req.NewPassword); msg != "" {
		api.FailJSON(c, msg)
		return
	}
	var hash string
	err := h.DB.QueryRow(`SELECT password_hash FROM iam_user WHERE id=? AND COALESCE(is_deleted,0)=0`, claims.UserID).Scan(&hash)
	if err == sql.ErrNoRows {
		api.FailJSON(c, "USER_NOT_FOUND")
		return
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	if !security.CheckPassword(hash, req.OldPassword) {
		api.FailJSON(c, "OLD_PASSWORD_WRONG")
		return
	}
	nh, err := security.HashPassword(req.NewPassword)
	if err != nil {
		api.FailJSON(c, "HASH_ERROR")
		return
	}
	_, err = h.DB.Exec(`UPDATE iam_user SET password_hash=?, pwd_changed_at=datetime('now') WHERE id=?`, nh, claims.UserID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	_, _ = h.DB.Exec(`INSERT INTO iam_password_history(user_id, password_hash) VALUES(?,?)`, claims.UserID, nh)
	api.OK(c, gin.H{"changed": true})
}

func validateNewPassword(db *sql.DB, pwd string) string {
	minLen := 8
	needLetter, needDigit := true, true
	if db != nil {
		var ml, letter, digit int
		_ = db.QueryRow(`SELECT password_min_len, password_require_letter, password_require_digit FROM iam_login_policy ORDER BY id LIMIT 1`).
			Scan(&ml, &letter, &digit)
		if ml > 0 {
			minLen = ml
		}
		needLetter = letter != 0
		needDigit = digit != 0
	}
	if len(pwd) < minLen {
		return "PASSWORD_TOO_SHORT"
	}
	hasL, hasD := false, false
	for _, r := range pwd {
		if unicode.IsLetter(r) {
			hasL = true
		}
		if unicode.IsDigit(r) {
			hasD = true
		}
	}
	if needLetter && !hasL {
		return "PASSWORD_NEED_LETTER"
	}
	if needDigit && !hasD {
		return "PASSWORD_NEED_DIGIT"
	}
	return ""
}

func (h *Handler) ListOAuthBindings(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	EnsureAuthSchema(h.DB)
	rows, err := h.DB.Query(`SELECT provider, open_id, COALESCE(union_id,''), bound_at FROM iam_user_oauth WHERE user_id=?`, claims.UserID)
	list := []gin.H{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var provider, openID, unionID, boundAt string
			_ = rows.Scan(&provider, &openID, &unionID, &boundAt)
			list = append(list, gin.H{
				"provider": provider, "open_id": maskOpenID(openID), "union_id": unionID, "bound_at": boundAt, "bound": true,
			})
		}
	}
	// always expose wechat slot for UI
	boundWechat := false
	for _, x := range list {
		if x["provider"] == "wechat" {
			boundWechat = true
			break
		}
	}
	api.OK(c, gin.H{
		"list": list,
		"providers": []gin.H{
			{"provider": "wechat", "bound": boundWechat, "label": "微信"},
		},
	})
}

func maskOpenID(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-2:]
}

func (h *Handler) BindOAuth(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Provider == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if h.Cfg == nil || !h.Cfg.OAuth.Enabled {
		api.FailJSON(c, "OAUTH_NOT_CONFIGURED")
		return
	}
	api.FailJSON(c, "OAUTH_NOT_IMPLEMENTED")
}

func (h *Handler) UnbindOAuth(c *gin.Context) {
	claims := middleware.Claims(c)
	if claims == nil {
		c.JSON(401, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if provider == "" {
		api.FailJSON(c, "INVALID_REQUEST")
		return
	}
	EnsureAuthSchema(h.DB)
	res, err := h.DB.Exec(`DELETE FROM iam_user_oauth WHERE user_id=? AND provider=?`, claims.UserID, provider)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	n, _ := res.RowsAffected()
	api.OK(c, gin.H{"unbound": n > 0, "provider": provider, "at": time.Now().Format(time.RFC3339)})
}
