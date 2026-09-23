package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticDeploymentCacheAndMissingBundles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<!doctype html><div id="root">Current app</div><script src="/assets/current.js"></script>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "current.js"), []byte(`console.log("current bundle")`), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewServer(nil, nil, Config{StaticDir: dir})
	for _, tc := range []struct {
		path   string
		status int
		cache  string
		html   bool
	}{
		{"/", http.StatusOK, "no-cache", true},
		{"/org/test--123/events/42", http.StatusOK, "no-cache", true},
		{"/assets/current.js", http.StatusOK, "", false},
		{"/assets/previous-deployment.js", http.StatusNotFound, "no-store", false},
		{"/assets/previous-deployment.css", http.StatusNotFound, "no-store", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("HTTP %d, want %d", response.Code, tc.status)
			}
			if cache := response.Header().Get("Cache-Control"); cache != tc.cache {
				t.Fatalf("Cache-Control=%q, want %q", cache, tc.cache)
			}
			if html := strings.Contains(response.Header().Get("Content-Type"), "text/html"); html != tc.html {
				t.Fatalf("wrong content type: %s", response.Header().Get("Content-Type"))
			}
			if tc.status == http.StatusNotFound && strings.Contains(response.Body.String(), "Current app") {
				t.Fatal("missing bundle returned the SPA shell")
			}
		})
	}
}
