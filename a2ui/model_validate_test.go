package a2ui

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestCatalogComponentsDeterministicAndOfficial(t *testing.T) {
	got := CatalogComponents()
	if len(got) == 0 {
		t.Fatalf("expected non-empty catalog")
	}

	if !sort.SliceIsSorted(got, func(i, j int) bool { return got[i] < got[j] }) {
		t.Fatalf("catalog components are not sorted")
	}

	for _, c := range got {
		if _, ok := officialCatalogTypes[c]; !ok {
			t.Fatalf("unexpected non-official type in catalog: %s", c)
		}
	}
}

func TestBoundValueStrAndBool(t *testing.T) {
	s := "hello"
	n := 42.0
	b := true
	dm := DataModel{"user": map[string]interface{}{"name": "alex"}, "enabled": true}

	cases := []struct {
		name string
		bv   *BoundValue
		str  string
		bl   bool
	}{
		{name: "nil", bv: nil, str: "", bl: false},
		{name: "literal string", bv: &BoundValue{LiteralString: &s}, str: "hello", bl: false},
		{name: "literal number", bv: &BoundValue{LiteralNumber: &n}, str: "42", bl: false},
		{name: "literal bool", bv: &BoundValue{LiteralBoolean: &b}, str: "true", bl: true},
		{name: "path string", bv: &BoundValue{Path: "/user/name"}, str: "alex", bl: false},
		{name: "path bool", bv: &BoundValue{Path: "/enabled"}, str: "true", bl: true},
		{name: "missing path", bv: &BoundValue{Path: "/missing"}, str: "", bl: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.bv.Str(dm); got != tc.str {
				t.Fatalf("Str() got %q want %q", got, tc.str)
			}
			if got := tc.bv.Bool(dm); got != tc.bl {
				t.Fatalf("Bool() got %v want %v", got, tc.bl)
			}
		})
	}
}

func TestDataModelGetAndGetList(t *testing.T) {
	dm := DataModel{
		"user": map[string]interface{}{"name": "alex"},
		"items": []interface{}{
			map[string]interface{}{"name": "one"},
			map[string]interface{}{"name": "two"},
			"ignored",
		},
		"typedItems": []map[string]interface{}{{"name": "typed"}},
	}

	if got := dm.Get("/user/name"); got != "alex" {
		t.Fatalf("Get(/user/name) got %v", got)
	}
	if got := dm.Get("/"); got != nil {
		t.Fatalf("Get(/) should be nil, got %v", got)
	}
	if got := dm.Get("/missing/path"); got != nil {
		t.Fatalf("Get missing path should be nil, got %v", got)
	}

	list := dm.GetList("/items")
	if len(list) != 2 {
		t.Fatalf("GetList(/items) len got %d want 2", len(list))
	}
	if list[0]["name"] != "one" || list[1]["name"] != "two" {
		t.Fatalf("GetList(/items) values unexpected: %#v", list)
	}

	typed := dm.GetList("/typedItems")
	if len(typed) != 1 || typed[0]["name"] != "typed" {
		t.Fatalf("GetList(/typedItems) values unexpected: %#v", typed)
	}
}

func TestDataModelApplyUpdate(t *testing.T) {
	dm := DataModel{}
	s := "value"
	n := 3.5
	b := true

	dm.ApplyUpdate("/", []DataEntry{{Key: "k1", ValueString: &s}, {Key: "k2", ValueNumber: &n}, {Key: "k3", ValueBoolean: &b}})
	if dm["k1"] != "value" || dm["k2"] != 3.5 || dm["k3"] != true {
		t.Fatalf("root apply update failed: %#v", dm)
	}

	dm.ApplyUpdate("/nested/path", []DataEntry{{Key: "inner", ValueString: &s}})
	if got := dm.Get("/nested/path/inner"); got != "value" {
		t.Fatalf("nested update failed, got %v", got)
	}

	dm.ApplyUpdate("/obj", []DataEntry{{
		Key:      "child",
		ValueMap: []DataEntry{{Key: "leaf", ValueString: &s}},
	}})
	if got := dm.Get("/obj/child/leaf"); got != "value" {
		t.Fatalf("valueMap update failed, got %v", got)
	}
}

func TestValidateMessage(t *testing.T) {
	surface := Message{SurfaceUpdate: &SurfaceUpdatePayload{SurfaceID: "s1", Components: []RawComponentDef{{ID: "c1", Component: map[string]json.RawMessage{"Text": json.RawMessage(`{"text":{"literalString":"x"}}`)}}}}}
	data := Message{DataModelUpdate: &DataModelUpdatePayload{SurfaceID: "s1"}}
	begin := Message{BeginRendering: &BeginRenderingPayload{SurfaceID: "s1", Root: "root"}}
	del := Message{DeleteSurface: &DeleteSurfacePayload{SurfaceID: "s1"}}

	for name, msg := range map[string]Message{"surface": surface, "data": data, "begin": begin, "delete": del} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateMessage(msg); err != nil {
				t.Fatalf("ValidateMessage(%s) unexpected error: %v", name, err)
			}
		})
	}

	if err := ValidateMessage(Message{}); err == nil {
		t.Fatalf("expected error for empty message")
	}
	if err := ValidateMessage(Message{BeginRendering: &BeginRenderingPayload{}, DeleteSurface: &DeleteSurfacePayload{SurfaceID: "x"}}); err == nil {
		t.Fatalf("expected error for multiple type keys")
	}
}

func TestParseComponentDef_AllTypes(t *testing.T) {
	raws := []RawComponentDef{
		{ID: "row", Component: mustCompMap(t, "Row", `{"children":{"explicitList":["a"]}}`)},
		{ID: "column", Component: mustCompMap(t, "Column", `{"children":{"explicitList":["a"]}}`)},
		{ID: "list", Component: mustCompMap(t, "List", `{"children":{"explicitList":["a"]}}`)},
		{ID: "text", Component: mustCompMap(t, "Text", `{"text":{"literalString":"x"}}`)},
		{ID: "image", Component: mustCompMap(t, "Image", `{"url":{"literalString":"/img.png"}}`)},
		{ID: "button", Component: mustCompMap(t, "Button", `{"child":"a"}`)},
		{ID: "textfield", Component: mustCompMap(t, "TextField", `{"label":{"literalString":"L"},"text":{"literalString":"T"}}`)},
		{ID: "checkbox", Component: mustCompMap(t, "CheckBox", `{"label":{"literalString":"L"},"value":{"literalBoolean":true}}`)},
		{ID: "slider", Component: mustCompMap(t, "Slider", `{"value":{"literalNumber":5},"minValue":0,"maxValue":10}`)},
		{ID: "datetime", Component: mustCompMap(t, "DateTimeInput", `{"value":{"literalString":"2026-01-01"},"enableDate":true}`)},
		{ID: "multi", Component: mustCompMap(t, "MultipleChoice", `{"options":[{"label":{"literalString":"A"},"value":"a"}]}`)},
		{ID: "icon", Component: mustCompMap(t, "Icon", `{"name":{"literalString":"home"}}`)},
		{ID: "divider", Component: mustCompMap(t, "Divider", `{"axis":"horizontal"}`)},
		{ID: "card", Component: mustCompMap(t, "Card", `{"child":"a"}`)},
		{ID: "modal", Component: mustCompMap(t, "Modal", `{"entryPointChild":"a","contentChild":"b"}`)},
		{ID: "tabs", Component: mustCompMap(t, "Tabs", `{"tabItems":[{"title":{"literalString":"Tab"},"child":"a"}]}`)},
		{ID: "input", Component: mustCompMap(t, "Input", `{"name":"n"}`)},
		{ID: "select", Component: mustCompMap(t, "Select", `{"name":"n","options":[{"value":"v","label":"l"}]}`)},
		{ID: "table", Component: mustCompMap(t, "Table", `{"headers":["h"],"rows":[["r"]]}`)},
		{ID: "form", Component: mustCompMap(t, "Form", `{"action":"/submit"}`)},
	}

	for _, raw := range raws {
		raw := raw
		t.Run(raw.ID, func(t *testing.T) {
			c, err := ParseComponentDef(raw)
			if err != nil {
				t.Fatalf("ParseComponentDef(%s) error: %v", raw.ID, err)
			}
			if c.ID != raw.ID {
				t.Fatalf("component ID mismatch: got %q want %q", c.ID, raw.ID)
			}
			if _, ok := validComponentTypes[c.Type]; !ok {
				t.Fatalf("unexpected parsed type: %s", c.Type)
			}
		})
	}

	if _, err := ParseComponentDef(RawComponentDef{ID: "bad", Component: map[string]json.RawMessage{}}); err == nil {
		t.Fatalf("expected error for empty component map")
	}

	multi := RawComponentDef{ID: "bad2", Component: map[string]json.RawMessage{"Text": json.RawMessage(`{"text":{"literalString":"x"}}`), "Button": json.RawMessage(`{"child":"x"}`)}}
	if _, err := ParseComponentDef(multi); err == nil {
		t.Fatalf("expected error for multiple component keys")
	}

	if _, err := ParseComponentDef(RawComponentDef{ID: "bad3", Component: mustCompMap(t, "Nope", `{}`)}); err == nil {
		t.Fatalf("expected error for unknown component type")
	}

}

func TestValidateMessageErrorCases(t *testing.T) {
	if err := ValidateMessage(Message{SurfaceUpdate: &SurfaceUpdatePayload{}}); err == nil {
		t.Fatalf("expected error for missing surfaceUpdate.surfaceId")
	}
	if err := ValidateMessage(Message{SurfaceUpdate: &SurfaceUpdatePayload{SurfaceID: "s1"}}); err == nil {
		t.Fatalf("expected error for empty surfaceUpdate.components")
	}
	if err := ValidateMessage(Message{SurfaceUpdate: &SurfaceUpdatePayload{SurfaceID: "s1", Components: []RawComponentDef{{ID: "", Component: mustCompMap(t, "Text", `{"text":{"literalString":"x"}}`)}}}}); err == nil {
		t.Fatalf("expected error for empty component ID")
	}
	if err := ValidateMessage(Message{SurfaceUpdate: &SurfaceUpdatePayload{SurfaceID: "s1", Components: []RawComponentDef{{ID: "c1", Component: map[string]json.RawMessage{}}}}}); err == nil {
		t.Fatalf("expected error for empty component payload")
	}
	if err := ValidateMessage(Message{DataModelUpdate: &DataModelUpdatePayload{}}); err == nil {
		t.Fatalf("expected error for missing dataModelUpdate.surfaceId")
	}
	if err := ValidateMessage(Message{BeginRendering: &BeginRenderingPayload{}}); err == nil {
		t.Fatalf("expected error for missing beginRendering fields")
	}
	if err := ValidateMessage(Message{BeginRendering: &BeginRenderingPayload{SurfaceID: "s1"}}); err == nil {
		t.Fatalf("expected error for missing beginRendering.root")
	}
	if err := ValidateMessage(Message{DeleteSurface: &DeleteSurfacePayload{}}); err == nil {
		t.Fatalf("expected error for missing deleteSurface.surfaceId")
	}
}

func TestParseComponentDef_UnmarshalErrors(t *testing.T) {
	bad := []RawComponentDef{
		{ID: "bad-row", Component: mustCompMap(t, "Row", `{"children":1}`)},
		{ID: "bad-text", Component: mustCompMap(t, "Text", `{"text":123}`)},
		{ID: "bad-multi", Component: mustCompMap(t, "MultipleChoice", `{"options":{}}`)},
	}

	for _, raw := range bad {
		if _, err := ParseComponentDef(raw); err == nil {
			t.Fatalf("expected parse error for %s", raw.ID)
		}
	}
}

func mustCompMap(t *testing.T, typeName, props string) map[string]json.RawMessage {
	t.Helper()
	return map[string]json.RawMessage{typeName: json.RawMessage(props)}
}
