package renderer_test

import (
	"strings"
	"testing"

	"github.com/dmundt/au2ui-go/a2ui"
	"github.com/dmundt/au2ui-go/renderer"
)

func TestRenderSurfaceDeterministic(t *testing.T) {
	reg, err := renderer.NewRegistry("templates")
	if err != nil {
		reg, err = renderer.NewRegistry("renderer/templates")
		if err != nil {
			t.Fatalf("registry: %v", err)
		}
	}
	r := renderer.New(reg)

	s := a2ui.Surface{
		ID:    "s1",
		Title: "Test",
		Root: a2ui.Component{
			ID:   "root",
			Type: a2ui.ComponentColumn,
			Column: &a2ui.ColumnProps{
				Gap: "8px",
			},
			Children: []a2ui.Component{{ID: "t1", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Value: "hello"}}},
		},
	}

	h1, err := r.RenderSurface(s)
	if err != nil {
		t.Fatalf("render1: %v", err)
	}
	h2, err := r.RenderSurface(s)
	if err != nil {
		t.Fatalf("render2: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("non-deterministic render")
	}
	if !strings.Contains(string(h1), "hello") {
		t.Fatalf("missing content")
	}
}
