package mcp

import "testing"

func TestToolNamesContainsRequiredContracts(t *testing.T) {
	names := toolNames()
	required := []string{
		"a2ui_discovery",
		"a2ui_describe_tool",
		"a2ui_capabilities",
		"a2ui_validate_jsonl",
		"a2ui_apply_jsonl",
		"a2ui_get_surface",
		"a2ui_get_surface_model",
		"a2ui_get_surface_components",
		"a2ui_render_surface",
		"a2ui_create_surface",
		"a2ui_delete_surface",
		"a2ui_reset_runtime",
		"a2ui_inspect_table_row",
		"a2ui_examples_list",
		"a2ui_example_get",
	}

	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; !ok {
			t.Fatalf("missing required tool name: %s", r)
		}
	}
}
