package biz

import (
	"testing"
)

func TestEnsureJobTitleAndEmployeeFK(t *testing.T) {
	s := openIssueDB(t)
	ensureJobTitleTable(s.DB)
	ensureFactoryJobTitles(s.DB)

	id := ensureJobTitleRow(s.DB, "测试去皮工", "piece", "JT-TEST-PEEL", 99)
	if id <= 0 {
		t.Fatal("expected job title id")
	}

	empID, msg := s.createEmployeeFromBody(map[string]interface{}{
		"name":         "测试员工",
		"mobile":       "13800009999",
		"id_card_no":   "450103199001011234",
		"emp_type":     "piece",
		"job_title_id": id,
		"dept_id":      1,
	}, "active")
	if msg != "" {
		t.Fatalf("create employee: %s", msg)
	}

	emp := s.loadEmployeeMap(empID)
	if emp["job_title_id"] != id {
		t.Fatalf("job_title_id want %d got %v", id, emp["job_title_id"])
	}
	if emp["job_title_name"] != "测试去皮工" {
		t.Fatalf("job_title_name want 测试去皮工 got %v", emp["job_title_name"])
	}
}

func TestListJobTitlesFilterEmpType(t *testing.T) {
	s := openIssueDB(t)
	ensureFactoryJobTitles(s.DB)

	piece := s.listJobTitles("piece", "")
	hasPeel := false
	for _, row := range piece {
		if row["name"] == "去皮工" {
			hasPeel = true
		}
		if name, _ := row["name"].(string); name == "会计" {
			t.Fatalf("office title should not appear in piece filter: %v", row)
		}
	}
	if !hasPeel {
		t.Fatal("expected 去皮工 in piece list")
	}
}

func TestEnsureJobTitleRow_Idempotent(t *testing.T) {
	s := openIssueDB(t)
	ensureJobTitleTable(s.DB)
	id1 := ensureJobTitleRow(s.DB, "新岗位A", "office", "", 0)
	id2 := ensureJobTitleRow(s.DB, "新岗位A", "office", "", 0)
	if id1 <= 0 || id1 != id2 {
		t.Fatalf("ensure idempotent: %d vs %d", id1, id2)
	}
}
