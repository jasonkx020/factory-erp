package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Source describes where UI assets are loaded from.
type Source string

const (
	SourceEmbedded Source = "embedded"
	SourceExternal Source = "external"
)

// FS holds the HTTP-ready filesystem for portal/admin/boss static assets.
type FS struct {
	http.FileSystem
	Source Source
	Root   string // external root when SourceExternal; empty when embedded
}

// Open returns an HTTP filesystem. Prefer external root when it contains index.html;
// otherwise fall back to the embedded dist/.
func Open(externalRoot string) (*FS, error) {
	root := strings.TrimSpace(externalRoot)
	if root != "" {
		abs, err := filepath.Abs(root)
		if err == nil {
			if st, err := os.Stat(filepath.Join(abs, "index.html")); err == nil && !st.IsDir() {
				return &FS{FileSystem: http.FS(os.DirFS(abs)), Source: SourceExternal, Root: abs}, nil
			}
		}
	}
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, err
	}
	return &FS{FileSystem: http.FS(sub), Source: SourceEmbedded}, nil
}

// ResolveSPAIndex maps a request path to the SPA shell when the file is missing.
// Returns empty string when no fallback applies (caller should 404).
func ResolveSPAIndex(reqPath string) string {
	p := path.Clean("/" + strings.TrimPrefix(reqPath, "/"))
	if p == "/" {
		return "index.html"
	}
	// Real asset paths keep their extension; SPA routes do not.
	base := path.Base(p)
	if strings.Contains(base, ".") {
		return ""
	}
	if strings.HasPrefix(p, "/admin") {
		return "admin/index.html"
	}
	if strings.HasPrefix(p, "/front/boss") {
		return "front/boss/index.html"
	}
	// Legacy /front/ employee web is a static notice page under front/index.html
	if strings.HasPrefix(p, "/front") {
		return "front/index.html"
	}
	return "index.html"
}

// HasEmbeddedPortal reports whether embed contains a real portal (not just placeholder).
func HasEmbeddedPortal() bool {
	f, err := embedded.Open("dist/index.html")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
