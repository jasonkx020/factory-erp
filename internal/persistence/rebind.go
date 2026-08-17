package persistence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/stdlib"
)

const driverName = "pgx_rebind"

var registerOnce sync.Once

func registerRebindDriver() {
	registerOnce.Do(func() {
		sql.Register(driverName, &rebindDriver{base: stdlib.GetDefaultDriver()})
	})
}

// RegisterRebindDriverForTest exposes driver registration for //go:build ignore tools.
func RegisterRebindDriverForTest() { registerRebindDriver() }

// Rebind converts ? placeholders to PostgreSQL $1, $2, ... form.
func Rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' && !inDouble {
			if inSingle && i+1 < len(query) && query[i+1] == '\'' {
				b.WriteByte(ch)
				b.WriteByte(query[i+1])
				i++
				continue
			}
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}
		if ch == '?' && !inSingle && !inDouble {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

type rebindDriver struct {
	base driver.Driver
}

func (d *rebindDriver) Open(name string) (driver.Conn, error) {
	c, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return wrapConn(c), nil
}

type rebindConn struct {
	driver.Conn
	execer   driver.ExecerContext
	queryer  driver.QueryerContext
	preparer driver.ConnPrepareContext
	pinger   driver.Pinger
	raw      driver.Conn
}

func wrapConn(c driver.Conn) driver.Conn {
	w := &rebindConn{Conn: c, raw: c}
	if ec, ok := c.(driver.ExecerContext); ok {
		w.execer = ec
	}
	if qc, ok := c.(driver.QueryerContext); ok {
		w.queryer = qc
	}
	if pc, ok := c.(driver.ConnPrepareContext); ok {
		w.preparer = pc
	}
	if p, ok := c.(driver.Pinger); ok {
		w.pinger = p
	}
	return w
}

func (c *rebindConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(Rebind(query))
}

func (c *rebindConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.preparer != nil {
		return c.preparer.PrepareContext(ctx, Rebind(query))
	}
	return c.Prepare(query)
}

func (c *rebindConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.execer == nil {
		return nil, fmt.Errorf("ExecContext not supported")
	}
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)
	// Auto RETURNING id so database/sql LastInsertId keeps working on PostgreSQL.
	if strings.HasPrefix(upper, "INSERT") && !strings.Contains(upper, "RETURNING") && c.queryer != nil {
		q := strings.TrimRight(trimmed, ";") + " RETURNING id"
		rows, err := c.queryer.QueryContext(ctx, Rebind(q), args)
		if err != nil {
			// Fallback: plain exec (tables without id column, or ON CONFLICT DO NOTHING with no row)
			res, err2 := c.execer.ExecContext(ctx, Rebind(query), args)
			if err2 != nil {
				return nil, err
			}
			return res, nil
		}
		defer rows.Close()
		vals := make([]driver.Value, len(rows.Columns()))
		if err := rows.Next(vals); err != nil {
			// no row returned (e.g. ON CONFLICT DO NOTHING) or end of rows
			return driver.RowsAffected(0), nil
		}
		var id int64
		switch v := vals[0].(type) {
		case int64:
			id = v
		case int32:
			id = int64(v)
		case []byte:
			id, _ = strconv.ParseInt(string(v), 10, 64)
		default:
			id, _ = strconv.ParseInt(fmt.Sprint(v), 10, 64)
		}
		_ = rows.Close()
		return insertResult{id: id}, nil
	}
	return c.execer.ExecContext(ctx, Rebind(query), args)
}

type insertResult struct {
	id int64
}

func (r insertResult) LastInsertId() (int64, error) { return r.id, nil }
func (r insertResult) RowsAffected() (int64, error) { return 1, nil }

func (c *rebindConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.queryer == nil {
		return nil, fmt.Errorf("QueryContext not supported")
	}
	return c.queryer.QueryContext(ctx, Rebind(query), args)
}

func (c *rebindConn) Ping(ctx context.Context) error {
	if c.pinger != nil {
		return c.pinger.Ping(ctx)
	}
	return nil
}

func (c *rebindConn) Begin() (driver.Tx, error) {
	return c.Conn.Begin()
}

func (c *rebindConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if bt, ok := c.raw.(driver.ConnBeginTx); ok {
		return bt.BeginTx(ctx, opts)
	}
	return c.Conn.Begin()
}

func (c *rebindConn) Close() error {
	return c.Conn.Close()
}

// Ensure driver.Conn interfaces are satisfied at compile time.
var (
	_ driver.Conn               = (*rebindConn)(nil)
	_ driver.ExecerContext      = (*rebindConn)(nil)
	_ driver.QueryerContext     = (*rebindConn)(nil)
	_ driver.ConnPrepareContext = (*rebindConn)(nil)
	_ driver.Pinger             = (*rebindConn)(nil)
)
