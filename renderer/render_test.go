package renderer_test

import (
	"strings"
	"testing"

	"github.com/dmundt/a2ui-go/a2ui"
	"github.com/dmundt/a2ui-go/renderer"
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

	hello := "hello"
	components := map[string]*a2ui.Component{
		"root": {
			ID:   "root",
			Type: a2ui.ComponentColumn,
			Column: &a2ui.ColumnProps{
				Children: a2ui.Children{ExplicitList: []string{"t1"}},
			},
		},
		"t1": {
			ID:   "t1",
			Type: a2ui.ComponentText,
			Text: &a2ui.TextProps{Text: &a2ui.BoundValue{LiteralString: &hello}},
		},
	}

	h1, err := r.RenderSurface(components, a2ui.DataModel{}, "root")
	if err != nil {
		t.Fatalf("render1: %v", err)
	}
	h2, err := r.RenderSurface(components, a2ui.DataModel{}, "root")
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
