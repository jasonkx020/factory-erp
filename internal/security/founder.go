package security

import (
	"database/sql"
	"strings"
	"unicode"
)

const WildcardPerm = "*:*:*"

// IsFounderEmpNo reports whether an employee number is the reserved first account (001 / E0001).
func IsFounderEmpNo(empNo string) bool {
	s := strings.ToUpper(strings.TrimSpace(empNo))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	if strings.HasPrefix(s, "EMP") {
		s = strings.TrimPrefix(s, "EMP")
	} else if len(s) >= 2 && s[0] == 'E' && unicode.IsDigit(rune(s[1])) {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return strings.TrimLeft(s, "0") == "1"
}

func isFounderLogin(login string) bool {
	return strings.EqualFold(strings.TrimSpace(login), "admin")
}

// IsFounderUser is the reserved first superuser: iam_user.id=1, login admin, or emp_no 001/E0001.
func IsFounderUser(db *sql.DB, userID int64) bool {
	if userID <= 0 {
		return false
	}
	if userID == 1 {
		return true
	}
	if db == nil {
		return false
	}
	var login, empNo, empNo2 string
	err := db.QueryRow(`
		SELECT u.login_name, COALESCE(e.emp_no,''), COALESCE(e2.emp_no,'')
		FROM iam_user u
		LEFT JOIN hr_employee e ON e.id = u.employee_id
		LEFT JOIN hr_employee e2 ON e2.user_id = u.id
		WHERE u.id=?`, userID).Scan(&login, &empNo, &empNo2)
	if err != nil {
		return false
	}
	return isFounderLogin(login) || IsFounderEmpNo(empNo) || IsFounderEmpNo(empNo2)
}

func elevateFounder(roles, perms []string) (outRoles, outPerms []string) {
	outRoles = append([]string{}, roles...)
	outPerms = append([]string{}, perms...)
	hasRole := false
	for _, r := range outRoles {
		if r == "sys_admin" || r == "系统管理员" {
			hasRole = true
			break
		}
	}
	if !hasRole {
		outRoles = append(outRoles, "sys_admin")
	}
	hasStar := false
	for _, p := range outPerms {
		if p == WildcardPerm {
			hasStar = true
			break
		}
	}
	if !hasStar {
		outPerms = append(outPerms, WildcardPerm)
	}
	return outRoles, outPerms
}
