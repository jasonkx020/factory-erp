package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/security"
)

const CtxClaims = "claims"

func CORS(origins []string) gin.HandlerFunc {
	allowAll := len(origins) == 0 || (len(origins) == 1 && origins[0] == "*")
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			for _, o := range origins {
				if o == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-Id")
		c.Header("Access-Control-Expose-Headers", "X-Trace-Id")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func JWT(secret string, db *sql.DB, permitAll []string, skipPath ...func(string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, p := range permitAll {
			if path == p || strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}
		for _, skip := range skipPath {
			if skip != nil && skip(path) {
				c.Next()
				return
			}
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
			return
		}
		claims, err := security.ParseToken(secret, strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
			return
		}
		// Permissions are no longer embedded in JWT; load from DB (cached).
		security.HydrateClaimsRolesPerms(db, claims)
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

func RequirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(CtxClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
			return
		}
		claims := v.(*security.Claims)
		for _, p := range claims.Permissions {
			if p == code || p == "*:*:*" {
				c.Next()
				return
			}
		}
		// 系统管理员角色放行（骨架阶段）
		for _, r := range claims.Roles {
			if r == "sys_admin" || r == "系统管理员" {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
	}
}

func Claims(c *gin.Context) *security.Claims {
	v, ok := c.Get(CtxClaims)
	if !ok {
		return nil
	}
	return v.(*security.Claims)
}
