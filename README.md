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
	- a2ui_render
	- a2ui_list_pages
	- a2ui_list_templates
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

The process serves MCP JSON-RPC over stdio and exposes both underscore and dotted tool aliases.

## Test

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See LICENSE for details.
