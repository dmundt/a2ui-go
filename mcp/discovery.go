package mcp

import "github.com/dmundt/a2ui-go/a2ui"

// toolArgSpec describes one tool argument for machine-readable discovery.
type toolArgSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// toolSpec describes one MCP tool contract for clients.
type toolSpec struct {
	Name        string         `json:"name"`
	Summary     string         `json:"summary"`
	Arguments   []toolArgSpec  `json:"arguments,omitempty"`
	Returns     string         `json:"returns"`
	Mutable     bool           `json:"mutable"`
	Idempotent  bool           `json:"idempotent"`
	Category    string         `json:"category"`
	ErrorCodes  []string       `json:"errorCodes,omitempty"`
	ExampleCall map[string]any `json:"exampleCall,omitempty"`
}

func discoveryToolSpecs() []toolSpec {
	return []toolSpec{
		{
			Name:       "a2ui_discovery",
			Summary:    "Return full machine-readable discovery document for this server",
			Returns:    "Envelope success.data with server/profile/features/tools/workflows",
			Mutable:    false,
			Idempotent: true,
			Category:   "discovery",
		},
		{
			Name:       "a2ui_describe_tool",
			Summary:    "Return one tool contract by tool name",
			Arguments:  []toolArgSpec{{Name: "name", Type: "string", Required: true, Description: "Canonical tool name"}},
			Returns:    "Envelope success.data with one tool contract",
			Mutable:    false,
			Idempotent: true,
			Category:   "discovery",
			ErrorCodes: []string{"INVALID_ARGUMENT", "TOOL_NOT_FOUND"},
			ExampleCall: map[string]any{
				"name": "a2ui_describe_tool",
				"arguments": map[string]any{
					"name": "a2ui_apply_jsonl",
				},
			},
		},
		{
			Name:       "a2ui_capabilities",
			Summary:    "Return high-level capabilities and compact tool list",
			Returns:    "Envelope success.data with protocolVersion/server/features/tools/components",
			Mutable:    false,
			Idempotent: true,
			Category:   "discovery",
		},
		{
			Name:       "a2ui_health",
			Summary:    "Health check",
			Returns:    "Envelope success.data with status/version",
			Mutable:    false,
			Idempotent: true,
			Category:   "diagnostics",
		},
		{
			Name:    "a2ui_validate_jsonl",
			Summary: "Validate A2UI JSONL syntax and semantics without mutating runtime",
			Arguments: []toolArgSpec{
				{Name: "jsonl", Type: "string", Required: true, Description: "A2UI v0.8 JSONL payload"},
			},
			Returns:    "Envelope success.data with valid/validatedLines",
			Mutable:    false,
			Idempotent: true,
			Category:   "authoring",
			ErrorCodes: []string{"INVALID_ARGUMENT", "DECODE_ERROR", "VALIDATION_ERROR", "SCAN_ERROR"},
		},
		{
			Name:    "a2ui_apply_jsonl",
			Summary: "Validate + apply JSONL updates and return final rendered HTML",
			Arguments: []toolArgSpec{
				{Name: "jsonl", Type: "string", Required: true, Description: "A2UI v0.8 JSONL payload"},
			},
			Returns:    "Envelope success.data with html",
			Mutable:    true,
			Idempotent: false,
			Category:   "authoring",
			ErrorCodes: []string{"INVALID_ARGUMENT", "APPLY_FAILED"},
		},
		{
			Name:    "a2ui_render",
			Summary: "Compatibility render endpoint for JSONL -> HTML",
			Arguments: []toolArgSpec{
				{Name: "jsonl", Type: "string", Required: true, Description: "A2UI v0.8 JSONL payload"},
			},
			Returns:    "Envelope success.data with html",
			Mutable:    true,
			Idempotent: false,
			Category:   "authoring",
			ErrorCodes: []string{"INVALID_ARGUMENT", "INVALID_JSONL"},
		},
		{
			Name:       "a2ui_list_pages",
			Summary:    "List current in-memory pages",
			Returns:    "Envelope success.data array of page records",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
		},
		{
			Name:       "a2ui_list_templates",
			Summary:    "List known template files",
			Returns:    "Envelope success.data array of template names",
			Mutable:    false,
			Idempotent: true,
			Category:   "diagnostics",
		},
		{
			Name:       "a2ui_list_surfaces",
			Summary:    "List in-memory surface ids",
			Returns:    "Envelope success.data with ids/count",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
		},
		{
			Name:    "a2ui_get_surface",
			Summary: "Get full surface record",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with full surface",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "SURFACE_NOT_FOUND"},
		},
		{
			Name:    "a2ui_get_surface_model",
			Summary: "Get one surface model",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with dataModel",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "SURFACE_NOT_FOUND"},
		},
		{
			Name:    "a2ui_get_surface_components",
			Summary: "Get one surface component metadata",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with rootId/componentIds",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "SURFACE_NOT_FOUND"},
		},
		{
			Name:    "a2ui_render_surface",
			Summary: "Render one existing surface",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with html",
			Mutable:    false,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "RENDER_SURFACE_FAILED"},
		},
		{
			Name:    "a2ui_create_surface",
			Summary: "Create empty surface",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with surfaceId",
			Mutable:    true,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "CREATE_SURFACE_FAILED"},
		},
		{
			Name:    "a2ui_delete_surface",
			Summary: "Delete one surface",
			Arguments: []toolArgSpec{
				{Name: "surface_id", Type: "string", Required: true, Description: "Surface id"},
			},
			Returns:    "Envelope success.data with deleted=true",
			Mutable:    true,
			Idempotent: true,
			Category:   "surfaces",
			ErrorCodes: []string{"INVALID_ARGUMENT", "SURFACE_NOT_FOUND"},
		},
		{
			Name:       "a2ui_reset_runtime",
			Summary:    "Delete all in-memory surfaces",
			Returns:    "Envelope success.data with removed",
			Mutable:    true,
			Idempotent: true,
			Category:   "runtime",
		},
		{
			Name:    "a2ui_inspect_table_row",
			Summary: "Render inspector HTML for one table row",
			Arguments: []toolArgSpec{
				{Name: "page_id", Type: "string", Required: true, Description: "Page id containing the table"},
				{Name: "table_id", Type: "string", Required: true, Description: "Table component id"},
				{Name: "row", Type: "string", Required: true, Description: "Zero-based row index as integer string"},
				{Name: "target_id", Type: "string", Required: false, Description: "Optional root id for generated inspector"},
			},
			Returns:    "Envelope success.data with html",
			Mutable:    false,
			Idempotent: true,
			Category:   "diagnostics",
			ErrorCodes: []string{"INVALID_ARGUMENT", "INSPECT_FAILED"},
		},
		{
			Name:       "a2ui_list_composites",
			Summary:    "List built-in composite reference names",
			Returns:    "Envelope success.data array of names",
			Mutable:    false,
			Idempotent: true,
			Category:   "examples",
			ErrorCodes: []string{"LIST_COMPOSITES_FAILED"},
		},
		{
			Name:    "a2ui_fetch_composite",
			Summary: "Fetch one composite JSONL by name",
			Arguments: []toolArgSpec{
				{Name: "name", Type: "string", Required: true, Description: "Composite name"},
			},
			Returns:    "Envelope success.data with name/route/jsonl",
			Mutable:    false,
			Idempotent: true,
			Category:   "examples",
			ErrorCodes: []string{"INVALID_ARGUMENT", "COMPOSITE_NOT_FOUND"},
		},
		{
			Name:       "a2ui_examples_list",
			Summary:    "List example payload names",
			Returns:    "Envelope success.data with examples/count",
			Mutable:    false,
			Idempotent: true,
			Category:   "examples",
			ErrorCodes: []string{"LIST_EXAMPLES_FAILED"},
		},
		{
			Name:    "a2ui_example_get",
			Summary: "Fetch one example JSONL payload by name",
			Arguments: []toolArgSpec{
				{Name: "name", Type: "string", Required: true, Description: "Example name without .jsonl"},
			},
			Returns:    "Envelope success.data with name/jsonl",
			Mutable:    false,
			Idempotent: true,
			Category:   "examples",
			ErrorCodes: []string{"INVALID_ARGUMENT", "INVALID_EXAMPLE_NAME", "EXAMPLE_NOT_FOUND", "READ_EXAMPLE_FAILED"},
		},
	}
}

func discoveryDoc() map[string]any {
	tools := discoveryToolSpecs()
	workflow := []string{
		"a2ui_discovery",
		"a2ui_validate_jsonl",
		"a2ui_apply_jsonl",
		"a2ui_get_surface_model",
		"a2ui_get_surface_components",
		"a2ui_render_surface",
	}
	return map[string]any{
		"schemaVersion":   "2026-04-20",
		"profile":         "a2ui-mcp-v2",
		"protocolName":    "A2UI",
		"protocolVersion": a2ui.VersionV08,
		"server":          "github.com/dmundt/a2ui-go",
		"communication": map[string]any{
			"format":   "jsonl",
			"spec":     "A2UI",
			"version":  a2ui.VersionV08,
			"required": true,
		},
		"uiSpecification": map[string]any{
			"spec":               "A2UI",
			"version":            a2ui.VersionV08,
			"required":           true,
			"incompatiblePolicy": "reject",
		},
		"envelope": map[string]any{
			"success": "boolean",
			"data":    "object|array|string|null",
			"warnings": []string{
				"optional non-fatal warnings",
			},
			"error": map[string]any{
				"code":    "string",
				"message": "string",
				"details": "object|null",
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
			"self-discovery",
		},
		"components":     a2ui.CatalogComponents(),
		"bootstrapOrder": workflow,
		"tools":          tools,
	}
}

func describeToolByName(name string) (toolSpec, bool) {
	for _, spec := range discoveryToolSpecs() {
		if spec.Name == name {
			return spec, true
		}
	}
	return toolSpec{}, false
}
