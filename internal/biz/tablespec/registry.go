package tablespec

type ColType int

const (
	TypeStr ColType = iota
	TypeInt
	TypeFloat
	TypeBool
)

type Col struct {
	Name string
	Type ColType
}

type LineSpec struct {
	Table   string
	FK      string
	OrderBy string
	Cols    []Col
}

type Spec struct {
	Table      string
	DocNo      string // auto-generate field name if empty on create
	Status     string
	SoftDelete bool
	Cols       []Col
	Lines      *LineSpec
	Actions    map[string]string // action name -> new status
}

// Registry maps OpenAPI resourceKey to real table specs.
var Registry = map[string]*Spec{
	"production/routings": {
		Table: "pd_routing", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"name", TypeStr}, {"product_id", TypeInt}, {"version_no", TypeStr}, {"status", TypeStr}},
		Lines: &LineSpec{
			Table: "pd_routing_step", FK: "routing_id", OrderBy: "seq_no",
			Cols: []Col{
				{"seq_no", TypeInt}, {"process_id", TypeInt}, {"step_code", TypeStr}, {"step_name", TypeStr},
				{"is_piecework", TypeBool}, {"is_inbound_checkpoint", TypeBool}, {"is_qc_required", TypeBool},
				{"auto_next", TypeBool}, {"auto_stock_in", TypeBool}, {"auto_stock_out", TypeBool},
				{"warehouse_id", TypeInt}, {"workshop_id", TypeInt},
			},
		},
		Actions: map[string]string{"activate": "active", "close": "closed"},
	},
	"production/workshops": {
		Table: "pd_workshop", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"org_id", TypeInt}, {"dept_id", TypeInt}, {"code", TypeStr}, {"name", TypeStr}, {"status", TypeStr}},
	},
	"production/boms": {
		Table: "pd_bom", DocNo: "code", Status: "status", SoftDelete: false,
		Cols: []Col{
			{"code", TypeStr}, {"product_id", TypeInt}, {"version_no", TypeStr}, {"name", TypeStr},
			{"is_auto_generated", TypeInt}, {"status", TypeStr}, {"remark", TypeStr},
		},
		Lines: &LineSpec{
			Table: "pd_bom_line", FK: "bom_id", OrderBy: "id",
			Cols: []Col{{"component_product_id", TypeInt}, {"qty", TypeFloat}, {"scrap_rate", TypeFloat}, {"remark", TypeStr}},
		},
		Actions: map[string]string{"activate": "active"},
	},
	"inventory/box-codes": {
		Table: "inv_box_code", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"code", TypeStr}, {"product_id", TypeInt}, {"warehouse_id", TypeInt}, {"batch_no", TypeStr},
			{"qty", TypeFloat}, {"weight", TypeFloat}, {"parent_box_id", TypeInt},
			{"current_process_id", TypeInt}, {"current_step_id", TypeInt}, {"task_id", TypeInt},
			{"work_order_id", TypeInt}, {"farmer_id", TypeInt}, {"trace_code", TypeStr},
			{"origin", TypeStr}, {"receive_date", TypeStr}, {"source_type", TypeStr}, {"status", TypeStr},
		},
	},
	"system/data-repairs": {
		Table: "sys_data_repair", DocNo: "doc_no", Status: "status", SoftDelete: false,
		Cols: []Col{
			{"doc_no", TypeStr}, {"target_type", TypeStr}, {"target_id", TypeInt}, {"action", TypeStr},
			{"reason", TypeStr}, {"status", TypeStr}, {"payload_json", TypeStr},
			{"applied_by", TypeInt}, {"applied_at", TypeStr}, {"created_by", TypeInt},
		},
		Actions: map[string]string{"apply": "applied"},
	},
	"production/flow-events": {
		Table: "pd_flow_event", Status: "status", SoftDelete: false,
		Cols: []Col{
			{"source_type", TypeStr}, {"source_id", TypeInt}, {"from_step_id", TypeInt}, {"to_step_id", TypeInt},
			{"trigger_action", TypeStr}, {"trace_id", TypeStr}, {"status", TypeStr}, {"error", TypeStr}, {"payload_json", TypeStr},
		},
	},
	"payroll/worker-profiles": {
		Table: "hr_employee", DocNo: "emp_no", Status: "status", SoftDelete: true,
		Cols: []Col{{"emp_no", TypeStr}, {"name", TypeStr}, {"org_id", TypeInt}, {"emp_type", TypeStr}, {"badge_code", TypeStr}, {"status", TypeStr}},
	},
	"system/print-templates": {
		Table: "sys_print_template", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"name", TypeStr}, {"doc_type", TypeStr}, {"content", TypeStr}, {"status", TypeStr}},
	},
	"system/formulas": {
		Table: "sys_formula", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"name", TypeStr}, {"scope", TypeStr}, {"expression", TypeStr}, {"remark", TypeStr}, {"status", TypeStr}},
	},
	"system/logistics/carriers": {
		Table: "sys_carrier", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"name", TypeStr}, {"contact", TypeStr}, {"phone", TypeStr}, {"remark", TypeStr}, {"status", TypeStr}},
	},
	"system/approval-flows": {
		Table: "sys_approval_flow", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"name", TypeStr}, {"doc_type", TypeStr}, {"status", TypeStr}},
		Lines: &LineSpec{
			Table: "sys_approval_flow_node", FK: "flow_id", OrderBy: "seq_no",
			Cols: []Col{{"seq_no", TypeInt}, {"node_name", TypeStr}, {"approver_role", TypeStr}, {"approver_user_id", TypeInt}, {"require_all", TypeBool}},
		},
	},
	"system/personnel-transfers": {
		Table: "sys_personnel_transfer", DocNo: "doc_no", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"doc_no", TypeStr}, {"employee_id", TypeInt}, {"from_dept_id", TypeInt}, {"to_dept_id", TypeInt},
			{"from_workshop_id", TypeInt}, {"to_workshop_id", TypeInt}, {"reason", TypeStr},
			{"status", TypeStr}, {"effective_date", TypeStr}, {"confirmed_at", TypeStr}, {"created_by", TypeInt},
		},
		Actions: map[string]string{"confirm": "confirmed"},
	},
	"system/batch-price-jobs": {
		Table: "sys_batch_price_job", DocNo: "doc_no", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"doc_no", TypeStr}, {"target_type", TypeStr}, {"adjust_type", TypeStr}, {"adjust_value", TypeFloat},
			{"scope_json", TypeStr}, {"status", TypeStr}, {"result_msg", TypeStr}, {"created_by", TypeInt}, {"applied_at", TypeStr},
		},
	},
	"system/batch-payroll-jobs": {
		Table: "sys_batch_payroll_job", DocNo: "doc_no", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"doc_no", TypeStr}, {"period_ym", TypeStr}, {"workshop_id", TypeInt}, {"status", TypeStr},
			{"result_msg", TypeStr}, {"created_by", TypeInt}, {"applied_at", TypeStr},
		},
	},
	"system/reminders": {
		Table: "sys_reminder", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"title", TypeStr}, {"content", TypeStr}, {"remind_at", TypeStr},
			{"target_user_id", TypeInt}, {"target_role", TypeStr}, {"status", TypeStr},
		},
	},
	"system/announcements": {
		Table: "sys_announcement", Status: "status", SoftDelete: true,
		Cols: []Col{{"title", TypeStr}, {"content", TypeStr}, {"status", TypeStr}, {"published_at", TypeStr}, {"created_by", TypeInt}},
		Actions: map[string]string{"publish": "published"},
	},
	"system/memos": {
		Table: "sys_memo", Status: "status", SoftDelete: true,
		Cols: []Col{{"title", TypeStr}, {"content", TypeStr}, {"owner_id", TypeInt}, {"status", TypeStr}},
	},
	"system/documents": {
		Table: "sys_document", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"title", TypeStr}, {"category", TypeStr}, {"content", TypeStr}, {"file_url", TypeStr}, {"status", TypeStr}},
	},
	"system/drawings": {
		Table: "sys_drawing", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"title", TypeStr}, {"product_id", TypeInt}, {"version_no", TypeStr}, {"file_url", TypeStr}, {"status", TypeStr}},
	},
	"system/knowledge": {
		Table: "sys_knowledge", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"title", TypeStr}, {"category", TypeStr}, {"content", TypeStr}, {"status", TypeStr}},
	},
	"system/courses": {
		Table: "sys_course", DocNo: "code", Status: "status", SoftDelete: true,
		Cols: []Col{{"code", TypeStr}, {"title", TypeStr}, {"category", TypeStr}, {"content", TypeStr}, {"duration_min", TypeInt}, {"status", TypeStr}},
	},
	"purchase/inbounds": {
		Table: "pur_purchase_inbound", DocNo: "doc_no", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"doc_no", TypeStr}, {"supplier_id", TypeInt}, {"warehouse_id", TypeInt}, {"status", TypeStr},
			{"biz_date", TypeStr}, {"plan_id", TypeInt}, {"remark", TypeStr},
		},
		Lines: &LineSpec{
			Table: "pur_purchase_inbound_line", FK: "inbound_id", OrderBy: "id",
			Cols: []Col{{"product_id", TypeInt}, {"qty", TypeFloat}, {"price", TypeFloat}, {"amount", TypeFloat}, {"batch_no", TypeStr}},
		},
		Actions: map[string]string{"post": "posted"},
	},
	"purchase/tasks": {
		Table: "pur_purchase_task", DocNo: "doc_no", Status: "status", SoftDelete: true,
		Cols: []Col{
			{"doc_no", TypeStr}, {"assignee_id", TypeInt}, {"product_id", TypeInt}, {"qty", TypeFloat},
			{"status", TypeStr}, {"due_date", TypeStr},
		},
		Actions: map[string]string{"assign": "assigned", "complete": "done"},
	},
}
