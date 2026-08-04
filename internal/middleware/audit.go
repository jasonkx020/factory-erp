package middleware

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CtxTraceID = "trace_id"

// bodyLogWriter captures response for audit.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Audit writes sys_operation_log for non-GET requests.
func Audit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		c.Set(CtxTraceID, traceID)
		c.Header("X-Trace-Id", traceID)

		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			c.Next()
			return
		}

		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		blw := &bodyLogWriter{ResponseWriter: c.Writer, body: bytes.NewBuffer(nil)}
		c.Writer = blw
		c.Next()

		claims := Claims(c)
		var userID interface{}
		if claims != nil {
			userID = claims.UserID
		}
		action := c.Request.Method + " " + c.FullPath()
		if action == c.Request.Method+" " {
			action = c.Request.Method + " " + c.Request.URL.Path
		}
		module := moduleFromPath(c.Request.URL.Path)
		detail := map[string]interface{}{
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"status":      c.Writer.Status(),
			"request":     jsonRaw(reqBody),
			"response":    jsonRaw(blw.body.Bytes()),
			"recorded_at": time.Now().Format("2006-01-02 15:04:05"),
		}
		b, _ := json.Marshal(detail)
		refType, refID := extractRef(c, reqBody, blw.body.Bytes())
		_, _ = db.Exec(
			`INSERT INTO sys_operation_log(user_id, action, module, ref_type, ref_id, detail_json, ip, trace_id, created_at)
			 VALUES(?,?,?,?,?,?,?,?,datetime('now'))`,
			userID, action, module, refType, refID, string(b), c.ClientIP(), traceID,
		)
	}
}

func TraceID(c *gin.Context) string {
	if v, ok := c.Get(CtxTraceID); ok {
		if s, _ := v.(string); s != "" {
			return s
		}
	}
	return ""
}

func moduleFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2] // api/v1/{domain}
	}
	return path
}

func jsonRaw(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	return v
}

func extractRef(c *gin.Context, req, resp []byte) (string, interface{}) {
	if id := c.Param("id"); id != "" {
		return c.FullPath(), id
	}
	var m map[string]interface{}
	if json.Unmarshal(resp, &m) == nil {
		if data, ok := m["data"].(map[string]interface{}); ok {
			if id, ok := data["id"]; ok {
				return c.Request.URL.Path, id
			}
		}
	}
	_ = req
	return "", nil
}
