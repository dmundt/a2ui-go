package mcp

import (
	"context"
	"os"
	"strconv"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var canonicalToolNames = []string{
	"a2ui_discovery",
	"a2ui_describe_tool",
	"a2ui_capabilities",
	"a2ui_health",
	"a2ui_validate_jsonl",
	"a2ui_apply_jsonl",
	"a2ui_render",
	"a2ui_list_pages",
	"a2ui_list_templates",
	"a2ui_list_surfaces",
	"a2ui_get_surface",
	"a2ui_get_surface_model",
	"a2ui_get_surface_components",
	"a2ui_render_surface",
	"a2ui_create_surface",
	"a2ui_delete_surface",
	"a2ui_reset_runtime",
	"a2ui_inspect_table_row",
	"a2ui_list_composites",
	"a2ui_fetch_composite",
	"a2ui_examples_list",
	"a2ui_example_get",
}

func toolNames() []string {
	out := make([]string, 0, len(canonicalToolNames))
	out = append(out, canonicalToolNames...)
	return out
}

// StartStdio starts MCP JSON-RPC over stdio.
func StartStdio(ctx context.Context, h *Handlers) error {
	s := server.NewMCPServer("github.com/dmundt/a2ui-go", "0.8.0")
	textResult := func(out string, err error) (*mcpgo.CallToolResult, error) {
		if err != nil {
			e, merr := envelopeErr("INTERNAL_ERROR", err.Error(), nil)
			if merr != nil {
				return nil, merr
			}
			return mcpgo.NewToolResultText(e), nil
		}
		return mcpgo.NewToolResultText(out), nil
	}

	registerRenderTool := func(name, description string, apply bool) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription(description),
				mcpgo.WithString("jsonl", mcpgo.Required(), mcpgo.Description("A2UI v0.8 JSONL payload")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				jsonl, err := req.RequireString("jsonl")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "jsonl"}))
				}
				if apply {
					return textResult(h.ApplyJSONL(jsonl))
				}
				return textResult(h.ValidateJSONL(jsonl))
			},
		)
	}

	registerRenderCompatTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Render A2UI v0.8 JSONL to HTML"),
				mcpgo.WithString("jsonl", mcpgo.Required(), mcpgo.Description("A2UI v0.8 JSONL payload")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				jsonl, err := req.RequireString("jsonl")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "jsonl"}))
				}
				return textResult(h.Render(jsonl))
			},
		)
	}

	registerPagesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List stored pages")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ListPages())
			},
		)
	}

	registerTemplatesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List template files")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ListTemplates())
			},
		)
	}

	registerCapabilitiesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("Get A2UI MCP capabilities, supported features, and schema hints")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.Capabilities())
			},
		)
	}

	registerDiscoveryTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("Get full machine-readable discovery profile for this MCP server")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.Discovery())
			},
		)
	}

	registerDescribeTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Describe one MCP tool contract by canonical name"),
				mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Canonical tool name")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				toolName, err := req.RequireString("name")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "name"}))
				}
				return textResult(h.DescribeTool(toolName))
			},
		)
	}

	registerListSurfacesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List in-memory A2UI surface ids")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ListSurfaces())
			},
		)
	}

	registerGetSurfaceTool := func(name, description string, getter func(surfaceID string) (string, error)) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription(description),
				mcpgo.WithString("surface_id", mcpgo.Required(), mcpgo.Description("Surface id")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				surfaceID, err := req.RequireString("surface_id")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "surface_id"}))
				}
				return textResult(getter(surfaceID))
			},
		)
	}

	registerCreateDeleteSurfaceTool := func(name, description string, op func(surfaceID string) (string, error)) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription(description),
				mcpgo.WithString("surface_id", mcpgo.Required(), mcpgo.Description("Surface id")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				surfaceID, err := req.RequireString("surface_id")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "surface_id"}))
				}
				return textResult(op(surfaceID))
			},
		)
	}

	registerResetRuntimeTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("Reset in-memory runtime state by deleting all surfaces")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ResetRuntime())
			},
		)
	}

	registerInspectRowTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Render inline inspector HTML for one table row"),
				mcpgo.WithString("page_id", mcpgo.Required(), mcpgo.Description("Page id containing the table")),
				mcpgo.WithString("table_id", mcpgo.Required(), mcpgo.Description("Table component id")),
				mcpgo.WithString("row", mcpgo.Required(), mcpgo.Description("Zero-based row index")),
				mcpgo.WithString("target_id", mcpgo.Description("Optional root id for generated inspector")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				pageID, err := req.RequireString("page_id")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "page_id"}))
				}
				tableID, err := req.RequireString("table_id")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "table_id"}))
				}
				rowText, err := req.RequireString("row")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "row"}))
				}
				row, convErr := strconv.Atoi(rowText)
				if convErr != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", "row must be an integer", map[string]interface{}{"row": rowText}))
				}
				targetID := req.GetString("target_id", "")
				return textResult(h.InspectTableRow(pageID, tableID, row, targetID))
			},
		)
	}

	registerListCompositesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List composite reference names")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ListComposites())
			},
		)
	}

	registerFetchCompositeTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Fetch one composite reference JSONL payload"),
				mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Composite name, e.g. header")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				name, err := req.RequireString("name")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "name"}))
				}
				return textResult(h.FetchComposite(name))
			},
		)
	}

	registerExamplesListTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List JSONL examples from the examples directory")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.ExamplesList())
			},
		)
	}

	registerExampleGetTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Fetch one example JSONL payload by name"),
				mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Example name without .jsonl")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				name, err := req.RequireString("name")
				if err != nil {
					return textResult(envelopeErr("INVALID_ARGUMENT", err.Error(), map[string]interface{}{"field": "name"}))
				}
				return textResult(h.ExampleGet(name))
			},
		)
	}

	registerHealthTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("Health check")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				return textResult(h.Health())
			},
		)
	}

	registerDiscoveryTool("a2ui_discovery")
	registerDescribeTool("a2ui_describe_tool")
	registerCapabilitiesTool("a2ui_capabilities")
	registerHealthTool("a2ui_health")
	registerRenderTool("a2ui_validate_jsonl", "Validate A2UI v0.8 JSONL without mutating runtime state", false)
	registerRenderTool("a2ui_apply_jsonl", "Apply A2UI v0.8 JSONL to runtime and return final render output", true)
	registerRenderCompatTool("a2ui_render")
	registerPagesTool("a2ui_list_pages")
	registerTemplatesTool("a2ui_list_templates")
	registerListSurfacesTool("a2ui_list_surfaces")
	registerGetSurfaceTool("a2ui_get_surface", "Get one in-memory surface including components and model", h.GetSurface)
	registerGetSurfaceTool("a2ui_get_surface_model", "Get data model for one in-memory surface", h.GetSurfaceModel)
	registerGetSurfaceTool("a2ui_get_surface_components", "Get component metadata for one in-memory surface", h.GetSurfaceComponents)
	registerGetSurfaceTool("a2ui_render_surface", "Render one in-memory surface by id", h.RenderSurface)
	registerCreateDeleteSurfaceTool("a2ui_create_surface", "Create an empty in-memory surface", h.CreateSurface)
	registerCreateDeleteSurfaceTool("a2ui_delete_surface", "Delete one in-memory surface", h.DeleteSurface)
	registerResetRuntimeTool("a2ui_reset_runtime")
	registerInspectRowTool("a2ui_inspect_table_row")
	registerListCompositesTool("a2ui_list_composites")
	registerFetchCompositeTool("a2ui_fetch_composite")
	registerExamplesListTool("a2ui_examples_list")
	registerExampleGetTool("a2ui_example_get")

	stdio := server.NewStdioServer(s)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}
