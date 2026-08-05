package openapi

import (
	"regexp"
	"strings"
)

var methodLine = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
var braceParam = regexp.MustCompile(`\{([^}]+)\}`)
var colonParam = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// Op is one OpenAPI operation (method + path).
type Op struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// ParseOps extracts HTTP operations from an OpenAPI document text (paths section only).
func ParseOps(text string) []Op {
	parts := strings.SplitN(text, "paths:", 2)
	if len(parts) < 2 {
		return nil
	}
	body := parts[1]
	if i := strings.Index(body, "components:"); i >= 0 {
		body = body[:i]
	}
	var ops []Op
	var cur string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "  /") {
			cur = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		if m := methodLine.FindStringSubmatch(line); m != nil && cur != "" {
			ops = append(ops, Op{Method: strings.ToUpper(m[1]), Path: cur})
		}
	}
	return ops
}

// GinPath converts /api/v1/foo/{id} to /foo/:id (relative to /api/v1 group).
func GinPath(openapiPath string) string {
	p := openapiPath
	if strings.HasPrefix(p, "/api/v1") {
		p = p[len("/api/v1"):]
	}
	return braceParam.ReplaceAllString(p, ":$1")
}

// ResourceKey normalizes path for erp_doc storage.
func ResourceKey(path string) string {
	p := path
	if strings.HasPrefix(p, "/api/v1/") {
		p = p[len("/api/v1/"):]
	}
	var parts []string
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, "{") || strings.HasPrefix(seg, ":") {
			break
		}
		parts = append(parts, seg)
	}
	if len(parts) == 0 {
		return p
	}
	return strings.Join(parts, "/")
}

// Classify returns action name for engine dispatch.
func Classify(method, path string) string {
	segs := strings.Split(strings.TrimRight(path, "/"), "/")
	last := ""
	if len(segs) > 0 {
		last = segs[len(segs)-1]
	}
	if strings.HasPrefix(last, "{") || strings.HasPrefix(last, ":") {
		switch method {
		case "GET":
			return "get"
		case "PUT", "PATCH":
			return "update"
		case "DELETE":
			return "delete"
		default:
			return "action"
		}
	}
	switch method {
	case "GET":
		return "list"
	case "POST":
		if len(segs) >= 2 {
			prev := segs[len(segs)-2]
			if strings.HasPrefix(prev, "{") || strings.HasPrefix(prev, ":") {
				return "action:" + last
			}
		}
		return "create"
	case "PUT":
		return "replace"
	case "DELETE":
		return "delete_collection"
	default:
		return "action"
	}
}

// NormGinToOpenAPI converts :id style paths to {id}.
func NormGinToOpenAPI(p string) string {
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	return colonParam.ReplaceAllString(p, "{$1}")
}

// StripAPI removes /api/v1 prefix.
func StripAPI(p string) string {
	if strings.HasPrefix(p, "/api/v1") {
		return strings.TrimPrefix(p, "/api/v1")
	}
	return p
}
