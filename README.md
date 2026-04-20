# A2UI Go Renderer (Experimental)

[![Go Test and Validation](https://github.com/dmundt/a2ui-go/actions/workflows/go-test-validation.yml/badge.svg)](https://github.com/dmundt/a2ui-go/actions/workflows/go-test-validation.yml)

This repository is an experimental renderer used to test key concepts around A2UI v0.8, Go, html/template, and dynamic rendering workflows.

It is intentionally focused on learning and validation over production hardening.

## Purpose

- Validate A2UI v0.8 schema-driven rendering in Go.
- Exercise deterministic component-to-template rendering without reflection.
- Test dynamic server-side rendering flows over HTTP and SSE.
- Expose the same rendering workflow through MCP stdio tools.

## Project Description

The project accepts A2UI v0.8 JSONL messages, validates them, applies page state transitions (begin/update/end), and renders HTML from a template registry.

It includes:

- HTTP endpoints for rendering, page retrieval, streaming updates, catalog pages, and health checks.
- A built-in component catalog and composite examples under internal UI JSONL files.
- An MCP stdio server with the tools:
	- a2ui_discovery
	- a2ui_describe_tool
	- a2ui_capabilities
	- a2ui_validate_jsonl
	- a2ui_apply_jsonl
	- a2ui_render
	- a2ui_list_pages
	- a2ui_list_templates
	- a2ui_list_surfaces
	- a2ui_get_surface
	- a2ui_get_surface_model
	- a2ui_get_surface_components
	- a2ui_render_surface
	- a2ui_create_surface
	- a2ui_delete_surface
	- a2ui_reset_runtime
	- a2ui_inspect_table_row
	- a2ui_list_composites
	- a2ui_fetch_composite
	- a2ui_examples_list
	- a2ui_example_get
	- a2ui_health

## How It Works

1. JSONL input is submitted through POST /render/a2ui or the MCP render tool.
2. Input is validated against the A2UI v0.8 model.
3. The engine updates in-memory page state.
4. The renderer resolves each component type to a fixed Go template.
5. Rendered HTML is returned and published to SSE subscribers.

## Structure

- a2ui/: A2UI model, catalog metadata, and validation.
- cmd/server/: HTTP + MCP entrypoint.
- internal/engine/: Runtime pipeline, handler wiring, and page loading.
- internal/store/: In-memory page store.
- internal/stream/: SSE broker.
- internal/ui/: JSONL pages used by index, debug, catalog, and composite routes.
- mcp/: MCP tool handlers and stdio server registration.
- renderer/: Template registry and render logic.
- renderer/templates/: Component/page templates.
- static/: Site CSS.

## Assumptions

- Input follows A2UI v0.8 message conventions.
- Rendering state is in-memory and ephemeral (no persistence layer).
- Template names and component mappings stay explicit and deterministic.
- This is a local/dev experimentation project, not a production deployment target.

## Build

```bash
go mod tidy
go build ./cmd/server
```

## Run

Start the HTTP server (localhost:8080):

```bash
go run ./cmd/server
```

Quick checks:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/catalog
curl http://localhost:8080/composites
```

Render a JSONL payload file:

```bash
curl -X POST http://localhost:8080/render/a2ui \
	-H "Content-Type: application/jsonl" \
	--data-binary @internal/ui/index.jsonl
```

## MCP Stdio Mode

```bash
ENABLE_MCP_STDIO=1 go run ./cmd/server
```

The process serves MCP JSON-RPC over stdio and exposes canonical underscore tool names.

Communication and UI specification are strictly A2UI v0.8:

- Communication format is A2UI JSONL v0.8.
- UI specification is A2UI v0.8 component/message model.
- Incompatible payloads are rejected by validation/apply tools.

All MCP tools return a structured JSON envelope in text content:

```json
{
	"success": true,
	"data": {"...": "..."},
	"warnings": ["optional warning"]
}
```

On error:

```json
{
	"success": false,
	"error": {
		"code": "ERROR_CODE",
		"message": "Human-readable message",
		"details": {"optional": "context"}
	}
}
```

## MCP Client Quickstart

Once MCP Stdio is running, clients can invoke tools over JSON-RPC. Here are key v2 workflows:

### 1. Discover Capabilities

Get protocol version, supported features, component types, and full tool list:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"a2ui_capabilities","arguments":{}}}
```

Response (truncated):

```json
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"protocolVersion\":\"0.8\",\"server\":\"github.com/dmundt/a2ui-go\",\"features\":[\"render\",\"validate\",\"surfaces\",\"composites\",\"examples\",\"inspector\",\"runtime-lifecycle\"],\"tools\":[\"a2ui_capabilities\",\"a2ui_health\",...]}}"} ]}}
```

### 2. Validate JSONL Without State Mutation

Check syntax and semantics without rendering:

```json
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"a2ui_validate_jsonl","arguments":{"jsonl":"{\"dataModelUpdate\":{\"surfaceId\":\"demo\",\"path\":\"/\",\"contents\":[{\"key\":\"name\",\"valueString\":\"Hello\"}]}}\\n{\"beginRendering\":{\"surfaceId\":\"demo\",\"root\":\"root\"}}"}}}
```

Response:

```json
{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"valid\":true,\"validatedLines\":2}}"}]}}
```

### 3. Apply JSONL and Render

Validate, apply, and render in one call:

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"a2ui_apply_jsonl","arguments":{"jsonl":"{\"dataModelUpdate\":{\"surfaceId\":\"demo\",\"path\":\"/\",\"contents\":[{\"key\":\"greeting\",\"valueString\":\"Hello World\"}]}}\\n{\"surfaceUpdate\":{\"surfaceId\":\"demo\",\"components\":[{\"id\":\"root\",\"component\":{\"Text\":{\"text\":{\"path\":\"/greeting\"}}}}]}}\\n{\"beginRendering\":{\"surfaceId\":\"demo\",\"root\":\"root\"}}"}}}
```

Response (html truncated):

```json
{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"html\":\"<!doctype html>...<p id=\\\"root\\\" class=\\\"a2ui-text\\\">Hello World</p>...</html>\"}}"}]}}
```

### 4. List Examples

Discover available demo JSONL payloads:

```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"a2ui_examples_list","arguments":{}}}
```

Response:

```json
{"jsonrpc":"2.0","id":4,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"examples\":[\"complex-demo\",\"dynamic-mcp\",\"mail-app-demo\",\"mail-app-pro-demo\",\"user-management-demo\"],\"count\":5}}"}]}}
```

### 5. Get Example JSONL

Fetch one example payload:

```json
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"a2ui_example_get","arguments":{"name":"user-management-demo"}}}
```

Response (jsonl payload truncated):

```json
{"jsonrpc":"2.0","id":5,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"name\":\"user-management-demo\",\"jsonl\":\"{\\\"dataModelUpdate\\\":{...}}\\n...\"}}"}]}}
```

### 6. Create and List In-Memory Surfaces

Create an empty surface and list all surfaces:

```json
{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"a2ui_create_surface","arguments":{"surface_id":"my-demo"}}}
```

```json
{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"a2ui_list_surfaces","arguments":{}}}
```

Response:

```json
{"jsonrpc":"2.0","id":7,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"ids\":[\"my-demo\"],\"count\":1}}"}]}}
```

### 7. Get Surface Model and Components

Inspect one surface state:

```json
{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"a2ui_get_surface_model","arguments":{"surface_id":"my-demo"}}}
```

```json
{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"a2ui_get_surface_components","arguments":{"surface_id":"my-demo"}}}
```

Responses show current data model and component tree metadata.

### 8. Reset Runtime

Clear all in-memory surfaces:

```json
{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"a2ui_reset_runtime","arguments":{}}}
```

Response:

```json
{"jsonrpc":"2.0","id":10,"result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"removed\":1}}"}]}}
```

## Integration Pattern

Effective MCP clients follow this iterative loop:

1. **Discover** capabilities and examples
2. **Validate** JSONL payloads before applying
3. **Apply** validated updates to create/update in-memory surfaces
4. **Inspect** surface model/components to diagnose state
5. **Reset** runtime between sessions or when needed
6. **Render** surfaces on-demand for preview or inspection

## User Management Workbench via MCP

The repository now includes a new interface payload at `examples/user-management-workbench.jsonl`.

Use the updated MCP methods to inject this interface into dynamic views:

1. Discover contract metadata (including A2UI v0.8 requirement):

```json
{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"a2ui_discovery","arguments":{}}}
```

2. Fetch the example payload:

```json
{"jsonrpc":"2.0","id":22,"method":"tools/call","params":{"name":"a2ui_example_get","arguments":{"name":"user-management-workbench"}}}
```

3. (Optional) Create the surface first for deterministic lifecycle control:

```json
{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"a2ui_create_surface","arguments":{"surface_id":"dynamic-user-mgmt-workbench"}}}
```

4. Apply JSONL to runtime (injects view state/components):

```json
{"jsonrpc":"2.0","id":24,"method":"tools/call","params":{"name":"a2ui_apply_jsonl","arguments":{"jsonl":"<jsonl from step 2>"}}}
```

5. Verify runtime model and component graph:

```json
{"jsonrpc":"2.0","id":25,"method":"tools/call","params":{"name":"a2ui_get_surface_model","arguments":{"surface_id":"dynamic-user-mgmt-workbench"}}}
```

```json
{"jsonrpc":"2.0","id":26,"method":"tools/call","params":{"name":"a2ui_get_surface_components","arguments":{"surface_id":"dynamic-user-mgmt-workbench"}}}
```

6. Render on demand (if needed):

```json
{"jsonrpc":"2.0","id":27,"method":"tools/call","params":{"name":"a2ui_render_surface","arguments":{"surface_id":"dynamic-user-mgmt-workbench"}}}
```

7. Open the injected dynamic route in browser:

```text
http://localhost:8080/dynamic/dynamic-user-mgmt-workbench
```

## Google Org Chart Sample via MCP

Source reference:
https://github.com/google/A2UI/blob/main/samples/agent/adk/custom-components-example/examples/0.8/org_chart.json

This repo includes an adapted payload at `examples/org-chart-google-sample.jsonl`.
It keeps the original org chart data but maps rendering to components supported by this renderer.

Inject it into runtime with MCP tools:

1. Fetch payload:

```json
{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"a2ui_example_get","arguments":{"name":"org-chart-google-sample"}}}
```

2. Apply payload:

```json
{"jsonrpc":"2.0","id":32,"method":"tools/call","params":{"name":"a2ui_apply_jsonl","arguments":{"jsonl":"<jsonl from step 1>"}}}
```

3. Open dynamic page:

```text
http://localhost:8080/dynamic/dynamic-org-chart-google
```

## Google Contact List Sample via MCP

Source reference:
https://github.com/google/A2UI/blob/main/samples/agent/adk/custom-components-example/examples/0.8/contact_list.json

This repo includes an adapted payload at `examples/contact-list-google-sample.jsonl`.
It preserves the original contact data but maps rendering to components supported by this renderer.

Inject it into runtime with MCP tools:

1. Fetch payload:

```json
{"jsonrpc":"2.0","id":41,"method":"tools/call","params":{"name":"a2ui_example_get","arguments":{"name":"contact-list-google-sample"}}}
```

2. Apply payload:

```json
{"jsonrpc":"2.0","id":42,"method":"tools/call","params":{"name":"a2ui_apply_jsonl","arguments":{"jsonl":"<jsonl from step 1>"}}}
```

3. Open dynamic page:

```text
http://localhost:8080/dynamic/dynamic-contact-list-google
```

## Asset Management Database App via MCP

A complete, multi-tabbed asset management application with header, footer, and three main sections:

**Features:**
- **Header & Footer** layout with app title and status
- **Items Tab:** Asset inventory with 9 properties (ID, name, category, status, location, serial number, purchase date, value)
- **Tags Tab:** Asset classification with 4 properties (ID, name, color, description)
- **Locations Tab:** Facility management with 6 properties (ID, name, building, floor, capacity, asset count)
- **CRUD Operations:** Create/Inspect/Edit/Delete actions for each entity type
- **Inline Inspectors:** Property editors for each selected object
- **Data Tables:** Multi-column views with action buttons
- **Sample Data:** 3 items, 3 tags, 3 locations pre-populated

Inject it via MCP:

1. Fetch payload:

```json
{"jsonrpc":"2.0","id":51,"method":"tools/call","params":{"name":"a2ui_example_get","arguments":{"name":"asset-management-app"}}}
```

2. Apply payload:

```json
{"jsonrpc":"2.0","id":52,"method":"tools/call","params":{"name":"a2ui_apply_jsonl","arguments":{"jsonl":"<jsonl from step 1>"}}}
```

3. Open dynamic page:

```text
http://localhost:8080/dynamic/dynamic-asset-management-app
```

## Test

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See LICENSE for details.
