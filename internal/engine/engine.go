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
	"strings"
	"time"

	"github.com/dmundt/au2ui-go/a2ui"
	"github.com/dmundt/au2ui-go/internal/store"
	"github.com/dmundt/au2ui-go/internal/stream"
	"github.com/dmundt/au2ui-go/renderer"
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

// ListPages returns persisted pages.
func (e *Engine) ListPages() []store.Page {
	return e.store.List()
}

// ListTemplateFiles returns deterministic template file names.
func (e *Engine) ListTemplateFiles() []string {
	return e.registry.TemplateNames()
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
	renderedHTML, err := e.ProcessJSONL(string(content))
	if err != nil {
		return "", false, err
	}
	return renderedHTML, true, nil
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
func RegisterHTTPHandlers(mux *http.ServeMux, eng *Engine) {
	mux.HandleFunc("/render/a2ui", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
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

	mux.HandleFunc("/page/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/page/")
		if strings.TrimSpace(id) == "" {
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

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

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

	mux.HandleFunc("/inspect/table-row", func(w http.ResponseWriter, r *http.Request) {
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

		page, ok := eng.store.Get(pageID)
		if !ok {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		tableComp := page.Components[tableID]
		if tableComp == nil || tableComp.Table == nil {
			http.Error(w, "table not found", http.StatusNotFound)
			return
		}
		if rowIndex < 0 || rowIndex >= len(tableComp.Table.Rows) {
			http.Error(w, "row index out of range", http.StatusBadRequest)
			return
		}

		inspectorComps, rootID := buildInspectorComponents(targetID, tableComp.Table.Headers, tableComp.Table.Rows[rowIndex])
		htmlOut, err := eng.renderer.RenderSurface(inspectorComps, page.DataModel, rootID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(htmlOut))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","version":"` + a2ui.VersionV08 + `"}`))
	})

	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if renderedHTML, ok, err := eng.renderUIFile("index.jsonl"); err == nil && ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(renderedHTML))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
			http.NotFound(w, r)
			return
		}
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

	mux.HandleFunc("/catalog/", func(w http.ResponseWriter, r *http.Request) {
		component := strings.TrimPrefix(r.URL.Path, "/catalog/")
		if strings.TrimSpace(component) == "" || strings.Contains(component, "/") {
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

	mux.HandleFunc("/composites", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/composites" {
			http.NotFound(w, r)
			return
		}
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

	mux.HandleFunc("/composites/", func(w http.ResponseWriter, r *http.Request) {
		compositeName := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/composites/"))
		if compositeName == "" || strings.Contains(compositeName, "/") {
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
		filepath.Join("github.com/dmundt/au2ui-go", "..", filename),
		filepath.Join("github.com/dmundt/au2ui-go/cmd/server", "..", "..", filename),
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
