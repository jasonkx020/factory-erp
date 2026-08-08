package security

import (
	"database/sql"
	"sync"
	"time"
)

const defaultRBACCacheTTL = 30 * time.Second

type rbacEntry struct {
	roles []string
	perms []string
	at    time.Time
}

var rbacCache = struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[int64]rbacEntry
}{
	ttl: defaultRBACCacheTTL,
	m:   map[int64]rbacEntry{},
}

// LoadUserRolesPerms loads role codes and distinct permission codes for a user.
func LoadUserRolesPerms(db *sql.DB, userID int64) (roles []string, perms []string) {
	roles = []string{}
	perms = []string{}
	if db == nil || userID <= 0 {
		return roles, perms
	}
	rows, err := db.Query(`
		SELECT r.code FROM iam_user_role ur
		JOIN iam_role r ON r.id = ur.role_id
		WHERE ur.user_id = ?`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			if rows.Scan(&code) == nil && code != "" {
				roles = append(roles, code)
			}
		}
	}
	prows, err := db.Query(`
		SELECT DISTINCT p.code FROM iam_user_role ur
		JOIN iam_role_permission rp ON rp.role_id = ur.role_id
		JOIN iam_permission p ON p.id = rp.permission_id
		WHERE ur.user_id = ?`, userID)
	if err == nil {
		defer prows.Close()
		for prows.Next() {
			var code string
			if prows.Scan(&code) == nil && code != "" {
				perms = append(perms, code)
			}
		}
	}
	return roles, perms
}

// CachedUserRolesPerms returns roles/perms with a short in-process TTL cache.
func CachedUserRolesPerms(db *sql.DB, userID int64) (roles []string, perms []string) {
	if db == nil || userID <= 0 {
		return []string{}, []string{}
	}
	now := time.Now()
	rbacCache.mu.RLock()
	if e, ok := rbacCache.m[userID]; ok && now.Sub(e.at) < rbacCache.ttl {
		roles, perms = e.roles, e.perms
		rbacCache.mu.RUnlock()
		return append([]string(nil), roles...), append([]string(nil), perms...)
	}
	rbacCache.mu.RUnlock()

	roles, perms = LoadUserRolesPerms(db, userID)
	rbacCache.mu.Lock()
	rbacCache.m[userID] = rbacEntry{roles: roles, perms: perms, at: now}
	rbacCache.mu.Unlock()
	return roles, perms
}

// InvalidateUserRBAC drops cached roles/perms for one user.
func InvalidateUserRBAC(userID int64) {
	if userID <= 0 {
		return
	}
	rbacCache.mu.Lock()
	delete(rbacCache.m, userID)
	rbacCache.mu.Unlock()
}

// InvalidateAllRBAC clears the whole roles/perms cache (role permission edits).
func InvalidateAllRBAC() {
	rbacCache.mu.Lock()
	rbacCache.m = map[int64]rbacEntry{}
	rbacCache.mu.Unlock()
}

// HydrateClaimsRolesPerms overwrites Roles/Permissions from DB (via cache).
func HydrateClaimsRolesPerms(db *sql.DB, claims *Claims) {
	if claims == nil || claims.UserID <= 0 {
		return
	}
	roles, perms := CachedUserRolesPerms(db, claims.UserID)
	claims.Roles = roles
	claims.Permissions = perms
}

// SlimForToken returns a copy safe to embed in JWT (no permissions payload).
func SlimForToken(c Claims) Claims {
	c.Permissions = nil
	return c
}
