package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dmundt/a2ui-go/a2ui"
	"github.com/dmundt/a2ui-go/internal/store"
	"github.com/dmundt/a2ui-go/internal/stream"
	"github.com/dmundt/a2ui-go/renderer"
	"github.com/go-chi/chi/v5"
)

// Engine validates, applies updates, renders html, and persists pages.
type Engine struct {
	renderer *renderer.Renderer
	registry *renderer.Registry
	store    *store.PageStore
	broker   *stream.Broker
	uiDir    string
}

// New creates an engine with explicit dependencies.
func New(r *renderer.Renderer, reg *renderer.Registry, s *store.PageStore, b *stream.Broker, uiDir string) *Engine {
	return &Engine{renderer: r, registry: reg, store: s, broker: b, uiDir: uiDir}
}

// ProcessJSONL parses and applies A2UI messages from JSONL.
func (e *Engine) ProcessJSONL(jsonl string) (string, error) {
	var lastHTML string
	sc := bufio.NewScanner(strings.NewReader(jsonl))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m a2ui.Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return "", fmt.Errorf("line %d decode: %w", lineNo, err)
		}
		if err := a2ui.ValidateMessage(m); err != nil {
			return "", fmt.Errorf("line %d validate: %w", lineNo, err)
		}
		h, err := e.ApplyMessage(m)
		if err != nil {
			return "", fmt.Errorf("line %d apply: %w", lineNo, err)
		}
		if h != "" {
			lastHTML = h
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return lastHTML, nil
}

// ApplyMessage processes one validated v0.8 message.
func (e *Engine) ApplyMessage(m a2ui.Message) (string, error) {
	switch {
	case m.SurfaceUpdate != nil:
		return e.applySurfaceUpdate(m.SurfaceUpdate)
	case m.DataModelUpdate != nil:
		return e.applyDataModelUpdate(m.DataModelUpdate)
	case m.BeginRendering != nil:
		return e.applyBeginRendering(m.BeginRendering)
	case m.DeleteSurface != nil:
		e.store.Delete(m.DeleteSurface.SurfaceID)
		return "", nil
	}
	return "", fmt.Errorf("empty message")
}

func (e *Engine) getOrCreatePage(surfaceID string) store.Page {
	p, ok := e.store.Get(surfaceID)
	if !ok {
		p = store.Page{
			ID:         surfaceID,
			Components: make(map[string]*a2ui.Component),
			DataModel:  make(a2ui.DataModel),
			UpdatedAt:  time.Now().UTC(),
		}
	}
	return p
}

func (e *Engine) applySurfaceUpdate(msg *a2ui.SurfaceUpdatePayload) (string, error) {
	p := e.getOrCreatePage(msg.SurfaceID)
	for _, raw := range msg.Components {
		c, err := a2ui.ParseComponentDef(raw)
		if err != nil {
			return "", fmt.Errorf("surfaceUpdate component %q: %w", raw.ID, err)
		}
		p.Components[c.ID] = &c
	}
	p.UpdatedAt = time.Now().UTC()
	e.store.Upsert(p)
	if p.Ready {
		return e.renderAndPublish(p)
	}
	return "", nil
}

func (e *Engine) applyDataModelUpdate(msg *a2ui.DataModelUpdatePayload) (string, error) {
	p := e.getOrCreatePage(msg.SurfaceID)
	p.DataModel.ApplyUpdate(msg.Path, msg.Contents)
	p.UpdatedAt = time.Now().UTC()
	e.store.Upsert(p)
	if p.Ready {
		return e.renderAndPublish(p)
	}
	return "", nil
}

func (e *Engine) applyBeginRendering(msg *a2ui.BeginRenderingPayload) (string, error) {
	p := e.getOrCreatePage(msg.SurfaceID)
	p.RootID = msg.Root
	p.CatalogID = msg.CatalogID
	p.Ready = true
	p.UpdatedAt = time.Now().UTC()
	e.store.Upsert(p)
	return e.renderAndPublish(p)
}

func (e *Engine) renderAndPublish(p store.Page) (string, error) {
	rendered, err := e.renderer.RenderSurface(p.Components, p.DataModel, p.RootID)
	if err != nil {
		return "", err
	}
	p.HTML = string(rendered)
	p.UpdatedAt = time.Now().UTC()
	e.store.Upsert(p)
	e.broker.Publish(string(rendered))
	return string(rendered), nil
}

// CreateSurface creates an empty surface when it does not already exist.
func (e *Engine) CreateSurface(surfaceID string) (store.Page, error) {
	id := strings.TrimSpace(surfaceID)
	if id == "" {
		return store.Page{}, fmt.Errorf("missing surface id")
	}
	if p, ok := e.store.Get(id); ok {
		return p, nil
	}
	p := store.Page{
		ID:         id,
		Components: make(map[string]*a2ui.Component),
		DataModel:  make(a2ui.DataModel),
		UpdatedAt:  time.Now().UTC(),
	}
	e.store.Upsert(p)
	return p, nil
}

// DeleteSurface removes one surface by id and reports whether it existed.
func (e *Engine) DeleteSurface(surfaceID string) bool {
	id := strings.TrimSpace(surfaceID)
	if id == "" {
		return false
	}
	if _, ok := e.store.Get(id); !ok {
		return false
	}
	e.store.Delete(id)
	return true
}

// ResetRuntime clears all in-memory surfaces and returns the number removed.
func (e *Engine) ResetRuntime() int {
	pages := e.store.List()
	for _, p := range pages {
		e.store.Delete(p.ID)
	}
	return len(pages)
}

// GetPage returns one surface from the in-memory store.
func (e *Engine) GetPage(surfaceID string) (store.Page, bool) {
	id := strings.TrimSpace(surfaceID)
	if id == "" {
		return store.Page{}, false
	}
	return e.store.Get(id)
}

// RenderSurfaceByID renders one in-memory surface by id.
func (e *Engine) RenderSurfaceByID(surfaceID string) (string, error) {
	p, ok := e.GetPage(surfaceID)
	if !ok {
		return "", fmt.Errorf("surface %q not found", strings.TrimSpace(surfaceID))
	}
	if strings.TrimSpace(p.RootID) == "" {
		return "", fmt.Errorf("surface %q has no root", p.ID)
	}
	rendered, err := e.renderer.RenderSurface(p.Components, p.DataModel, p.RootID)
	if err != nil {
		return "", err
	}
	p.HTML = string(rendered)
	p.UpdatedAt = time.Now().UTC()
	e.store.Upsert(p)
	return p.HTML, nil
}

// InspectTableRow renders the inline inspector HTML for a table row.
func (e *Engine) InspectTableRow(pageID, tableID string, rowIndex int, targetID string) (string, error) {
	pageID = strings.TrimSpace(pageID)
	tableID = strings.TrimSpace(tableID)
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		targetID = "inspector-target"
	}
	if pageID == "" || tableID == "" {
		return "", fmt.Errorf("missing page id or table id")
	}

	page, ok := e.store.Get(pageID)
	if !ok {
		return "", fmt.Errorf("page not found")
	}
	tableComp := page.Components[tableID]
	if tableComp == nil || tableComp.Table == nil {
		return "", fmt.Errorf("table not found")
	}
	if rowIndex < 0 || rowIndex >= len(tableComp.Table.Rows) {
		return "", fmt.Errorf("row index out of range")
	}

	if tableID == "items-table" && targetID == "items-inspector-form" {
		var rowAction *a2ui.ActionDef
		if tableComp.Table.RowActions != nil && rowIndex < len(tableComp.Table.RowActions) {
			rowAction = tableComp.Table.RowActions[rowIndex]
		}
		if inspectorComps, rootID, ok := buildItemInspectorComponents(targetID, rowAction, page.DataModel); ok {
			htmlOut, err := e.renderer.RenderSurface(inspectorComps, page.DataModel, rootID)
			if err != nil {
				return "", err
			}
			return string(htmlOut), nil
		}
	}

	inspectorComps, rootID := buildInspectorComponents(targetID, tableComp.Table.Headers, tableComp.Table.Rows[rowIndex])
	htmlOut, err := e.renderer.RenderSurface(inspectorComps, page.DataModel, rootID)
	if err != nil {
		return "", err
	}
	return string(htmlOut), nil
}

func buildItemInspectorComponents(targetID string, rowAction *a2ui.ActionDef, dm a2ui.DataModel) (map[string]*a2ui.Component, string, bool) {
	if rowAction == nil {
		return nil, "", false
	}

	ctx := make(map[string]string)
	for _, entry := range rowAction.Context {
		ctx[entry.Key] = entry.Value.Str(dm)
	}

	required := []string{"itemId", "itemName", "itemCategory", "itemStatus", "itemSerial", "itemPurchaseDate", "itemValue", "locationBuilding", "locationFloor", "locationName"}
	for _, key := range required {
		if strings.TrimSpace(ctx[key]) == "" {
			return nil, "", false
		}
	}

	literalStr := func(s string) *a2ui.BoundValue { return &a2ui.BoundValue{LiteralString: &s} }
	comps := make(map[string]*a2ui.Component)

	formID := targetID
	titleID := "items-inspector-title"

	comps[titleID] = &a2ui.Component{
		ID:   titleID,
		Type: a2ui.ComponentText,
		Text: &a2ui.TextProps{Text: literalStr("Item Inspector"), UsageHint: "h3"},
	}

	type pair struct {
		labelID string
		valueID string
		label   string
		value   string
	}

	pairs := []pair{
		{labelID: "inspector-item-id-label", valueID: "inspector-item-id", label: "Asset ID", value: ctx["itemId"]},
		{labelID: "inspector-item-name-label", valueID: "inspector-item-name", label: "Name", value: ctx["itemName"]},
		{labelID: "inspector-item-cat-label", valueID: "inspector-item-cat", label: "Category", value: ctx["itemCategory"]},
		{labelID: "inspector-item-status-label", valueID: "inspector-item-status", label: "Status", value: ctx["itemStatus"]},
		{labelID: "inspector-item-serial-label", valueID: "inspector-item-serial", label: "Serial Number", value: ctx["itemSerial"]},
		{labelID: "inspector-item-purchase-label", valueID: "inspector-item-purchase", label: "Purchase Date", value: ctx["itemPurchaseDate"]},
		{labelID: "inspector-item-value-label", valueID: "inspector-item-value", label: "Value", value: ctx["itemValue"]},
	}

	for _, p := range pairs {
		comps[p.labelID] = &a2ui.Component{ID: p.labelID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(p.label)}}
		comps[p.valueID] = &a2ui.Component{ID: p.valueID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(p.value)}}
	}

	locLabelID := "inspector-item-location-label"
	locID := "inspector-item-location"
	crumbIDs := []string{"inspector-item-location-crumb-building", "inspector-item-location-crumb-sep-1", "inspector-item-location-crumb-floor", "inspector-item-location-crumb-sep-2", "inspector-item-location-crumb-name"}

	comps[locLabelID] = &a2ui.Component{ID: locLabelID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr("Location")}}
	comps[crumbIDs[0]] = &a2ui.Component{ID: crumbIDs[0], Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(ctx["locationBuilding"])}}
	comps[crumbIDs[1]] = &a2ui.Component{ID: crumbIDs[1], Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(">")}}
	comps[crumbIDs[2]] = &a2ui.Component{ID: crumbIDs[2], Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(ctx["locationFloor"])}}
	comps[crumbIDs[3]] = &a2ui.Component{ID: crumbIDs[3], Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(">")}}
	comps[crumbIDs[4]] = &a2ui.Component{ID: crumbIDs[4], Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(ctx["locationName"])}}
	comps[locID] = &a2ui.Component{ID: locID, Type: a2ui.ComponentList, List: &a2ui.ListProps{Direction: "horizontal", Children: a2ui.Children{ExplicitList: crumbIDs}}}

	formChildren := []string{
		titleID,
		"inspector-item-id-label", "inspector-item-id",
		"inspector-item-name-label", "inspector-item-name",
		"inspector-item-cat-label", "inspector-item-cat",
		"inspector-item-status-label", "inspector-item-status",
		locLabelID, locID,
		"inspector-item-serial-label", "inspector-item-serial",
		"inspector-item-purchase-label", "inspector-item-purchase",
		"inspector-item-value-label", "inspector-item-value",
	}

	comps[formID] = &a2ui.Component{
		ID:   formID,
		Type: a2ui.ComponentColumn,
		Column: &a2ui.ColumnProps{
			ClassName: "a2ui-inspector",
			Children:  a2ui.Children{ExplicitList: formChildren},
		},
	}

	return comps, formID, true
}

// ListPages returns persisted pages.
func (e *Engine) ListPages() []store.Page {
	return e.store.List()
}

// ListTemplateFiles returns deterministic template file names.
func (e *Engine) ListTemplateFiles() []string {
	return e.registry.TemplateNames()
}

// ListCompositeNames returns sorted composite names from internal/ui/composites.
func (e *Engine) ListCompositeNames() ([]string, error) {
	compositesDir := filepath.Join(e.uiDir, "composites")
	entries, err := os.ReadDir(compositesDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".jsonl") || fileName == "index.jsonl" {
			continue
		}
		names = append(names, strings.TrimSuffix(fileName, ".jsonl"))
	}
	sort.Strings(names)
	return names, nil
}

// ReadCompositeJSONL returns the raw JSONL payload for a named composite reference.
func (e *Engine) ReadCompositeJSONL(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return "", fmt.Errorf("missing composite name")
	}
	if strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") || strings.Contains(normalized, "..") {
		return "", fmt.Errorf("invalid composite name")
	}

	compositePath := filepath.Join(e.uiDir, "composites", normalized+".jsonl")
	b, err := os.ReadFile(compositePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("composite %q not found", normalized)
		}
		return "", err
	}
	return string(b), nil
}

func (e *Engine) renderUIFile(fileName string) (string, bool, error) {
	pagePath := filepath.Join(e.uiDir, fileName)
	content, err := os.ReadFile(pagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	start := time.Now()
	renderedHTML, err := e.ProcessJSONL(string(content))
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return "", false, err
	}

	renderedHTML = addRenderTimingBadge(renderedHTML, elapsed)

	return renderedHTML, true, nil
}

func addRenderTimingBadge(renderedHTML string, elapsedMillis int64) string {
	badge := fmt.Sprintf(`<p class="a2ui-render-timing" aria-label="rendering-time">Render time: %d ms</p>`, elapsedMillis)

	return appendBeforeBodyEnd(renderedHTML, badge)
}

func appendBeforeBodyEnd(renderedHTML, fragment string) string {
	lower := strings.ToLower(renderedHTML)
	idx := strings.LastIndex(lower, "</body>")
	if idx == -1 {
		return renderedHTML + fragment
	}

	return renderedHTML[:idx] + fragment + renderedHTML[idx:]
}

func addDynamicPagesSection(renderedHTML string, pages []store.Page) string {
	ids := make([]string, 0, len(pages))
	for _, p := range pages {
		id := strings.TrimSpace(p.ID)
		if id == "" || id == "ui-index-surface" {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return renderedHTML
	}

	sort.Strings(ids)

	var sb strings.Builder
	sb.WriteString(`<section class="a2ui-card" id="dynamic-pages"><h3 class="a2ui-text">Dynamic Pages (In Memory)</h3><ul class="a2ui-list">`)
	for _, id := range ids {
		escapedID := html.EscapeString(id)
		sb.WriteString(`<li><a class="a2ui-link" href="/dynamic/` + escapedID + `">` + escapedID + `</a></li>`)
	}
	sb.WriteString(`</ul></section>`)

	return appendBeforeBodyEnd(renderedHTML, sb.String())
}

func (e *Engine) loadPageFromUIFile(pageID string) (store.Page, bool, error) {
	if _, ok, err := e.renderUIFile(pageID + ".jsonl"); err != nil {
		return store.Page{}, false, err
	} else if !ok {
		return store.Page{}, false, nil
	}
	page, ok := e.store.Get(pageID)
	if !ok {
		return store.Page{}, false, fmt.Errorf("page %s was not loaded from %s", pageID, filepath.Join(e.uiDir, pageID+".jsonl"))
	}
	return page, true, nil
}

func catalogComponentFileName(component string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(component))
	if normalized == "table" {
		return "", false
	}
	for _, t := range a2ui.CatalogComponents() {
		if strings.ToLower(string(t)) == normalized {
			return filepath.Join("catalog", normalized+".jsonl"), true
		}
	}
	return "", false
}

// Health returns a static health payload.
func (e *Engine) Health() map[string]string {
	return map[string]string{"status": "ok", "version": a2ui.VersionV08}
}

// RegisterHTTPHandlers wires all required HTTP endpoints.
func RegisterHTTPHandlers(router chi.Router, eng *Engine) {
	router.Post("/render/a2ui", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed reading body", http.StatusBadRequest)
			return
		}
		out, err := eng.ProcessJSONL(string(b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if out == "" {
			out = "<!doctype html><html><body>no render output</body></html>"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(out))
	})

	router.Get("/page", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing page id", http.StatusBadRequest)
	})

	router.Get("/page/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing page id", http.StatusBadRequest)
	})

	router.Get("/page/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			http.Error(w, "missing page id", http.StatusBadRequest)
			return
		}
		p, ok := eng.store.Get(id)
		if !ok {
			var err error
			p, ok, err = eng.loadPageFromUIFile(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(p.HTML))
	})

	router.Get("/dynamic", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing dynamic page id", http.StatusBadRequest)
	})

	router.Get("/dynamic/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing dynamic page id", http.StatusBadRequest)
	})

	router.Get("/dynamic/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			http.Error(w, "missing dynamic page id", http.StatusBadRequest)
			return
		}
		p, ok := eng.store.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(p.HTML))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Send an initial comment frame so clients receive headers immediately.
		_, _ = w.Write([]byte(": connected\n\n"))
		flusher.Flush()

		ch, cancel := eng.broker.Subscribe()
		defer cancel()
		for {
			select {
			case <-r.Context().Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "event: update\ndata: %s\n\n", msg)
				flusher.Flush()
			}
		}
	})

	router.Get("/inspect/table-row", func(w http.ResponseWriter, r *http.Request) {
		pageID := strings.TrimSpace(r.URL.Query().Get("page_id"))
		tableID := strings.TrimSpace(r.URL.Query().Get("table_id"))
		rowIndexStr := strings.TrimSpace(r.URL.Query().Get("row"))
		targetID := strings.TrimSpace(r.URL.Query().Get("target_id"))
		if targetID == "" {
			targetID = "inspector-target"
		}

		if pageID == "" || tableID == "" || rowIndexStr == "" {
			http.Error(w, "missing required query parameters: page_id, table_id, row", http.StatusBadRequest)
			return
		}
		rowIndex := -1
		if _, err := fmt.Sscanf(rowIndexStr, "%d", &rowIndex); err != nil {
			http.Error(w, "invalid row index", http.StatusBadRequest)
			return
		}

		htmlOut, err := eng.InspectTableRow(pageID, tableID, rowIndex, targetID)
		if err != nil {
			if err.Error() == "page not found" || err.Error() == "table not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			if err.Error() == "row index out of range" {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(htmlOut))
	})

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","version":"` + a2ui.VersionV08 + `"}`))
	})

	router.Get("/debug", func(w http.ResponseWriter, r *http.Request) {
		if renderedHTML, ok, err := eng.renderUIFile("debug.jsonl"); err == nil && ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderedHTML))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pages := eng.ListPages()
		var sb strings.Builder
		sb.WriteString("<!doctype html><html><body><h1>Debug - Page List</h1><ul>")
		for _, p := range pages {
			sb.WriteString(fmt.Sprintf("<li><a href=\"/page/%s\">%s</a> - updated %s - ready=%v ended=%v</li>",
				html.EscapeString(p.ID), html.EscapeString(p.ID), p.UpdatedAt.Format(time.RFC3339), p.Ready, p.Ended))
		}
		sb.WriteString("</ul></body></html>")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(sb.String()))
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if renderedHTML, ok, err := eng.renderUIFile("index.jsonl"); err == nil && ok {
			renderedHTML = addDynamicPagesSection(renderedHTML, eng.ListPages())
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderedHTML))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	})

	router.Get("/catalog", func(w http.ResponseWriter, r *http.Request) {
		if renderedHTML, ok, err := eng.renderUIFile(filepath.Join("catalog", "index.jsonl")); err == nil && ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderedHTML))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	})

	router.Get("/catalog/{component}", func(w http.ResponseWriter, r *http.Request) {
		component := strings.TrimSpace(chi.URLParam(r, "component"))
		if component == "" {
			http.NotFound(w, r)
			return
		}
		fileName, ok := catalogComponentFileName(component)
		if !ok {
			http.NotFound(w, r)
			return
		}
		renderedHTML, ok, err := eng.renderUIFile(fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderedHTML))
	})

	router.Get("/composites", func(w http.ResponseWriter, r *http.Request) {
		if renderedHTML, ok, err := eng.renderUIFile(filepath.Join("composites", "index.jsonl")); err == nil && ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderedHTML))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	})

	router.Get("/composites/{name}", func(w http.ResponseWriter, r *http.Request) {
		compositeName := strings.TrimSpace(chi.URLParam(r, "name"))
		if compositeName == "" {
			http.NotFound(w, r)
			return
		}
		fileName := filepath.Join("composites", compositeName+".jsonl")
		renderedHTML, ok, err := eng.renderUIFile(fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderedHTML))
	})
}

// buildInspectorComponents builds an adjacency-list component map for the table row inspector.
func buildInspectorComponents(targetID string, headers, values []string) (map[string]*a2ui.Component, string) {
	comps := make(map[string]*a2ui.Component)

	literalStr := func(s string) *a2ui.BoundValue { return &a2ui.BoundValue{LiteralString: &s} }

	titleID := targetID + "-title"
	dividerID := targetID + "-divider"
	comps[titleID] = &a2ui.Component{ID: titleID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr("Selected Row Inspector"), UsageHint: "h3"}}
	comps[dividerID] = &a2ui.Component{ID: dividerID, Type: a2ui.ComponentDivider, Divider: &a2ui.DividerProps{Axis: "horizontal"}}

	colChildren := []string{titleID, dividerID}
	for i, header := range headers {
		value := ""
		if i < len(values) {
			value = values[i]
		}
		labelID := fmt.Sprintf("%s-label-%d", targetID, i)
		valueID := fmt.Sprintf("%s-value-%d", targetID, i)
		rowID := fmt.Sprintf("%s-field-%d", targetID, i)
		comps[labelID] = &a2ui.Component{ID: labelID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(header + ":")}}
		comps[valueID] = &a2ui.Component{ID: valueID, Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalStr(value)}}
		comps[rowID] = &a2ui.Component{
			ID:   rowID,
			Type: a2ui.ComponentRow,
			Row:  &a2ui.RowProps{Children: a2ui.Children{ExplicitList: []string{labelID, valueID}}, Distribution: "start"},
		}
		colChildren = append(colChildren, rowID)
	}

	comps[targetID] = &a2ui.Component{
		ID:     targetID,
		Type:   a2ui.ComponentColumn,
		Column: &a2ui.ColumnProps{Children: a2ui.Children{ExplicitList: colChildren}},
	}
	return comps, targetID
}

func resolveFile(filename string) (string, error) {
	candidates := []string{
		filename,
		filepath.Join(".", filename),
		filepath.Join("..", filename),
		filepath.Join("../..", filename),
		filepath.Join("github.com/dmundt/a2ui-go", "..", filename),
		filepath.Join("github.com/dmundt/a2ui-go/cmd/server", "..", "..", filename),
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, filename))
		candidates = append(candidates, filepath.Join(exeDir, "..", filename))
		candidates = append(candidates, filepath.Join(exeDir, "../..", filename))
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, filename))
		candidates = append(candidates, filepath.Join(cwd, "..", filename))
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find file %q", filename)
}
