package apigen

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/biz"
	"erp/internal/persistence/sqlutil"
	"erp/internal/store"
)

// Handler is implemented by Engine.
type Handler interface {
	Dispatch(method, openapiPath, ginPath, resourceKey, action string) gin.HandlerFunc
}

type Engine struct {
	DB     *sql.DB
	Driver string
	Store  *store.Store
	Biz    *biz.Services
}

func NewEngine(db *sql.DB, driver string) *Engine {
	st := &store.Store{DB: db, Driver: driver}
	_ = st.Ensure()
	return &Engine{
		DB:     db,
		Driver: driver,
		Store:  st,
		Biz:    biz.New(db, driver, st),
	}
}

func (e *Engine) Dispatch(method, openapiPath, ginPath, resourceKey, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prefer dedicated business handlers when available.
		if e.Biz.Handle(c, method, openapiPath, resourceKey, action) {
			return
		}
		e.handleGeneric(c, method, resourceKey, action)
	}
}

func (e *Engine) handleGeneric(c *gin.Context, method, resourceKey, action string) {
	pageNum, pageSize := sqlutil.Page(c)
	switch {
	case action == "list" || action == "replace" && method == "GET":
		list, total, err := e.Store.List(resourceKey, pageNum, pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		out := make([]map[string]interface{}, 0, len(list))
		for _, d := range list {
			out = append(out, d.Payload)
		}
		api.PageOK(c, out, total, pageNum, pageSize)

	case action == "create":
		var body map[string]interface{}
		_ = c.ShouldBindJSON(&body)
		if body == nil {
			body = map[string]interface{}{}
		}
		status, _ := body["status"].(string)
		d, err := e.Store.Create(resourceKey, body, status)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		api.OK(c, d.Payload)

	case action == "get":
		id := paramID(c)
		d, err := e.Store.Get(id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		if d == nil || d.ResourceKey != resourceKey && !strings.HasPrefix(d.ResourceKey, strings.Split(resourceKey, "/")[0]) {
			// resource_key on get path may be parent without id — accept if id exists
			if d == nil {
				api.FailJSON(c, "NOT_FOUND")
				return
			}
		}
		api.OK(c, d.Payload)

	case action == "update" || action == "replace":
		id := paramID(c)
		var body map[string]interface{}
		_ = c.ShouldBindJSON(&body)
		status := ""
		if body != nil {
			if s, ok := body["status"].(string); ok {
				status = s
			}
		}
		d, err := e.Store.Update(id, body, status)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		if d == nil {
			api.FailJSON(c, "NOT_FOUND")
			return
		}
		api.OK(c, d.Payload)

	case action == "delete":
		id := paramID(c)
		if err := e.Store.Delete(id); err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		api.OK(c, gin.H{})

	case strings.HasPrefix(action, "action:"):
		name := strings.TrimPrefix(action, "action:")
		id := paramID(c)
		var body map[string]interface{}
		_ = c.ShouldBindJSON(&body)
		d, err := e.Biz.ApplyDocAction(id, name, body)
		if err != nil {
			if be, ok := err.(*api.BusinessError); ok {
				api.FailJSON(c, be.Msg)
				return
			}
			api.FailJSON(c, err.Error())
			return
		}
		api.OK(c, d)

	default:
		// collection-level GET configs/stats etc.
		if method == "GET" {
			list, total, err := e.Store.List(resourceKey, pageNum, pageSize)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return
			}
			out := make([]map[string]interface{}, 0, len(list))
			for _, d := range list {
				out = append(out, d.Payload)
			}
			api.PageOK(c, out, total, pageNum, pageSize)
			return
		}
		if method == "POST" || method == "PUT" {
			var body map[string]interface{}
			_ = c.ShouldBindJSON(&body)
			if body == nil {
				body = map[string]interface{}{}
			}
			body["_action"] = action
			d, err := e.Store.Create(resourceKey, body, "done")
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return
			}
			api.OK(c, d.Payload)
			return
		}
		api.OK(c, gin.H{"ok": true})
	}
}

func paramID(c *gin.Context) int64 {
	for _, k := range []string{"id", "product_id", "order_id", "cost_id", "role_id"} {
		if v := c.Param(k); v != "" {
			id, _ := strconv.ParseInt(v, 10, 64)
			return id
		}
	}
	return 0
}

// DumpRoutes returns registered gin routes for coverage.
func DumpRoutes(eng *gin.Engine) []map[string]string {
	routes := eng.Routes()
	out := make([]map[string]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, map[string]string{"method": r.Method, "path": r.Path})
	}
	_ = json.Marshal // keep encoding/json used if needed
	return out
}
