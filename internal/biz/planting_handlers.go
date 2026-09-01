package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handlePlanting(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/planting/dashboard"):
		return s.plantingDashboard(c)
	case strings.HasPrefix(openapiPath, "/api/v1/planting/plots"):
		return s.handlePlantPlots(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/planting/contracts"):
		return s.handlePlantContracts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/planting/field-logs"):
		return s.handlePlantFieldLogs(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/planting/harvest-plans"):
		if strings.Contains(openapiPath, "/confirm") && method == "POST" {
			return s.confirmHarvestPlan(c)
		}
		if strings.Contains(openapiPath, "/to-arrival") && method == "POST" {
			return s.harvestPlanToArrival(c)
		}
		return s.handlePlantHarvestPlans(c, method, action)
	}
	return false
}

func (s *Services) plantingDashboard(c *gin.Context) bool {
	today := time.Now().Format("2006-01-02")
	monthStart := time.Now().Format("2006-01") + "-01"
	kpis := []gin.H{
		{"key": "plots", "title": "地块数", "value": s.queryCount(`SELECT COUNT(1) FROM plant_plot WHERE COALESCE(is_deleted,0)=0 AND status='active'`)},
		{"key": "contracts", "title": "有效合同", "value": s.queryCount(`SELECT COUNT(1) FROM plant_contract WHERE COALESCE(is_deleted,0)=0 AND status='active'`)},
		{"key": "area_mu", "title": "种植面积(亩)", "value": s.queryFloat(`SELECT COALESCE(SUM(area_mu),0) FROM plant_plot WHERE COALESCE(is_deleted,0)=0 AND status='active'`)},
		{"key": "logs_month", "title": "本月田间作业", "value": s.queryCount(`SELECT COUNT(1) FROM plant_field_log WHERE COALESCE(is_deleted,0)=0 AND biz_date>=?`, monthStart)},
		{"key": "harvest_pending", "title": "待采收计划", "value": s.queryCount(`SELECT COUNT(1) FROM plant_harvest_plan WHERE COALESCE(is_deleted,0)=0 AND status IN ('draft','confirmed') AND plan_date>=?`, today)},
		{"key": "harvest_today", "title": "今日采收计划", "value": s.queryCount(`SELECT COUNT(1) FROM plant_harvest_plan WHERE COALESCE(is_deleted,0)=0 AND plan_date=?`, today)},
	}
	api.OK(c, gin.H{"title": "木薯种植总览", "as_of": time.Now().Format("2006-01-02 15:04:05"), "kpis": kpis, "list": kpis})
	return true
}

func (s *Services) handlePlantPlots(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listPlantPlots(c)
	case action == "create":
		return s.createPlantPlot(c)
	case action == "get":
		return s.getPlantPlot(c)
	case action == "update" || action == "replace":
		return s.updatePlantPlot(c)
	case action == "delete":
		return s.softDeletePlantPlot(c)
	}
	return false
}

func (s *Services) listPlantPlots(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	kw := strings.TrimSpace(c.Query("keyword"))
	farmerID := strings.TrimSpace(c.Query("farmer_id"))
	status := strings.TrimSpace(c.Query("status"))
	where := `WHERE COALESCE(p.is_deleted,0)=0`
	args := []interface{}{}
	if farmerID != "" {
		where += ` AND p.farmer_id=?`
		args = append(args, farmerID)
	}
	if status != "" {
		where += ` AND p.status=?`
		args = append(args, status)
	}
	if kw != "" {
		where += ` AND (p.code LIKE ? OR p.name LIKE ? OR p.location LIKE ? OR COALESCE(f.name,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM plant_plot p LEFT JOIN pur_farmer f ON f.id=p.farmer_id `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT p.id, p.code, p.name, p.farmer_id, COALESCE(f.name,''), p.area_mu,
		COALESCE(p.location,''), COALESCE(p.soil_type,''), COALESCE(p.irrigation_type,''), COALESCE(p.variety,''),
		p.status, COALESCE(p.remark,''), p.created_at
		FROM plant_plot p LEFT JOIN pur_farmer f ON f.id=p.farmer_id `+where+`
		ORDER BY p.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := scanPlantPlots(rows)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func scanPlantPlots(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
}) []gin.H {
	list := []gin.H{}
	for rows.Next() {
		var id, farmerID int64
		var code, name, fname, loc, soil, irr, variety, status, remark, created string
		var area float64
		_ = rows.Scan(&id, &code, &name, &farmerID, &fname, &area, &loc, &soil, &irr, &variety, &status, &remark, &created)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "farmer_id": farmerID, "farmer_name": fname,
			"area_mu": area, "location": loc, "soil_type": soil, "irrigation_type": irr,
			"variety": variety, "status": status, "remark": remark, "created_at": created,
		})
	}
	return list
}

func (s *Services) loadPlantPlot(id int64) gin.H {
	var farmerID int64
	var code, name, fname, loc, soil, irr, variety, status, remark, created string
	var area float64
	err := s.DB.QueryRow(`SELECT p.id, p.code, p.name, p.farmer_id, COALESCE(f.name,''), p.area_mu,
		COALESCE(p.location,''), COALESCE(p.soil_type,''), COALESCE(p.irrigation_type,''), COALESCE(p.variety,''),
		p.status, COALESCE(p.remark,''), p.created_at
		FROM plant_plot p LEFT JOIN pur_farmer f ON f.id=p.farmer_id
		WHERE p.id=? AND COALESCE(p.is_deleted,0)=0`, id).
		Scan(&id, &code, &name, &farmerID, &fname, &area, &loc, &soil, &irr, &variety, &status, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "code": code, "name": name, "farmer_id": farmerID, "farmer_name": fname,
		"area_mu": area, "location": loc, "soil_type": soil, "irrigation_type": irr,
		"variety": variety, "status": status, "remark": remark, "created_at": created,
	}
}

func (s *Services) createPlantPlot(c *gin.Context) bool {
	body := bindBody(c)
	name := strOr(body["name"])
	if name == "" {
		api.FailJSON(c, "NAME_REQUIRED")
		return true
	}
	farmerID, _ := asInt64(body["farmer_id"])
	code := strOr(body["code"])
	if code == "" {
		code = fmt.Sprintf("PLT%s", time.Now().Format("060102150405"))
	}
	area, _ := asFloat(body["area_mu"])
	res, err := s.DB.Exec(`INSERT INTO plant_plot(code, name, farmer_id, area_mu, location, soil_type, irrigation_type, variety, status, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		code, name, farmerID, area, strOr(body["location"]), strOr(body["soil_type"]),
		strOr(body["irrigation_type"]), strOrDef(body["variety"], "鲜木薯"),
		strOrDef(body["status"], "active"), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadPlantPlot(id))
	return true
}

func (s *Services) getPlantPlot(c *gin.Context) bool {
	m := s.loadPlantPlot(paramID(c))
	if m["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) updatePlantPlot(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE plant_plot SET
		name=COALESCE(NULLIF(?,''),name), farmer_id=COALESCE(NULLIF(?,0),farmer_id),
		area_mu=COALESCE(NULLIF(?,0),area_mu), location=COALESCE(?,location),
		soil_type=COALESCE(?,soil_type), irrigation_type=COALESCE(?,irrigation_type),
		variety=COALESCE(NULLIF(?,''),variety), status=COALESCE(NULLIF(?,''),status),
		remark=COALESCE(?,remark), updated_at=NOW()
		WHERE id=? AND COALESCE(is_deleted,0)=0`,
		strOr(body["name"]), nullInt64Or(body["farmer_id"]), nullFloatOr(body["area_mu"]),
		strOr(body["location"]), strOr(body["soil_type"]), strOr(body["irrigation_type"]),
		strOr(body["variety"]), strOr(body["status"]), strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadPlantPlot(id))
	return true
}

func (s *Services) softDeletePlantPlot(c *gin.Context) bool {
	_, _ = s.DB.Exec(`UPDATE plant_plot SET is_deleted=1, updated_at=NOW() WHERE id=?`, paramID(c))
	api.OK(c, gin.H{"ok": true})
	return true
}

func (s *Services) handlePlantContracts(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listPlantContracts(c)
	case action == "create":
		return s.createPlantContract(c)
	case action == "get":
		m := s.loadPlantContract(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		return s.updatePlantContract(c)
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE plant_contract SET is_deleted=1, updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"ok": true})
		return true
	}
	return false
}

func (s *Services) listPlantContracts(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	farmerID := strings.TrimSpace(c.Query("farmer_id"))
	status := strings.TrimSpace(c.Query("status"))
	where := `WHERE COALESCE(c.is_deleted,0)=0`
	args := []interface{}{}
	if farmerID != "" {
		where += ` AND c.farmer_id=?`
		args = append(args, farmerID)
	}
	if status != "" {
		where += ` AND c.status=?`
		args = append(args, status)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM plant_contract c `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT c.id, c.doc_no, c.farmer_id, COALESCE(f.name,''), c.plot_id, COALESCE(p.name,''),
		COALESCE(c.variety,''), c.area_mu, c.unit_price, c.start_date, COALESCE(c.end_date,''), c.status, COALESCE(c.remark,''), c.created_at
		FROM plant_contract c
		LEFT JOIN pur_farmer f ON f.id=c.farmer_id
		LEFT JOIN plant_plot p ON p.id=c.plot_id `+where+`
		ORDER BY c.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, farmerID, plotID int64
		var docNo, fname, plotName, variety, start, end, status, remark, created string
		var area, price float64
		_ = rows.Scan(&id, &docNo, &farmerID, &fname, &plotID, &plotName, &variety, &area, &price, &start, &end, &status, &remark, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": fname,
			"plot_id": plotID, "plot_name": plotName, "variety": variety, "area_mu": area,
			"unit_price": price, "start_date": start, "end_date": end, "status": status,
			"remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) loadPlantContract(id int64) gin.H {
	var farmerID, plotID int64
	var docNo, fname, plotName, variety, start, end, status, remark, created string
	var area, price float64
	err := s.DB.QueryRow(`SELECT c.id, c.doc_no, c.farmer_id, COALESCE(f.name,''), c.plot_id, COALESCE(p.name,''),
		COALESCE(c.variety,''), c.area_mu, c.unit_price, c.start_date, COALESCE(c.end_date,''), c.status, COALESCE(c.remark,''), c.created_at
		FROM plant_contract c
		LEFT JOIN pur_farmer f ON f.id=c.farmer_id
		LEFT JOIN plant_plot p ON p.id=c.plot_id
		WHERE c.id=? AND COALESCE(c.is_deleted,0)=0`, id).
		Scan(&id, &docNo, &farmerID, &fname, &plotID, &plotName, &variety, &area, &price, &start, &end, &status, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": fname,
		"plot_id": plotID, "plot_name": plotName, "variety": variety, "area_mu": area,
		"unit_price": price, "start_date": start, "end_date": end, "status": status,
		"remark": remark, "created_at": created,
	}
}

func (s *Services) createPlantContract(c *gin.Context) bool {
	body := bindBody(c)
	farmerID, _ := asInt64(body["farmer_id"])
	if farmerID <= 0 {
		api.FailJSON(c, "FARMER_REQUIRED")
		return true
	}
	plotID, _ := asInt64(body["plot_id"])
	start := strOrDef(body["start_date"], time.Now().Format("2006-01-02"))
	docNo := fmt.Sprintf("PC%s", time.Now().Format("20060102150405"))
	area, _ := asFloat(body["area_mu"])
	price, _ := asFloat(body["unit_price"])
	res, err := s.DB.Exec(`INSERT INTO plant_contract(doc_no, farmer_id, plot_id, variety, area_mu, unit_price, start_date, end_date, status, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		docNo, farmerID, plotID, strOrDef(body["variety"], "鲜木薯"), area, price, start,
		strOr(body["end_date"]), strOrDef(body["status"], "active"), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadPlantContract(id))
	return true
}

func (s *Services) updatePlantContract(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE plant_contract SET
		farmer_id=COALESCE(NULLIF(?,0),farmer_id), plot_id=COALESCE(NULLIF(?,0),plot_id),
		variety=COALESCE(NULLIF(?,''),variety), area_mu=COALESCE(NULLIF(?,0),area_mu),
		unit_price=COALESCE(NULLIF(?,0),unit_price), start_date=COALESCE(NULLIF(?,''),start_date),
		end_date=COALESCE(?,end_date), status=COALESCE(NULLIF(?,''),status),
		remark=COALESCE(?,remark), updated_at=NOW()
		WHERE id=? AND COALESCE(is_deleted,0)=0`,
		nullInt64Or(body["farmer_id"]), nullInt64Or(body["plot_id"]), strOr(body["variety"]),
		nullFloatOr(body["area_mu"]), nullFloatOr(body["unit_price"]), strOr(body["start_date"]),
		strOr(body["end_date"]), strOr(body["status"]), strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadPlantContract(id))
	return true
}

func (s *Services) handlePlantFieldLogs(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listPlantFieldLogs(c)
	case action == "create":
		return s.createPlantFieldLog(c)
	case action == "get":
		m := s.loadPlantFieldLog(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE plant_field_log SET is_deleted=1 WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"ok": true})
		return true
	}
	return false
}

func (s *Services) listPlantFieldLogs(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	plotID := strings.TrimSpace(c.Query("plot_id"))
	logType := strings.TrimSpace(c.Query("log_type"))
	where := `WHERE COALESCE(l.is_deleted,0)=0`
	args := []interface{}{}
	if plotID != "" {
		where += ` AND l.plot_id=?`
		args = append(args, plotID)
	}
	if logType != "" {
		where += ` AND l.log_type=?`
		args = append(args, logType)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM plant_field_log l `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT l.id, l.doc_no, l.plot_id, COALESCE(p.name,''), l.farmer_id, COALESCE(f.name,''),
		l.log_type, l.biz_date, COALESCE(l.operator_name,''), COALESCE(l.content,''), l.qty, COALESCE(l.unit,''), COALESCE(l.remark,''), l.created_at
		FROM plant_field_log l
		LEFT JOIN plant_plot p ON p.id=l.plot_id
		LEFT JOIN pur_farmer f ON f.id=l.farmer_id `+where+`
		ORDER BY l.biz_date DESC, l.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, plotID, farmerID int64
		var docNo, plotName, fname, logType, bizDate, operator, content, unit, remark, created string
		var qty float64
		_ = rows.Scan(&id, &docNo, &plotID, &plotName, &farmerID, &fname, &logType, &bizDate, &operator, &content, &qty, &unit, &remark, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "plot_id": plotID, "plot_name": plotName,
			"farmer_id": farmerID, "farmer_name": fname, "log_type": logType, "biz_date": bizDate,
			"operator_name": operator, "content": content, "qty": qty, "unit": unit,
			"remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) loadPlantFieldLog(id int64) gin.H {
	var plotID, farmerID int64
	var docNo, plotName, fname, logType, bizDate, operator, content, unit, remark, created string
	var qty float64
	err := s.DB.QueryRow(`SELECT l.id, l.doc_no, l.plot_id, COALESCE(p.name,''), l.farmer_id, COALESCE(f.name,''),
		l.log_type, l.biz_date, COALESCE(l.operator_name,''), COALESCE(l.content,''), l.qty, COALESCE(l.unit,''), COALESCE(l.remark,''), l.created_at
		FROM plant_field_log l
		LEFT JOIN plant_plot p ON p.id=l.plot_id
		LEFT JOIN pur_farmer f ON f.id=l.farmer_id
		WHERE l.id=? AND COALESCE(l.is_deleted,0)=0`, id).
		Scan(&id, &docNo, &plotID, &plotName, &farmerID, &fname, &logType, &bizDate, &operator, &content, &qty, &unit, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "plot_id": plotID, "plot_name": plotName,
		"farmer_id": farmerID, "farmer_name": fname, "log_type": logType, "biz_date": bizDate,
		"operator_name": operator, "content": content, "qty": qty, "unit": unit,
		"remark": remark, "created_at": created,
	}
}

func (s *Services) createPlantFieldLog(c *gin.Context) bool {
	body := bindBody(c)
	plotID, _ := asInt64(body["plot_id"])
	if plotID <= 0 {
		api.FailJSON(c, "PLOT_REQUIRED")
		return true
	}
	plot := s.loadPlantPlot(plotID)
	farmerID, _ := asInt64(body["farmer_id"])
	if farmerID <= 0 {
		farmerID, _ = asInt64(plot["farmer_id"])
	}
	docNo := fmt.Sprintf("FL%s", time.Now().Format("20060102150405"))
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	qty, _ := asFloat(body["qty"])
	res, err := s.DB.Exec(`INSERT INTO plant_field_log(doc_no, plot_id, farmer_id, log_type, biz_date, operator_name, content, qty, unit, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		docNo, plotID, farmerID, strOrDef(body["log_type"], "other"), bizDate,
		strOr(body["operator_name"]), strOr(body["content"]), qty, strOr(body["unit"]), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadPlantFieldLog(id))
	return true
}

func (s *Services) handlePlantHarvestPlans(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listPlantHarvestPlans(c)
	case action == "create":
		return s.createPlantHarvestPlan(c)
	case action == "get":
		m := s.loadPlantHarvestPlan(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		return s.updatePlantHarvestPlan(c)
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE plant_harvest_plan SET is_deleted=1, updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"ok": true})
		return true
	}
	return false
}

func (s *Services) listPlantHarvestPlans(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	status := strings.TrimSpace(c.Query("status"))
	farmerID := strings.TrimSpace(c.Query("farmer_id"))
	where := `WHERE COALESCE(h.is_deleted,0)=0`
	args := []interface{}{}
	if status != "" {
		where += ` AND h.status=?`
		args = append(args, status)
	}
	if farmerID != "" {
		where += ` AND h.farmer_id=?`
		args = append(args, farmerID)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM plant_harvest_plan h `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT h.id, h.doc_no, h.plot_id, COALESCE(p.name,''), h.farmer_id, COALESCE(f.name,''),
		COALESCE(h.variety,''), h.plan_date, h.estimate_weight, h.status, h.arrival_id, COALESCE(h.remark,''), h.created_at
		FROM plant_harvest_plan h
		LEFT JOIN plant_plot p ON p.id=h.plot_id
		LEFT JOIN pur_farmer f ON f.id=h.farmer_id `+where+`
		ORDER BY h.plan_date DESC, h.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, plotID, farmerID, arrivalID int64
		var docNo, plotName, fname, variety, planDate, status, remark, created string
		var est float64
		_ = rows.Scan(&id, &docNo, &plotID, &plotName, &farmerID, &fname, &variety, &planDate, &est, &status, &arrivalID, &remark, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "plot_id": plotID, "plot_name": plotName,
			"farmer_id": farmerID, "farmer_name": fname, "variety": variety,
			"plan_date": planDate, "estimate_weight": est, "status": status,
			"arrival_id": arrivalID, "remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) loadPlantHarvestPlan(id int64) gin.H {
	var plotID, farmerID, arrivalID int64
	var docNo, plotName, fname, variety, planDate, status, remark, created string
	var est float64
	err := s.DB.QueryRow(`SELECT h.id, h.doc_no, h.plot_id, COALESCE(p.name,''), h.farmer_id, COALESCE(f.name,''),
		COALESCE(h.variety,''), h.plan_date, h.estimate_weight, h.status, h.arrival_id, COALESCE(h.remark,''), h.created_at
		FROM plant_harvest_plan h
		LEFT JOIN plant_plot p ON p.id=h.plot_id
		LEFT JOIN pur_farmer f ON f.id=h.farmer_id
		WHERE h.id=? AND COALESCE(h.is_deleted,0)=0`, id).
		Scan(&id, &docNo, &plotID, &plotName, &farmerID, &fname, &variety, &planDate, &est, &status, &arrivalID, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "plot_id": plotID, "plot_name": plotName,
		"farmer_id": farmerID, "farmer_name": fname, "variety": variety,
		"plan_date": planDate, "estimate_weight": est, "status": status,
		"arrival_id": arrivalID, "remark": remark, "created_at": created,
	}
}

func (s *Services) createPlantHarvestPlan(c *gin.Context) bool {
	body := bindBody(c)
	plotID, _ := asInt64(body["plot_id"])
	farmerID, _ := asInt64(body["farmer_id"])
	if plotID <= 0 || farmerID <= 0 {
		api.FailJSON(c, "PLOT_AND_FARMER_REQUIRED")
		return true
	}
	docNo := fmt.Sprintf("HP%s", time.Now().Format("20060102150405"))
	planDate := strOrDef(body["plan_date"], time.Now().Format("2006-01-02"))
	est, _ := asFloat(body["estimate_weight"])
	res, err := s.DB.Exec(`INSERT INTO plant_harvest_plan(doc_no, plot_id, farmer_id, variety, plan_date, estimate_weight, status, remark)
		VALUES(?,?,?,?,?,?,?,?)`,
		docNo, plotID, farmerID, strOrDef(body["variety"], "鲜木薯"), planDate, est,
		strOrDef(body["status"], "draft"), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadPlantHarvestPlan(id))
	return true
}

func (s *Services) updatePlantHarvestPlan(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE plant_harvest_plan SET
		plot_id=COALESCE(NULLIF(?,0),plot_id), farmer_id=COALESCE(NULLIF(?,0),farmer_id),
		variety=COALESCE(NULLIF(?,''),variety), plan_date=COALESCE(NULLIF(?,''),plan_date),
		estimate_weight=COALESCE(NULLIF(?,0),estimate_weight), status=COALESCE(NULLIF(?,''),status),
		remark=COALESCE(?,remark), updated_at=NOW()
		WHERE id=? AND COALESCE(is_deleted,0)=0`,
		nullInt64Or(body["plot_id"]), nullInt64Or(body["farmer_id"]), strOr(body["variety"]),
		strOr(body["plan_date"]), nullFloatOr(body["estimate_weight"]), strOr(body["status"]),
		strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadPlantHarvestPlan(id))
	return true
}

func (s *Services) confirmHarvestPlan(c *gin.Context) bool {
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE plant_harvest_plan SET status='confirmed', updated_at=NOW()
		WHERE id=? AND COALESCE(is_deleted,0)=0 AND status='draft'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadPlantHarvestPlan(id))
	return true
}

func (s *Services) harvestPlanToArrival(c *gin.Context) bool {
	id := paramID(c)
	plan := s.loadPlantHarvestPlan(id)
	if plan["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if arrivalID, _ := asInt64(plan["arrival_id"]); arrivalID > 0 {
		api.OK(c, gin.H{"arrival_id": arrivalID, "plan": plan})
		return true
	}
	farmerID, _ := asInt64(plan["farmer_id"])
	est, _ := asFloat(plan["estimate_weight"])
	variety := strOrDef(plan["variety"], "鲜木薯")
	plot := s.loadPlantPlot(asInt64Must(plan["plot_id"]))
	origin := strOr(plot["location"])
	docNo := fmt.Sprintf("AR%s", time.Now().Format("20060102150405"))
	bizDate := strOrDef(plan["plan_date"], time.Now().Format("2006-01-02"))
	res, err := s.DB.Exec(`INSERT INTO pur_inbound_arrival(doc_no, farmer_id, origin, variety, estimate_weight, source_type, channel, status, biz_date, remark)
		VALUES(?,?,?,?,?,'planting','internal','qc_pending',?,?)`,
		docNo, farmerID, origin, variety, est, bizDate, fmt.Sprintf("采收计划 %s 生成", strOr(plan["doc_no"])))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	arrivalID, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`UPDATE plant_harvest_plan SET arrival_id=?, status='confirmed', updated_at=NOW() WHERE id=?`, arrivalID, id)
	plan = s.loadPlantHarvestPlan(id)
	api.OK(c, gin.H{"arrival_id": arrivalID, "plan": plan, "arrival": s.loadArrival(arrivalID)})
	return true
}

func asInt64Must(v interface{}) int64 {
	n, _ := asInt64(v)
	return n
}

func nullFloatOr(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	f, ok := asFloat(v)
	if !ok && f == 0 {
		s := fmt.Sprint(v)
		if s == "" || s == "0" {
			return nil
		}
	}
	return f
}
