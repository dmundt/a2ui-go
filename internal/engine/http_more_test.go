package engine_test

import (
	"net/http"
	"testing"
)

func TestHTTPErrorAndFallbackRoutes(t *testing.T) {
	t.Run("render method not allowed", func(t *testing.T) {
		srv := setup(t)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/render/a2ui")
		if err != nil {
			t.Fatalf("GET /render/a2ui: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})

	t.Run("page endpoint missing and unknown ids", func(t *testing.T) {
		srv := setup(t)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/page/")
		if err != nil {
			t.Fatalf("GET /page/: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing page ID, got %d", resp.StatusCode)
		}

		resp2, err := http.Get(srv.URL + "/page/does-not-exist")
		if err != nil {
			t.Fatalf("GET /page/does-not-exist: %v", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for missing page, got %d", resp2.StatusCode)
		}
	})

	t.Run("catalog and composites route filtering", func(t *testing.T) {
		srv := setup(t)
		defer srv.Close()

		for path, want := range map[string]int{
			"/catalog/table":      http.StatusNotFound,
			"/catalog/unknown":    http.StatusNotFound,
			"/catalog/extra/path": http.StatusNotFound,
			"/composites/unknown": http.StatusNotFound,
		} {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != want {
				t.Fatalf("%s expected %d got %d", path, want, resp.StatusCode)
			}
		}
	})

	t.Run("inspector validation errors", func(t *testing.T) {
		srv := setup(t)
		defer srv.Close()

		cases := map[string]int{
			"/inspect/table-row": http.StatusBadRequest,
			"/inspect/table-row?page_id=p1&table_id=t1&row=abc":                                  http.StatusBadRequest,
			"/inspect/table-row?page_id=missing&table_id=t1&row=0":                               http.StatusNotFound,
			"/inspect/table-row?page_id=debug-surface&table_id=missing&row=0":                    http.StatusNotFound,
			"/inspect/table-row?page_id=debug-surface&table_id=missing&row=999&target_id=custom": http.StatusNotFound,
		}
		for path, want := range cases {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != want {
				t.Fatalf("%s expected %d got %d", path, want, resp.StatusCode)
			}
		}
	})

	t.Run("fallback when ui directory has no files", func(t *testing.T) {
		srv := setupWithUIDir(t, t.TempDir())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for missing index ui file, got %d", resp.StatusCode)
		}

		resp2, err := http.Get(srv.URL + "/debug")
		if err != nil {
			t.Fatalf("GET /debug: %v", err)
		}
		resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 debug fallback, got %d", resp2.StatusCode)
		}
	})
}
