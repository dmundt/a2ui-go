package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmundt/a2ui-go/a2ui"
	"github.com/dmundt/a2ui-go/internal/store"
	"github.com/dmundt/a2ui-go/internal/stream"
	"github.com/dmundt/a2ui-go/renderer"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	reg, err := renderer.NewRegistry("../../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	return New(r, reg, store.NewPageStore(), stream.NewBroker(), "../../internal/ui")
}

func mustJSONLine(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json line: %v", err)
	}
	return string(b)
}

func TestEngineProcessJSONL_WithDataModelUpdateAndDelete(t *testing.T) {
	eng := newTestEngine(t)

	greeting := "Hello Data Model"
	jsonl := strings.Join([]string{
		mustJSONLine(t, a2ui.Message{SurfaceUpdate: &a2ui.SurfaceUpdatePayload{
			SurfaceID: "p1",
			Components: []a2ui.RawComponentDef{{
				ID: "root",
				Component: map[string]json.RawMessage{
					"Text": json.RawMessage(`{"text":{"path":"/greeting"}}`),
				},
			}},
		}}),
		mustJSONLine(t, a2ui.Message{DataModelUpdate: &a2ui.DataModelUpdatePayload{
			SurfaceID: "p1",
			Path:      "/",
			Contents:  []a2ui.DataEntry{{Key: "greeting", ValueString: &greeting}},
		}}),
		mustJSONLine(t, a2ui.Message{BeginRendering: &a2ui.BeginRenderingPayload{SurfaceID: "p1", Root: "root"}}),
	}, "\n")

	html, err := eng.ProcessJSONL(jsonl)
	if err != nil {
		t.Fatalf("ProcessJSONL error: %v", err)
	}
	if !strings.Contains(html, greeting) {
		t.Fatalf("rendered html missing greeting: %s", html)
	}

	pages := eng.ListPages()
	if len(pages) != 1 || pages[0].ID != "p1" {
		t.Fatalf("unexpected page list: %#v", pages)
	}

	if _, err := eng.ApplyMessage(a2ui.Message{DeleteSurface: &a2ui.DeleteSurfacePayload{SurfaceID: "p1"}}); err != nil {
		t.Fatalf("DeleteSurface apply failed: %v", err)
	}
	if got := eng.ListPages(); len(got) != 0 {
		t.Fatalf("expected empty pages after delete, got %#v", got)
	}
}

func TestEngineUIFileAndLookupHelpers(t *testing.T) {
	eng := newTestEngine(t)

	html, ok, err := eng.renderUIFile("index.jsonl")
	if err != nil {
		t.Fatalf("renderUIFile index error: %v", err)
	}
	if !ok || !strings.Contains(html, "<!doctype html>") {
		t.Fatalf("renderUIFile index unexpected result ok=%v", ok)
	}

	if _, ok, err := eng.renderUIFile("does-not-exist.jsonl"); err != nil || ok {
		t.Fatalf("renderUIFile missing expected (false,nil), got ok=%v err=%v", ok, err)
	}

	// Existing UI files can render, but their internal surface IDs are not guaranteed
	// to match their filenames, so this can legitimately return an error.
	if _, ok, err := eng.loadPageFromUIFile("index"); err == nil || ok {
		t.Fatalf("loadPageFromUIFile index expected mismatch error, ok=%v err=%v", ok, err)
	}
	if _, ok, err := eng.loadPageFromUIFile("not-a-page"); err != nil || ok {
		t.Fatalf("loadPageFromUIFile missing expected not found, ok=%v err=%v", ok, err)
	}

	if health := eng.Health(); health["status"] != "ok" || health["version"] != a2ui.VersionV08 {
		t.Fatalf("unexpected health: %#v", health)
	}

	templates := eng.ListTemplateFiles()
	if len(templates) == 0 {
		t.Fatalf("expected non-empty template list")
	}
}

func TestLoadPageFromUIFile_SuccessWhenSurfaceIDMatchesFileName(t *testing.T) {
	reg, err := renderer.NewRegistry("../../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	eng := New(renderer.New(reg), reg, store.NewPageStore(), stream.NewBroker(), t.TempDir())

	file := filepath.Join(eng.uiDir, "demo.jsonl")
	payload := "{\"surfaceUpdate\":{\"surfaceId\":\"demo\",\"components\":[{\"id\":\"root\",\"component\":{\"Text\":{\"text\":{\"literalString\":\"Demo\"}}}}]}}\n{\"beginRendering\":{\"surfaceId\":\"demo\",\"root\":\"root\"}}\n"
	if err := os.WriteFile(file, []byte(payload), 0o644); err != nil {
		t.Fatalf("write demo jsonl: %v", err)
	}

	p, ok, err := eng.loadPageFromUIFile("demo")
	if err != nil || !ok {
		t.Fatalf("expected successful load, ok=%v err=%v", ok, err)
	}
	if p.ID != "demo" || !strings.Contains(p.HTML, "Demo") {
		t.Fatalf("unexpected loaded page: %#v", p)
	}
}

func TestEngineHelpersAndErrors(t *testing.T) {
	if got, ok := catalogComponentFileName("table"); ok || got != "" {
		t.Fatalf("table should be excluded from catalog route: got=%q ok=%v", got, ok)
	}
	if got, ok := catalogComponentFileName("text"); !ok || got == "" {
		t.Fatalf("expected text component file name, got=%q ok=%v", got, ok)
	}
	if got, ok := catalogComponentFileName("unknown-component"); ok || got != "" {
		t.Fatalf("unknown component should fail lookup: got=%q ok=%v", got, ok)
	}

	if _, err := resolveFile("README.md"); err != nil {
		t.Fatalf("resolveFile should find README.md: %v", err)
	}
	if _, err := resolveFile("definitely-not-a-real-file.txt"); err == nil {
		t.Fatalf("resolveFile expected error for missing file")
	}

	eng := newTestEngine(t)
	if _, err := eng.ApplyMessage(a2ui.Message{}); err == nil {
		t.Fatalf("ApplyMessage expected error for empty message")
	}
	if _, err := eng.ProcessJSONL("{invalid"); err == nil {
		t.Fatalf("ProcessJSONL expected decode error")
	}
}
