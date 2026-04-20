package renderer

import (
	"bytes"
	"html/template"

	"github.com/dmundt/a2ui-go/a2ui"
)

// Renderer renders A2UI components into HTML.
type Renderer struct {
	registry *Registry
}

// New creates a deterministic A2UI renderer.
func New(registry *Registry) *Renderer {
	return &Renderer{registry: registry}
}

// RenderSurface renders a surface given its component map, data model and root component ID.
func (r *Renderer) RenderSurface(components map[string]*a2ui.Component, dm a2ui.DataModel, rootID string) (template.HTML, error) {
	body, err := r.renderByID(components, dm, rootID)
	if err != nil {
		return "", err
	}

	data := struct {
		Title string
		Body  template.HTML
	}{
		Title: rootID,
		Body:  body,
	}

	var out bytes.Buffer
	if err := r.registry.Templates().ExecuteTemplate(&out, "page", data); err != nil {
		return "", err
	}

	return template.HTML(out.String()), nil
}

// renderByID looks up a component by ID and renders it.
func (r *Renderer) renderByID(components map[string]*a2ui.Component, dm a2ui.DataModel, id string) (template.HTML, error) {
	c, ok := components[id]
	if !ok {
		return "", nil // missing child silently skipped
	}
	return r.RenderComponent(components, dm, c)
}
