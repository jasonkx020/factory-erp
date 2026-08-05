package biz

import "strings"

// EmpTypeDef 员工类型字典项。各类型采集的档案字段一致，仅类别不同：
// 类别决定入职默认角色模板与工资/统计口径。
type EmpTypeDef struct {
	Code string
	Name string
	// RoleCode 入职未指定角色时使用的默认角色编码
	RoleCode string
}

// DefaultEmpType 未传员工类型时的默认值。
const DefaultEmpType = "piece"

// EmpTypes 全量员工类型，入职角色模板按此初始化。
// admin 仅供历史数据与模板初始化，不作为前端可选项。
var EmpTypes = []EmpTypeDef{
	{Code: "piece", Name: "计件工", RoleCode: "piece"},
	{Code: "temp", Name: "临时工", RoleCode: "piece"},
	{Code: "fixed", Name: "固定工", RoleCode: "fixed"},
	{Code: "office", Name: "职能/内勤", RoleCode: "hr"},
	{Code: "admin", Name: "系统管理", RoleCode: "sys_admin"},
}

// normalizeEmpType 兼容大小写与中文名（批量导入场景常见），返回标准 code。
func normalizeEmpType(v string) (string, bool) {
	raw := strings.TrimSpace(v)
	if raw == "" {
		return DefaultEmpType, true
	}
	lower := strings.ToLower(raw)
	for _, t := range EmpTypes {
		if t.Code == lower || t.Name == raw {
			return t.Code, true
		}
	}
	return "", false
}
