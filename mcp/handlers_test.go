package mcp_test

import (
	"strings"
	"testing"

	"github.com/dmundt/au2ui-go/internal/engine"
	"github.com/dmundt/au2ui-go/internal/store"
	"github.com/dmundt/au2ui-go/internal/stream"
	"github.com/dmundt/au2ui-go/mcp"
	"github.com/dmundt/au2ui-go/renderer"
)

func TestHandlersRenderAndList(t *testing.T) {
	reg, err := renderer.NewRegistry("../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../internal/ui")
	h := mcp.NewHandlers(eng, reg)

	jsonl := `{"version":"0.8","type":"begin","page_id":"p1","surface":{"id":"s1","title":"T","root":{"id":"root","type":"Text","text":{"value":"Hello"}}}}`
	html, err := h.Render(jsonl)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Hello") {
		t.Fatalf("render output missing text")
	}

	pages, err := h.ListPages()
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	if !strings.Contains(pages, "\"ID\":\"p1\"") {
		t.Fatalf("unexpected page list: %s", pages)
	}

	templates, err := h.ListTemplates()
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if !strings.Contains(templates, "page.html") {
		t.Fatalf("unexpected templates output: %s", templates)
	}
}
