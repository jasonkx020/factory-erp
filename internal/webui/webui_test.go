package webui

import "testing"

func TestResolveSPAIndex(t *testing.T) {
	cases := map[string]string{
		"/":                              "index.html",
		"/admin":                         "admin/index.html",
		"/admin/":                        "admin/index.html",
		"/admin/production/hub/tasks":    "admin/index.html",
		"/front/boss":                    "front/boss/index.html",
		"/front/boss/dashboard":          "front/boss/index.html",
		"/front":                         "front/index.html",
		"/assets/index-abc.js":           "",
		"/admin/assets/index-abc.js":     "",
		"/favicon.ico":                   "",
	}
	for in, want := range cases {
		got := ResolveSPAIndex(in)
		if got != want {
			t.Fatalf("ResolveSPAIndex(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsStaticPath(t *testing.T) {
	if !IsStaticPath("/") || !IsStaticPath("/admin/x") || !IsStaticPath("/assets/a.js") {
		t.Fatal("expected static")
	}
	if IsStaticPath("/api/v1/health") {
		t.Fatal("api must not skip as static UI")
	}
	// /files is JWT-permitted separately; IsStaticPath leaves it false
	if IsStaticPath("/files/uploads/a.png") {
		t.Fatal("/files should not use UI static skip")
	}
}

func TestOpenFallbackEmbedded(t *testing.T) {
	fs, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	if fs.Source != SourceEmbedded {
		t.Fatalf("want embedded got %s", fs.Source)
	}
	f, err := fs.Open("index.html")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}
