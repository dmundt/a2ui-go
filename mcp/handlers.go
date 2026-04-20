package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dmundt/a2ui-go/a2ui"
	"github.com/dmundt/a2ui-go/internal/engine"
	"github.com/dmundt/a2ui-go/renderer"
)

// Handlers exposes tool methods for MCP.
type Handlers struct {
	engine   *engine.Engine
	registry *renderer.Registry
	examples string
}

// NewHandlers creates MCP handlers.
func NewHandlers(engine *engine.Engine, registry *renderer.Registry) *Handlers {
	return &Handlers{engine: engine, registry: registry, examples: resolveExamplesDir()}
}

type toolResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data,omitempty"`
	Warnings []string    `json:"warnings,omitempty"`
	Error    *toolError  `json:"error,omitempty"`
}

type toolError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func envelopeOK(data interface{}, warnings ...string) (string, error) {
	resp := toolResponse{Success: true, Data: data}
	if len(warnings) > 0 {
		resp.Warnings = warnings
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func envelopeErr(code, message string, details interface{}) (string, error) {
	resp := toolResponse{Success: false, Error: &toolError{Code: code, Message: message, Details: details}}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Render validates and renders A2UI JSONL through shared engine.
func (h *Handlers) Render(jsonl string) (string, error) {
	html, err := h.engine.ProcessJSONL(jsonl)
	if err != nil {
		return envelopeErr("INVALID_JSONL", err.Error(), nil)
	}
	return envelopeOK(map[string]interface{}{
		"html": html,
	})
}

// ListPages returns pages in JSON text.
func (h *Handlers) ListPages() (string, error) {
	return envelopeOK(h.engine.ListPages())
}

// ListTemplates returns known template files in JSON text.
func (h *Handlers) ListTemplates() (string, error) {
	return envelopeOK(h.engine.ListTemplateFiles())
}

// ListComposites returns available composite reference names in JSON text.
func (h *Handlers) ListComposites() (string, error) {
	names, err := h.engine.ListCompositeNames()
	if err != nil {
		return envelopeErr("LIST_COMPOSITES_FAILED", err.Error(), nil)
	}
	return envelopeOK(names)
}

// FetchComposite returns one composite reference payload in JSON text.
func (h *Handlers) FetchComposite(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	jsonl, err := h.engine.ReadCompositeJSONL(normalized)
	if err != nil {
		return envelopeErr("COMPOSITE_NOT_FOUND", err.Error(), map[string]interface{}{"name": normalized})
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
	return envelopeOK(payload)
}

// Health returns static health JSON text.
func (h *Handlers) Health() (string, error) {
	return envelopeOK(h.engine.Health())
}

// Capabilities returns MCP feature discovery metadata.
func (h *Handlers) Capabilities() (string, error) {
	doc := discoveryDoc()

	// Build composite component descriptors so the MCP client knows which
	// custom components (composites) are available alongside the base catalog.
	compositeNames, _ := h.engine.ListCompositeNames()
	compositeComponents := make([]map[string]interface{}, 0, len(compositeNames))
	for _, name := range compositeNames {
		compositeComponents = append(compositeComponents, map[string]interface{}{
			"name":    name,
			"route":   "/composites/" + name,
			"spec":    "A2UI",
			"version": a2ui.VersionV08,
			"source":  "composite",
			"fetchTool": map[string]interface{}{
				"tool": "a2ui_fetch_composite",
				"arg":  "name",
			},
		})
	}

	data := map[string]interface{}{
		"protocolVersion": a2ui.VersionV08,
		"protocolName":    "A2UI",
		"server":          "github.com/dmundt/a2ui-go",
		"requiredSpec": map[string]interface{}{
			"communication": map[string]interface{}{
				"format":  "jsonl",
				"spec":    "A2UI",
				"version": a2ui.VersionV08,
			},
			"ui": map[string]interface{}{
				"spec":    "A2UI",
				"version": a2ui.VersionV08,
			},
		},
		"features": []string{
			"render",
			"validate",
			"surfaces",
			"composites",
			"examples",
			"inspector",
			"runtime-lifecycle",
		},
		"tools":               toolNames(),
		"components":          a2ui.CatalogComponents(),
		"compositeComponents": compositeComponents,
		"discoverySchema":     doc["schemaVersion"],
		"discoveryTools":      []string{"a2ui_discovery", "a2ui_describe_tool"},
	}
	return envelopeOK(data)
}

// Discovery returns a machine-readable MCP profile so clients can self-bootstrap.
func (h *Handlers) Discovery() (string, error) {
	return envelopeOK(discoveryDoc())
}

// DescribeTool returns one machine-readable tool contract by name.
func (h *Handlers) DescribeTool(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return envelopeErr("INVALID_ARGUMENT", "name is required", map[string]interface{}{"field": "name"})
	}
	spec, ok := describeToolByName(normalized)
	if !ok {
		return envelopeErr("TOOL_NOT_FOUND", "unknown tool", map[string]interface{}{"name": normalized})
	}
	return envelopeOK(spec)
}

// ValidateJSONL checks payload syntax and A2UI semantics without mutating state.
func (h *Handlers) ValidateJSONL(jsonl string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(jsonl))
	lineNo := 0
	validated := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m a2ui.Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return envelopeErr("DECODE_ERROR", fmt.Sprintf("line %d decode: %v", lineNo, err), nil)
		}
		if err := a2ui.ValidateMessage(m); err != nil {
			return envelopeErr("VALIDATION_ERROR", fmt.Sprintf("line %d validate: %v", lineNo, err), nil)
		}
		validated++
	}
	if err := sc.Err(); err != nil {
		return envelopeErr("SCAN_ERROR", err.Error(), nil)
	}
	return envelopeOK(map[string]interface{}{"valid": true, "validatedLines": validated})
}

// ApplyJSONL applies valid JSONL updates and returns final render output, if available.
func (h *Handlers) ApplyJSONL(jsonl string) (string, error) {
	html, err := h.engine.ProcessJSONL(jsonl)
	if err != nil {
		return envelopeErr("APPLY_FAILED", err.Error(), nil)
	}
	return envelopeOK(map[string]interface{}{"html": html})
}

// ListSurfaces returns all in-memory surfaces.
func (h *Handlers) ListSurfaces() (string, error) {
	pages := h.engine.ListPages()
	ids := make([]string, 0, len(pages))
	for _, p := range pages {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return envelopeOK(map[string]interface{}{"ids": ids, "count": len(ids)})
}

// GetSurface returns one in-memory surface record.
func (h *Handlers) GetSurface(surfaceID string) (string, error) {
	p, ok := h.engine.GetPage(surfaceID)
	if !ok {
		return envelopeErr("SURFACE_NOT_FOUND", "surface not found", map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID)})
	}
	return envelopeOK(p)
}

// GetSurfaceModel returns one surface data model.
func (h *Handlers) GetSurfaceModel(surfaceID string) (string, error) {
	p, ok := h.engine.GetPage(surfaceID)
	if !ok {
		return envelopeErr("SURFACE_NOT_FOUND", "surface not found", map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID)})
	}
	return envelopeOK(map[string]interface{}{"surfaceId": p.ID, "dataModel": p.DataModel})
}

// GetSurfaceComponents returns component ids and root metadata for one surface.
func (h *Handlers) GetSurfaceComponents(surfaceID string) (string, error) {
	p, ok := h.engine.GetPage(surfaceID)
	if !ok {
		return envelopeErr("SURFACE_NOT_FOUND", "surface not found", map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID)})
	}
	ids := make([]string, 0, len(p.Components))
	for id := range p.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return envelopeOK(map[string]interface{}{"surfaceId": p.ID, "rootId": p.RootID, "componentIds": ids, "count": len(ids)})
}

// RenderSurface renders one in-memory surface by id.
func (h *Handlers) RenderSurface(surfaceID string) (string, error) {
	html, err := h.engine.RenderSurfaceByID(surfaceID)
	if err != nil {
		return envelopeErr("RENDER_SURFACE_FAILED", err.Error(), map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID)})
	}
	return envelopeOK(map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID), "html": html})
}

// CreateSurface creates one empty surface.
func (h *Handlers) CreateSurface(surfaceID string) (string, error) {
	p, err := h.engine.CreateSurface(surfaceID)
	if err != nil {
		return envelopeErr("CREATE_SURFACE_FAILED", err.Error(), nil)
	}
	return envelopeOK(map[string]interface{}{"surfaceId": p.ID})
}

// DeleteSurface deletes one surface by id.
func (h *Handlers) DeleteSurface(surfaceID string) (string, error) {
	deleted := h.engine.DeleteSurface(surfaceID)
	if !deleted {
		return envelopeErr("SURFACE_NOT_FOUND", "surface not found", map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID)})
	}
	return envelopeOK(map[string]interface{}{"surfaceId": strings.TrimSpace(surfaceID), "deleted": true})
}

// ResetRuntime clears all in-memory surfaces.
func (h *Handlers) ResetRuntime() (string, error) {
	removed := h.engine.ResetRuntime()
	return envelopeOK(map[string]interface{}{"removed": removed})
}

// InspectTableRow renders inspector markup for one table row.
func (h *Handlers) InspectTableRow(pageID, tableID string, row int, targetID string) (string, error) {
	html, err := h.engine.InspectTableRow(pageID, tableID, row, targetID)
	if err != nil {
		return envelopeErr("INSPECT_FAILED", err.Error(), map[string]interface{}{"pageId": strings.TrimSpace(pageID), "tableId": strings.TrimSpace(tableID), "row": row})
	}
	return envelopeOK(map[string]interface{}{"html": html})
}

// ExamplesList returns available example payload names from examples/*.jsonl.
func (h *Handlers) ExamplesList() (string, error) {
	entries, err := os.ReadDir(h.examples)
	if err != nil {
		return envelopeErr("LIST_EXAMPLES_FAILED", err.Error(), map[string]interface{}{"dir": h.examples})
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".jsonl") {
			names = append(names, strings.TrimSuffix(name, ".jsonl"))
		}
	}
	sort.Strings(names)
	return envelopeOK(map[string]interface{}{"examples": names, "count": len(names)})
}

// ExampleGet returns one example JSONL payload from the examples directory.
func (h *Handlers) ExampleGet(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" || strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") || strings.Contains(normalized, "..") {
		return envelopeErr("INVALID_EXAMPLE_NAME", "invalid example name", map[string]interface{}{"name": name})
	}
	path := filepath.Join(h.examples, normalized+".jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return envelopeErr("EXAMPLE_NOT_FOUND", "example not found", map[string]interface{}{"name": normalized})
		}
		return envelopeErr("READ_EXAMPLE_FAILED", err.Error(), map[string]interface{}{"name": normalized})
	}
	return envelopeOK(map[string]interface{}{"name": normalized, "jsonl": string(b)})
}

func resolveExamplesDir() string {
	candidates := []string{
		"examples",
		filepath.Join(".", "examples"),
		filepath.Join("..", "examples"),
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, "examples"))
		candidates = append(candidates, filepath.Join(exeDir, "..", "examples"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			if abs, aerr := filepath.Abs(c); aerr == nil {
				return abs
			}
			return c
		}
	}
	return "examples"
}
