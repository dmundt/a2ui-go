package mcp

import (
	"encoding/json"
	"strings"

	"github.com/dmundt/a2ui-go/internal/engine"
	"github.com/dmundt/a2ui-go/renderer"
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

// ListComposites returns available composite reference names in JSON text.
func (h *Handlers) ListComposites() (string, error) {
	names, err := h.engine.ListCompositeNames()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(names)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FetchComposite returns one composite reference payload in JSON text.
func (h *Handlers) FetchComposite(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	jsonl, err := h.engine.ReadCompositeJSONL(normalized)
	if err != nil {
		return "", err
	}
	payload := struct {
		Name  string `json:"name"`
		Route string `json:"route"`
		JSONL string `json:"jsonl"`
	}{
		Name:  normalized,
		Route: "/composites/" + normalized,
		JSONL: jsonl,
	}
	b, err := json.Marshal(payload)
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
