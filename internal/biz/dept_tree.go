package biz

import (
	"fmt"
	"sort"

	"github.com/gin-gonic/gin"
)

const maxDeptLevel = 3

type deptFlat struct {
	ID          int64
	OrgID       int64
	ParentID    int64
	Code        string
	Name        string
	Status      string
	DeptType    string
	MemberCount int64
}

func (s *Services) loadDeptFlatRows() ([]deptFlat, error) {
	rows, err := s.DB.Query(`SELECT d.id, COALESCE(d.org_id,0), COALESCE(d.parent_id,0), COALESCE(d.code,''), COALESCE(d.name,''),
		COALESCE(d.status,'active'), COALESCE(d.dept_type,'normal'),
		(SELECT COUNT(1) FROM hr_employee_department ed
		 JOIN hr_employee e ON e.id=ed.employee_id
		 WHERE ed.dept_id=d.id AND COALESCE(e.is_deleted,0)=0 AND COALESCE(e.status,'')<>'left')
		FROM sys_department d WHERE COALESCE(d.is_deleted,0)=0 ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []deptFlat{}
	for rows.Next() {
		var d deptFlat
		if err := rows.Scan(&d.ID, &d.OrgID, &d.ParentID, &d.Code, &d.Name, &d.Status, &d.DeptType, &d.MemberCount); err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func deptParentMap(rows []deptFlat) map[int64]int64 {
	m := make(map[int64]int64, len(rows))
	for _, d := range rows {
		m[d.ID] = d.ParentID
	}
	return m
}

func deptChildrenMap(rows []deptFlat) map[int64][]int64 {
	m := make(map[int64][]int64)
	for _, d := range rows {
		if d.ParentID > 0 {
			m[d.ParentID] = append(m[d.ParentID], d.ID)
		}
	}
	return m
}

func calcDeptLevel(id int64, parentByID map[int64]int64) int {
	level := 1
	cur := id
	for step := 0; step < maxDeptLevel+2; step++ {
		p := parentByID[cur]
		if p <= 0 {
			break
		}
		level++
		cur = p
	}
	return level
}

func (s *Services) getDeptParentID(deptID int64) int64 {
	var pid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(parent_id,0) FROM sys_department WHERE id=? AND COALESCE(is_deleted,0)=0`, deptID).Scan(&pid)
	return pid
}

func (s *Services) getDeptDescendantIDs(deptID int64) []int64 {
	rows, err := s.loadDeptFlatRows()
	if err != nil {
		return nil
	}
	children := deptChildrenMap(rows)
	out := []int64{}
	var walk func(id int64)
	walk = func(id int64) {
		for _, cid := range children[id] {
			out = append(out, cid)
			walk(cid)
		}
	}
	walk(deptID)
	return out
}

func (s *Services) validateDeptParent(deptID, parentID int64) error {
	if parentID <= 0 {
		return nil
	}
	if deptID > 0 && parentID == deptID {
		return fmt.Errorf("PARENT_INVALID")
	}
	var exists int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sys_department WHERE id=? AND COALESCE(is_deleted,0)=0`, parentID).Scan(&exists)
	if exists == 0 {
		return fmt.Errorf("PARENT_NOT_FOUND")
	}
	if s.getDeptType(parentID) == deptTypeWorkshop {
		return fmt.Errorf("WORKSHOP_NO_CHILDREN")
	}
	rows, err := s.loadDeptFlatRows()
	if err != nil {
		return err
	}
	parentByID := deptParentMap(rows)
	if calcDeptLevel(parentID, parentByID) >= maxDeptLevel {
		return fmt.Errorf("MAX_DEPTH_EXCEEDED")
	}
	if deptID > 0 {
		for _, desc := range s.getDeptDescendantIDs(deptID) {
			if desc == parentID {
				return fmt.Errorf("PARENT_IS_DESCENDANT")
			}
		}
	}
	return nil
}

func (s *Services) getDeptEffectiveRoleIDs(deptID int64) ([]int64, error) {
	if deptID <= 0 {
		return nil, nil
	}
	targets := append([]int64{deptID}, s.getDeptDescendantIDs(deptID)...)
	seen := make(map[int64]struct{})
	out := []int64{}
	for _, id := range targets {
		direct, err := s.getDeptRoleIDs(id)
		if err != nil {
			return nil, err
		}
		for _, rid := range direct {
			if _, ok := seen[rid]; ok {
				continue
			}
			seen[rid] = struct{}{}
			out = append(out, rid)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func subtractRoleIDs(all, direct []int64) []int64 {
	directSet := make(map[int64]struct{}, len(direct))
	for _, rid := range direct {
		directSet[rid] = struct{}{}
	}
	out := []int64{}
	for _, rid := range all {
		if _, ok := directSet[rid]; ok {
			continue
		}
		out = append(out, rid)
	}
	return out
}

func deptLevelLabel(level int) string {
	switch level {
	case 1:
		return "一级"
	case 2:
		return "二级"
	case 3:
		return "三级"
	default:
		return fmt.Sprintf("%d级", level)
	}
}

func (s *Services) deptPathNames(deptID int64) string {
	if deptID <= 0 {
		return ""
	}
	rows, err := s.loadDeptFlatRows()
	if err != nil {
		return ""
	}
	nameByID := make(map[int64]string, len(rows))
	parentByID := deptParentMap(rows)
	for _, d := range rows {
		nameByID[d.ID] = d.Name
	}
	parts := []string{}
	cur := deptID
	for step := 0; step < maxDeptLevel+1; step++ {
		name := nameByID[cur]
		if name == "" {
			break
		}
		parts = append([]string{name}, parts...)
		p := parentByID[cur]
		if p <= 0 {
			break
		}
		cur = p
	}
	return joinStrings(parts, " / ")
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

func buildDeptTreeNodes(flat []gin.H) []gin.H {
	byParent := make(map[int64][]gin.H)
	for _, item := range flat {
		pid, _ := asInt64(item["parent_id"])
		byParent[pid] = append(byParent[pid], item)
	}
	for _, items := range byParent {
		sort.Slice(items, func(i, j int) bool {
			a, _ := asInt64(items[i]["id"])
			b, _ := asInt64(items[j]["id"])
			return a < b
		})
	}
	var attach func(parentID int64) []gin.H
	attach = func(parentID int64) []gin.H {
		items := byParent[parentID]
		for i := range items {
			id, _ := asInt64(items[i]["id"])
			children := attach(id)
			if len(children) > 0 {
				items[i]["children"] = children
			}
		}
		return items
	}
	return attach(0)
}

func (s *Services) packDeptDetail(id int64) (gin.H, error) {
	var orgID, parentID int64
	var code, name, status, deptType string
	err := s.DB.QueryRow(`SELECT COALESCE(org_id,0), COALESCE(parent_id,0), COALESCE(code,''), COALESCE(name,''), COALESCE(status,'active'), COALESCE(dept_type,'normal')
		FROM sys_department WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&orgID, &parentID, &code, &name, &status, &deptType)
	if err != nil {
		return nil, err
	}
	rows, _ := s.loadDeptFlatRows()
	parentByID := deptParentMap(rows)
	level := calcDeptLevel(id, parentByID)
	parentName := ""
	if parentID > 0 {
		for _, d := range rows {
			if d.ID == parentID {
				parentName = d.Name
				break
			}
		}
	}
	directRoleIDs, _ := s.getDeptRoleIDs(id)
	effectiveRoleIDs, _ := s.getDeptEffectiveRoleIDs(id)
	inheritedRoleIDs := subtractRoleIDs(effectiveRoleIDs, directRoleIDs)
	members, _ := s.listDeptMembers(id)
	childCount := len(s.getDeptDescendantIDs(id))
	out := gin.H{
		"id": id, "org_id": orgID, "parent_id": parentID, "parent_name": parentName,
		"code": code, "name": name, "status": status, "dept_type": deptType,
		"level": level, "level_label": deptLevelLabel(level),
		"path": s.deptPathNames(id), "child_count": childCount,
		"member_count": len(members), "members": members,
		"role_ids":           directRoleIDs,
		"base_roles":         s.loadRoleDetailsByIDs(directRoleIDs),
		"effective_role_ids": effectiveRoleIDs,
		"effective_roles":    s.loadRoleDetailsByIDs(effectiveRoleIDs),
		"inherited_roles":    s.loadRoleDetailsByIDs(inheritedRoleIDs),
	}
	if deptType == deptTypeWorkshop {
		out["teams"] = s.listWorkTeamsByDept(id)
	}
	return out, nil
}

func (s *Services) syncDeptHierarchyRoleImpact(deptID int64, extraParentIDs ...int64) {
	seen := map[int64]struct{}{}
	var rebuildChain func(id int64)
	rebuildChain = func(id int64) {
		for id > 0 {
			if _, ok := seen[id]; ok {
				break
			}
			seen[id] = struct{}{}
			s.syncDeptBaseRolesForAllMembers(id)
			id = s.getDeptParentID(id)
		}
	}
	rebuildChain(deptID)
	for _, pid := range extraParentIDs {
		rebuildChain(pid)
	}
}
