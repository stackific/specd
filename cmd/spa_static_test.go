package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// TestMakeSPAStaticServesAssetsAndIndex verifies the static SPA handler
// returns real files for asset paths and the cached index.html for any
// non-asset path (SPA fallback).
func TestMakeSPAStaticServesAssetsAndIndex(t *testing.T) {
	const (
		indexBody = "<!doctype html><html><body>spa</body></html>"
		jsBody    = "console.log('x');"
	)
	fsys := fstest.MapFS{
		"index.html":  &fstest.MapFile{Data: []byte(indexBody)},
		"assets/x.js": &fstest.MapFile{Data: []byte(jsBody)},
	}

	h := makeSPAStatic(fsys)

	t.Run("asset path serves file", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/assets/x.js", http.NoBody)
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Body.String(); got != jsBody {
			t.Fatalf("body = %q, want %q", got, jsBody)
		}
	})

	t.Run("non-asset path returns index.html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/welcome", http.NoBody)
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Body.String(); got != indexBody {
			t.Fatalf("body = %q, want %q", got, indexBody)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("content-type = %q, want text/html...", ct)
		}
	})

	t.Run("nested SPA route returns index.html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/specs/SPEC-1", http.NoBody)
		h.ServeHTTP(rec, req)
		if got := rec.Body.String(); got != indexBody {
			t.Fatalf("body = %q, want %q", got, indexBody)
		}
	})
}
