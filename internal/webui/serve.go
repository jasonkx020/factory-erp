package webui

import (
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// Register mounts SPA/static handlers via NoRoute. Must not steal /api or /files.
func Register(eng *gin.Engine, ui *FS) {
	if ui == nil {
		return
	}
	eng.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/files") {
			c.Status(http.StatusNotFound)
			return
		}
		serveUI(c, ui, p)
	})
}

func serveUI(c *gin.Context, ui *FS, reqPath string) {
	rel := strings.TrimPrefix(path.Clean("/"+reqPath), "/")
	if rel == "." || rel == "" {
		rel = "index.html"
	}
	if tryFile(c, ui, rel) {
		return
	}
	if idx := ResolveSPAIndex(reqPath); idx != "" {
		if tryFile(c, ui, idx) {
			return
		}
	}
	c.Status(http.StatusNotFound)
}

func tryFile(c *gin.Context, ui *FS, name string) bool {
	f, err := ui.Open(name)
	if err != nil {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return false
	}
	if st.IsDir() {
		_ = f.Close()
		return tryFile(c, ui, path.Join(name, "index.html"))
	}
	defer f.Close()
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(c.Writer, c.Request, st.Name(), st.ModTime(), rs)
		return true
	}
	c.Header("Content-Type", contentTypeByName(name))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
	return true
}

func contentTypeByName(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// IsStaticPath reports paths that should skip JWT (HTML/JS/CSS assets and SPA shells).
func IsStaticPath(p string) bool {
	if p == "/" || p == "" {
		return true
	}
	if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/files") {
		return false
	}
	if strings.HasPrefix(p, "/admin") || strings.HasPrefix(p, "/front") {
		return true
	}
	if strings.HasPrefix(p, "/assets/") {
		return true
	}
	base := path.Base(p)
	if i := strings.LastIndex(base, "."); i >= 0 {
		ext := strings.ToLower(base[i:])
		switch ext {
		case ".html", ".js", ".css", ".map", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".woff", ".woff2", ".ttf", ".json", ".txt":
			return true
		}
	}
	if !strings.Contains(base, ".") {
		return true
	}
	return false
}
