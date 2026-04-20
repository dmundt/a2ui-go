package engine_test

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dmundt/au2ui-go/a2ui"
	"github.com/dmundt/au2ui-go/internal/engine"
	"github.com/dmundt/au2ui-go/internal/store"
	"github.com/dmundt/au2ui-go/internal/stream"
	"github.com/dmundt/au2ui-go/renderer"
)

func setup(t *testing.T) *httptest.Server {
	t.Helper()
	reg, err := renderer.NewRegistry("../../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../../internal/ui")
	mux := http.NewServeMux()
	engine.RegisterHTTPHandlers(mux, eng)
	return httptest.NewServer(mux)
}

func TestRenderAndPageEndpoint(t *testing.T) {
	srv := setup(t)
	defer srv.Close()

	jsonl := `{"version":"0.8","type":"begin","page_id":"home","surface":{"id":"s1","title":"Home","root":{"id":"root","type":"Text","text":{"value":"Hello"}}}}`
	resp, err := http.Post(srv.URL+"/render/a2ui", "application/jsonl", strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("post render: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	pageResp, err := http.Get(srv.URL + "/page/home")
	if err != nil {
		t.Fatalf("get page: %v", err)
	}
	defer pageResp.Body.Close()
	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected page status: %d", pageResp.StatusCode)
	}
}

func TestCatalogIndexEndpointRendersFromJSONL(t *testing.T) {
	srv := setup(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/catalog")
	if err != nil {
		t.Fatalf("get catalog index: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read catalog index: %v", err)
	}
	content := string(body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected catalog status: %d body=%s", resp.StatusCode, content)
	}
	if !strings.Contains(content, "A2UI v0.8 Reference Component Catalog") || !strings.Contains(content, "/catalog/button") {
		t.Fatalf("unexpected catalog index content: %s", content)
	}
}

func TestCatalogComponentEndpointsRenderAllBaseControls(t *testing.T) {
	srv := setup(t)
	defer srv.Close()

	for _, componentType := range a2ui.CatalogComponents() {
		component := strings.ToLower(string(componentType))
		resp, err := http.Get(srv.URL + "/catalog/" + component)
		if err != nil {
			t.Fatalf("get catalog component %s: %v", component, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read catalog component %s: %v", component, err)
		}

		content := string(body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d body=%s", component, resp.StatusCode, content)
		}
		if !strings.Contains(content, "Back to Catalog") {
			t.Fatalf("unexpected content for %s: %s", component, content)
		}
	}
}

func TestSSEStreamReceivesUpdate(t *testing.T) {
	srv := setup(t)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/stream", nil)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	streamResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer streamResp.Body.Close()

	done := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(streamResp.Body)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "event: update") {
				done <- line
				return
			}
		}
		done <- ""
	}()

	jsonl := `{"version":"0.8","type":"begin","page_id":"sse","surface":{"id":"sse1","title":"SSE","root":{"id":"root","type":"Text","text":{"value":"Tick"}}}}`
	resp, err := http.Post(srv.URL+"/render/a2ui", "application/jsonl", strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("trigger render: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	select {
	case got := <-done:
		if got == "" {
			t.Fatalf("stream ended without update event")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for sse event")
	}
}

func TestTableInspectorEndpoint(t *testing.T) {
	srv := setup(t)
	defer srv.Close()

	jsonl := `{"version":"0.8","type":"begin","page_id":"dashboard","surface":{"id":"dashboard-surface","title":"Dashboard","root":{"id":"root","type":"Column","column":{"gap":"12px"},"children":[{"id":"services-table","type":"Table","attrs":{"data-inspector-endpoint":"/inspect/table-row","data-page-id":"dashboard","data-inspector-target":"#service-inspector","data-inspector-component-id":"service-inspector"},"table":{"headers":["Service","Version","Status"],"rows":[["api","v1.8.2","healthy"],["worker","v1.8.2","healthy"]]}},{"id":"service-inspector","type":"Column","column":{"gap":"8px"},"children":[{"id":"placeholder","type":"Text","text":{"value":"Select a row."}}]}]}}}`
	resp, err := http.Post(srv.URL+"/render/a2ui", "application/jsonl", strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("post render: %v", err)
	}
	defer resp.Body.Close()

	inspectResp, err := http.Get(srv.URL + "/inspect/table-row?page_id=dashboard&table_id=services-table&row=0&target_id=service-inspector")
	if err != nil {
		t.Fatalf("get inspector: %v", err)
	}
	defer inspectResp.Body.Close()

	body, err := io.ReadAll(inspectResp.Body)
	if err != nil {
		t.Fatalf("read inspector body: %v", err)
	}
	content := string(body)
	if inspectResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected inspector status: %d body=%s", inspectResp.StatusCode, content)
	}
	if !strings.Contains(content, "Selected Service Inspector") || !strings.Contains(content, "api") || !strings.Contains(content, "healthy") {
		t.Fatalf("unexpected inspector content: %s", content)
	}
}
