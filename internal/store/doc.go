package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"erp/internal/persistence/sqlutil"
)

// Doc is a generic persisted business document.
type Doc struct {
	ID          int64                  `json:"id"`
	ResourceKey string                 `json:"resource_key"`
	DocNo       string                 `json:"doc_no,omitempty"`
	Status      string                 `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

type Store struct {
	DB     *sql.DB
	Driver string
}

func (s *Store) Ensure() error {
	_, err := s.DB.Exec(`
CREATE TABLE IF NOT EXISTS erp_doc (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  resource_key TEXT NOT NULL,
  doc_no TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  payload TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  is_deleted INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_erp_doc_key ON erp_doc(resource_key, is_deleted);
CREATE INDEX IF NOT EXISTS idx_erp_doc_no ON erp_doc(doc_no);
`)
	return err
}

func (s *Store) List(resourceKey string, pageNum, pageSize int) ([]Doc, int, error) {
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(1) FROM erp_doc WHERE resource_key=? AND is_deleted=0`, resourceKey).Scan(&total); err != nil {
		return nil, 0, err
	}
	off := sqlutil.Offset(pageNum, pageSize)
	rows, err := s.DB.Query(`
SELECT id, resource_key, COALESCE(doc_no,''), status, payload, created_at, updated_at
FROM erp_doc WHERE resource_key=? AND is_deleted=0
ORDER BY id DESC LIMIT ? OFFSET ?`, resourceKey, pageSize, off)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := []Doc{}
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, d)
	}
	return list, total, nil
}

func (s *Store) Get(id int64) (*Doc, error) {
	row := s.DB.QueryRow(`
SELECT id, resource_key, COALESCE(doc_no,''), status, payload, created_at, updated_at
FROM erp_doc WHERE id=? AND is_deleted=0`, id)
	d, err := scanDoc(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) Create(resourceKey string, payload map[string]interface{}, status string) (*Doc, error) {
	if status == "" {
		status = "draft"
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	docNo, _ := payload["doc_no"].(string)
	if docNo == "" {
		docNo = fmt.Sprintf("%s-%d", strings.ReplaceAll(resourceKey, "/", "_"), time.Now().UnixNano()%1e12)
		payload["doc_no"] = docNo
	}
	b, _ := json.Marshal(payload)
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`INSERT INTO erp_doc(resource_key, doc_no, status, payload, created_at, updated_at) VALUES(?,?,?,?,?,?)`,
		resourceKey, docNo, status, string(b), now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(id)
}

func (s *Store) Update(id int64, payload map[string]interface{}, status string) (*Doc, error) {
	cur, err := s.Get(id)
	if err != nil || cur == nil {
		return nil, err
	}
	if payload == nil {
		payload = cur.Payload
	} else {
		// merge
		merged := map[string]interface{}{}
		for k, v := range cur.Payload {
			merged[k] = v
		}
		for k, v := range payload {
			merged[k] = v
		}
		payload = merged
	}
	if status == "" {
		status = cur.Status
	}
	b, _ := json.Marshal(payload)
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = s.DB.Exec(`UPDATE erp_doc SET payload=?, status=?, updated_at=? WHERE id=? AND is_deleted=0`, string(b), status, now, id)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Store) Delete(id int64) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.DB.Exec(`UPDATE erp_doc SET is_deleted=1, updated_at=? WHERE id=?`, now, id)
	return err
}

func (s *Store) SetStatus(id int64, status string) (*Doc, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.DB.Exec(`UPDATE erp_doc SET status=?, updated_at=? WHERE id=? AND is_deleted=0`, status, now, id)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDoc(sc rowScanner) (Doc, error) {
	var d Doc
	var payload string
	err := sc.Scan(&d.ID, &d.ResourceKey, &d.DocNo, &d.Status, &payload, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}
	_ = json.Unmarshal([]byte(payload), &d.Payload)
	if d.Payload == nil {
		d.Payload = map[string]interface{}{}
	}
	d.Payload["id"] = d.ID
	d.Payload["status"] = d.Status
	d.Payload["doc_no"] = d.DocNo
	d.Payload["created_at"] = d.CreatedAt
	d.Payload["updated_at"] = d.UpdatedAt
	return d, nil
}
