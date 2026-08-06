package biz

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

// systemModules 与前端 SYSTEM_ADMIN_MODULES 对齐
var systemModules = []string{
	"基础设置", "自定义打印", "自定义菜单", "自定义权限", "表格自定义", "公式设置",
	"销售设置", "生产设置", "物流信息管理", "审批流程设定", "人事调动", "登录控制",
	"批量改价", "批量核算工资", "单据审批", "单据锁定", "单据通知", "单据编辑", "单据删除",
	"事项提醒", "多条件检索", "账户冻结", "财审管控", "学堂管理", "知识库", "图纸管理",
	"文档管理", "员工日志", "操作日志", "数据修复", "业务闭环", "公告设置", "备忘录",
}

// resourceKey → 系统管理中文模块
var systemResourceModule = map[string]string{
	"system/settings":               "基础设置",
	"system/print-templates":        "自定义打印",
	"iam/menus":                     "自定义菜单",
	"iam/permissions":               "自定义权限",
	"system/table-customs":          "表格自定义",
	"system/formulas":               "公式设置",
	"system/sales-settings":         "销售设置",
	"system/production-settings":    "生产设置",
	"system/logistics/carriers":     "物流信息管理",
	"system/approval-flows":         "审批流程设定",
	"system/personnel-transfers":    "人事调动",
	"iam/login-policy":              "登录控制",
	"system/batch-price-jobs":       "批量改价",
	"system/batch-payroll-jobs":     "批量核算工资",
	"system/doc-approve-switches":   "单据审批",
	"system/doc-lock-rules":         "单据锁定",
	"system/notify-rules":           "单据通知",
	"system/doc-edit-rules":         "单据编辑",
	"system/doc-delete-rules":       "单据删除",
	"system/reminders":              "事项提醒",
	"system/search-configs":         "多条件检索",
	"iam/users":                     "账户冻结",
	"system/finance-audit-controls": "财审管控",
	"system/courses":                "学堂管理",
	"system/knowledge":              "知识库",
	"system/drawings":               "图纸管理",
	"system/documents":              "文档管理",
	"system/employee-journals":      "员工日志",
	"hr/employee-journals":          "员工日志",
	"system/operation-logs":         "操作日志",
	"system/data-repairs":           "数据修复",
	"system/announcements":          "公告设置",
	"system/memos":                  "备忘录",
}

func isSystemProtectedResource(resourceKey string) bool {
	if _, ok := systemResourceModule[resourceKey]; ok {
		return true
	}
	return strings.HasPrefix(resourceKey, "system/")
}

func claimsIsSysAdmin(roles, perms []string) bool {
	for _, r := range roles {
		if r == "sys_admin" || r == "系统管理员" {
			return true
		}
	}
	for _, p := range perms {
		if p == "*:*:*" {
			return true
		}
	}
	return false
}

func claimsHasCode(perms []string, codes ...string) bool {
	set := map[string]struct{}{}
	for _, p := range perms {
		set[p] = struct{}{}
	}
	for _, c := range codes {
		if _, ok := set[c]; ok {
			return true
		}
	}
	return false
}

// CheckSystemAPIPerm 对系统管理/IAM 敏感 API 做二次鉴权；非保护资源直接放行。
// 返回 false 表示已写入拒绝响应。
func CheckSystemAPIPerm(c *gin.Context, resourceKey, action, method string) bool {
	if !isSystemProtectedResource(resourceKey) {
		return true
	}
	claims := middleware.Claims(c)
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return false
	}
	if claimsIsSysAdmin(claims.Roles, claims.Permissions) {
		return true
	}

	module, ok := systemResourceModule[resourceKey]
	if !ok {
		c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
		return false
	}

	needEdit := method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" ||
		strings.HasPrefix(action, "action:") ||
		(action == "create" || action == "update" || action == "delete")
	// replace：GET 查看，PUT/POST 编辑
	if action == "replace" && (method == "GET" || method == "") {
		needEdit = false
	}
	if action == "list" || action == "get" {
		needEdit = false
	}

	viewCode := "系统管理:" + module + ":查看"
	editCode := "系统管理:" + module + ":编辑"
	if needEdit {
		if !claimsHasCode(claims.Permissions, editCode) {
			c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
			return false
		}
		return true
	}
	if !claimsHasCode(claims.Permissions, viewCode, editCode) {
		c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
		return false
	}
	return true
}

// EnsureSystemAdminPermissions 幂等植入系统管理 查看/编辑 权限并绑定 sys_admin。
func EnsureSystemAdminPermissions(db *sql.DB) {
	for _, mod := range systemModules {
		for _, act := range []string{"查看", "编辑"} {
			code := "系统管理:" + mod + ":" + act
			name := act + mod
			_, _ = db.Exec(`INSERT OR IGNORE INTO iam_permission(code, name, domain, module, action) VALUES(?,?,?,?,?)`,
				code, name, "系统管理", mod, act)
		}
	}
	var roleID int64
	if err := db.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' LIMIT 1`).Scan(&roleID); err != nil || roleID == 0 {
		return
	}
	rows, err := db.Query(`SELECT id FROM iam_permission WHERE domain='系统管理' AND COALESCE(is_deleted,0)=0`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		if rows.Scan(&pid) != nil {
			continue
		}
		_, _ = db.Exec(`INSERT OR IGNORE INTO iam_role_permission(role_id, permission_id) VALUES(?,?)`, roleID, pid)
	}
}
