package mcp

import (
	"encoding/json"

	"github.com/dmundt/au2ui-go/internal/engine"
	"github.com/dmundt/au2ui-go/renderer"
)

// Handlers exposes tool methods for MCP.
type Handlers struct {
	engine   *engine.Engine
	registry *renderer.Registry
}

// NewHandlers creates MCP handlers.
func NewHandlers(engine *engine.Engine, registry *renderer.Registry) *Handlers {
	return &Handlers{engine: engine, registry: registry}
}

// Render validates and renders A2UI JSONL through shared engine.
func (h *Handlers) Render(jsonl string) (string, error) {
	return h.engine.ProcessJSONL(jsonl)
}

// ListPages returns pages in JSON text.
func (h *Handlers) ListPages() (string, error) {
	p := h.engine.ListPages()
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ListTemplates returns known template files in JSON text.
func (h *Handlers) ListTemplates() (string, error) {
	b, err := json.Marshal(h.engine.ListTemplateFiles())
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Health returns static health JSON text.
func (h *Handlers) Health() (string, error) {
	b, err := json.Marshal(h.engine.Health())
	if err != nil {
		return "", err
	}
	return string(b), nil
}
