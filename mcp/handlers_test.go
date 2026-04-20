package mcp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dmundt/a2ui-go/internal/engine"
	"github.com/dmundt/a2ui-go/internal/store"
	"github.com/dmundt/a2ui-go/internal/stream"
	"github.com/dmundt/a2ui-go/mcp"
	"github.com/dmundt/a2ui-go/renderer"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, raw string) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("decode envelope: %v\nraw=%s", err, raw)
	}
	return env
}

func TestHandlersRenderAndList(t *testing.T) {
	reg, err := renderer.NewRegistry("../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../internal/ui")
	h := mcp.NewHandlers(eng, reg)

	jsonl := `{"surfaceUpdate":{"surfaceId":"p1","components":[{"id":"root","component":{"Text":{"text":{"literalString":"Hello"}}}}]}}
{"beginRendering":{"surfaceId":"p1","root":"root"}}`
	html, err := h.Render(jsonl)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	renderEnv := decodeEnvelope(t, html)
	if !renderEnv.Success {
		t.Fatalf("render failed envelope: %s", html)
	}
	if !strings.Contains(string(renderEnv.Data), "Hello") {
		t.Fatalf("render output missing text")
	}

	pages, err := h.ListPages()
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	pagesEnv := decodeEnvelope(t, pages)
	if !pagesEnv.Success {
		t.Fatalf("list pages failed envelope: %s", pages)
	}
	if !strings.Contains(string(pagesEnv.Data), "\"ID\":\"p1\"") {
		t.Fatalf("unexpected page list: %s", pages)
	}

	templates, err := h.ListTemplates()
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	templatesEnv := decodeEnvelope(t, templates)
	if !templatesEnv.Success {
		t.Fatalf("list templates failed envelope: %s", templates)
	}
	if !strings.Contains(string(templatesEnv.Data), "page.html") {
		t.Fatalf("unexpected templates output: %s", templates)
	}

	health, err := h.Health()
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	healthEnv := decodeEnvelope(t, health)
	if !healthEnv.Success {
		t.Fatalf("health failed envelope: %s", health)
	}
	if !strings.Contains(string(healthEnv.Data), `"status":"ok"`) || !strings.Contains(string(healthEnv.Data), `"version":"0.8"`) {
		t.Fatalf("unexpected health output: %s", health)
	}
}

func TestHandlersValidateAndLifecycle(t *testing.T) {
	reg, err := renderer.NewRegistry("../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../internal/ui")
	h := mcp.NewHandlers(eng, reg)

	caps, err := h.Capabilities()
	if err != nil {
		t.Fatalf("capabilities: %v", err)
	}
	if !strings.Contains(caps, `"a2ui_validate_jsonl"`) || !strings.Contains(caps, `"a2ui_reset_runtime"`) {
		t.Fatalf("capabilities missing expected tools: %s", caps)
	}
	if !strings.Contains(caps, `"protocolName":"A2UI"`) || !strings.Contains(caps, `"version":"0.8"`) || !strings.Contains(caps, `"format":"jsonl"`) {
		t.Fatalf("capabilities missing required A2UI v0.8 spec contract: %s", caps)
	}

	discovery, err := h.Discovery()
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	if !strings.Contains(discovery, `"uiSpecification"`) || !strings.Contains(discovery, `"incompatiblePolicy":"reject"`) {
		t.Fatalf("discovery missing strict UI spec contract: %s", discovery)
	}

	invalid, err := h.ValidateJSONL(`{"surfaceUpdate":`) // malformed
	if err != nil {
		t.Fatalf("validate invalid: %v", err)
	}
	invalidEnv := decodeEnvelope(t, invalid)
	if invalidEnv.Success || invalidEnv.Error == nil || invalidEnv.Error.Code != "DECODE_ERROR" {
		t.Fatalf("expected decode error envelope, got: %s", invalid)
	}

	create, err := h.CreateSurface("demo-surface")
	if err != nil {
		t.Fatalf("create surface: %v", err)
	}
	if !decodeEnvelope(t, create).Success {
		t.Fatalf("create surface failed: %s", create)
	}

	list, err := h.ListSurfaces()
	if err != nil {
		t.Fatalf("list surfaces: %v", err)
	}
	if !strings.Contains(list, `"demo-surface"`) {
		t.Fatalf("surface missing from list: %s", list)
	}

	get, err := h.GetSurface("demo-surface")
	if err != nil {
		t.Fatalf("get surface: %v", err)
	}
	if !decodeEnvelope(t, get).Success {
		t.Fatalf("get surface failed: %s", get)
	}

	del, err := h.DeleteSurface("demo-surface")
	if err != nil {
		t.Fatalf("delete surface: %v", err)
	}
	if !decodeEnvelope(t, del).Success {
		t.Fatalf("delete surface failed: %s", del)
	}

	reset, err := h.ResetRuntime()
	if err != nil {
		t.Fatalf("reset runtime: %v", err)
	}
	if !decodeEnvelope(t, reset).Success {
		t.Fatalf("reset runtime failed: %s", reset)
	}
}

func TestHandlersExamples(t *testing.T) {
	reg, err := renderer.NewRegistry("../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../internal/ui")
	h := mcp.NewHandlers(eng, reg)

	list, err := h.ExamplesList()
	if err != nil {
		t.Fatalf("examples list: %v", err)
	}
	if !decodeEnvelope(t, list).Success {
		t.Fatalf("examples list failed: %s", list)
	}

	get, err := h.ExampleGet("user-management-demo")
	if err != nil {
		t.Fatalf("example get: %v", err)
	}
	if !strings.Contains(get, "dynamic-user-mgmt-demo") {
		t.Fatalf("unexpected example payload: %s", get)
	}
}
