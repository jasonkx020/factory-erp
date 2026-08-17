package biz

import (
	"database/sql"
	"log"
)

// EnsureCleanDevWageRates keeps one active wage rate per process (dev cleanliness).
// Safe to call repeatedly; also ensures partial unique index when possible.
func EnsureCleanDevWageRates(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY process_id ORDER BY id DESC) AS rn
  FROM pay_process_wage_rate
  WHERE status = 'active'
)
UPDATE pay_process_wage_rate r
SET status = 'inactive'
FROM ranked x
WHERE r.id = x.id AND x.rn > 1`)
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_pay_process_wage_rate_active_process
		ON pay_process_wage_rate (process_id) WHERE status = 'active'`); err != nil {
		log.Printf("dev wage-rate index: %v", err)
	}
}
