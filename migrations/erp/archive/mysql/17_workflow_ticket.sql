-- 协作工单（分类派发 + 动态表单）
CREATE TABLE IF NOT EXISTS wf_ticket_category (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  remark TEXT,
  form_schema_json TEXT,
  biz_hint TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS wf_ticket_category_handler (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category_id INTEGER NOT NULL,
  handler_type TEXT NOT NULL,
  handler_ref INTEGER NOT NULL,
  UNIQUE(category_id, handler_type, handler_ref)
);

CREATE TABLE IF NOT EXISTS wf_ticket (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  category_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  applicant_user_id INTEGER NOT NULL,
  current_assignee_user_id INTEGER,
  biz_type TEXT,
  biz_id INTEGER,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  closed_at TEXT
);

CREATE TABLE IF NOT EXISTS wf_ticket_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
