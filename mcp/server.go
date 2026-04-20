package mcp

import (
	"context"
	"os"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// StartStdio starts MCP JSON-RPC over stdio.
func StartStdio(ctx context.Context, h *Handlers) error {
	s := server.NewMCPServer("github.com/dmundt/a2ui-go", "0.8.0")

	registerRenderTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(
				name,
				mcpgo.WithDescription("Render A2UI v0.8 JSONL to HTML"),
				mcpgo.WithString("jsonl", mcpgo.Required(), mcpgo.Description("A2UI v0.8 JSONL payload")),
			),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				jsonl, err := req.RequireString("jsonl")
				if err != nil {
					return nil, err
				}
				html, err := h.Render(jsonl)
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(html), nil
			},
		)
	}

	registerPagesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List stored pages")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				out, err := h.ListPages()
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(out), nil
			},
		)
	}

	registerTemplatesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List template files")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				out, err := h.ListTemplates()
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(out), nil
			},
		)
	}

	registerListCompositesTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("List composite reference names")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				out, err := h.ListComposites()
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(out), nil
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
					return nil, err
				}
				out, err := h.FetchComposite(name)
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(out), nil
			},
		)
	}

	registerHealthTool := func(name string) {
		s.AddTool(
			mcpgo.NewTool(name, mcpgo.WithDescription("Health check")),
			func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
				out, err := h.Health()
				if err != nil {
					return nil, err
				}
				return mcpgo.NewToolResultText(out), nil
			},
		)
	}

	registerRenderTool("a2ui_render")
	registerPagesTool("a2ui_list_pages")
	registerTemplatesTool("a2ui_list_templates")
	registerListCompositesTool("a2ui_list_composites")
	registerFetchCompositeTool("a2ui_fetch_composite")
	registerHealthTool("a2ui_health")

	stdio := server.NewStdioServer(s)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}
