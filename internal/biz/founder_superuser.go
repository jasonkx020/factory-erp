package biz

import (
	"database/sql"
	"log"

	"erp/internal/api"
	"erp/internal/security"

	"github.com/gin-gonic/gin"
)

// EnsureFounderSuperuser makes the first account (id=1 / admin / 工号 001) a full superuser.
func EnsureFounderSuperuser(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO iam_permission(code, name, domain, module, action, remark)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT (code) DO NOTHING`,
		security.WildcardPerm, "超级管理员全部权限", "*", "*", "*", "founder superuser")

	var roleID int64
	_ = db.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' AND COALESCE(is_deleted,0)=0 LIMIT 1`).Scan(&roleID)
	if roleID == 0 {
		_, _ = db.Exec(`INSERT INTO iam_role(code, name, data_scope_type, is_system, remark, status)
			VALUES('sys_admin','系统管理员','all',1,'系统全部权限','active')
			ON CONFLICT (code) DO UPDATE SET is_deleted=0, status='active', data_scope_type='all', name='系统管理员'`)
		_ = db.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' LIMIT 1`).Scan(&roleID)
	}
	if roleID > 0 {
		_, _ = db.Exec(`UPDATE iam_role SET data_scope_type='all', is_system=1, status='active', is_deleted=0, name='系统管理员' WHERE id=?`, roleID)
	}
	bindAllPermissionsToSysAdmin(db)

	uid := lookupFounderUserID(db)
	if uid <= 0 {
		return
	}
	_, _ = db.Exec(`UPDATE iam_user SET user_type='admin', status='active', is_deleted=0,
		freeze_reason=NULL, frozen_at=NULL, frozen_by=NULL WHERE id=?`, uid)
	if roleID > 0 {
		_, _ = db.Exec(`INSERT INTO iam_user_role(user_id, role_id) VALUES(?,?)
			ON CONFLICT (user_id, role_id) DO NOTHING`, uid, roleID)
		appendExtraRoleIDs(db, uid, []int64{roleID})
	}
	security.InvalidateUserRBAC(uid)
	log.Printf("founder superuser ensured (user_id=%d, all permissions)", uid)
}

func lookupFounderUserID(db *sql.DB) int64 {
	var uid int64
	_ = db.QueryRow(`SELECT id FROM iam_user WHERE id=1 AND COALESCE(is_deleted,0)=0 LIMIT 1`).Scan(&uid)
	if uid > 0 {
		return uid
	}
	_ = db.QueryRow(`SELECT id FROM iam_user WHERE lower(login_name)='admin' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&uid)
	if uid > 0 {
		return uid
	}
	rows, err := db.Query(`
		SELECT u.id
		FROM iam_user u
		LEFT JOIN hr_employee e ON e.id = u.employee_id
		LEFT JOIN hr_employee e2 ON e2.user_id = u.id
		WHERE COALESCE(u.is_deleted,0)=0
		ORDER BY u.id`)
	if err != nil {
		return 0
	}
	defer rows.Close()
	var first int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) != nil || id <= 0 {
			continue
		}
		if first == 0 {
			first = id
		}
		if security.IsFounderUser(db, id) {
			return id
		}
	}
	return first
}

func (s *Services) refuseIfFounderProtected(c *gin.Context, userID int64) bool {
	if userID <= 0 || s == nil || s.DB == nil {
		return false
	}
	if security.IsFounderUser(s.DB, userID) {
		api.FailJSON(c, "SUPERUSER_PROTECTED")
		return true
	}
	return false
}

func (s *Services) keepFounderSysAdmin(userID int64) {
	if userID <= 0 || !security.IsFounderUser(s.DB, userID) {
		return
	}
	var roleID int64
	_ = s.DB.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' AND COALESCE(is_deleted,0)=0 LIMIT 1`).Scan(&roleID)
	if roleID <= 0 {
		return
	}
	_, _ = s.DB.Exec(`INSERT INTO iam_user_role(user_id, role_id) VALUES(?,?)
		ON CONFLICT (user_id, role_id) DO NOTHING`, userID, roleID)
	appendExtraRoleIDs(s.DB, userID, []int64{roleID})
	_, _ = s.DB.Exec(`UPDATE iam_user SET user_type='admin', status='active', is_deleted=0 WHERE id=?`, userID)
}
