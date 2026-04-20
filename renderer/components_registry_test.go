package renderer

import (
	"strings"
	"testing"

	"github.com/dmundt/a2ui-go/a2ui"
)

func literalBV(v string) *a2ui.BoundValue {
	return &a2ui.BoundValue{LiteralString: &v}
}

func TestRendererCoversComponentBranches(t *testing.T) {
	reg, err := NewRegistry("templates")
	if err != nil {
		reg, err = NewRegistry("../renderer/templates")
		if err != nil {
			t.Fatalf("registry: %v", err)
		}
	}
	r := New(reg)

	dm := a2ui.DataModel{
		"items": []interface{}{
			map[string]interface{}{"name": "first"},
			map[string]interface{}{"name": "second"},
		},
	}

	components := map[string]*a2ui.Component{
		"root":         {ID: "root", Type: a2ui.ComponentColumn, Column: &a2ui.ColumnProps{Children: a2ui.Children{ExplicitList: []string{"text", "image", "button", "textfield", "checkbox", "slider", "datetime", "multi", "icon", "divider", "list", "card", "modal", "tabs", "input", "select", "table", "form", "templated"}}}},
		"text":         {ID: "text", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("hello")}},
		"image":        {ID: "image", Type: a2ui.ComponentImage, Image: &a2ui.ImageProps{URL: literalBV("/img.png")}},
		"buttonLabel":  {ID: "buttonLabel", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("Go")}},
		"button":       {ID: "button", Type: a2ui.ComponentButton, Button: &a2ui.ButtonProps{Child: "buttonLabel", Action: &a2ui.ActionDef{Name: "link:/catalog"}}},
		"textfield":    {ID: "textfield", Type: a2ui.ComponentTextField, TextField: &a2ui.TextFieldProps{Label: literalBV("Name"), Text: literalBV("Dan")}},
		"checkbox":     {ID: "checkbox", Type: a2ui.ComponentCheckBox, CheckBox: &a2ui.CheckBoxProps{Label: literalBV("Enabled"), Value: &a2ui.BoundValue{LiteralBoolean: boolPtr(true)}}},
		"slider":       {ID: "slider", Type: a2ui.ComponentSlider, Slider: &a2ui.SliderProps{Value: literalBV("5"), MinValue: 0, MaxValue: 10}},
		"datetime":     {ID: "datetime", Type: a2ui.ComponentDateTimeInput, DateTimeInput: &a2ui.DateTimeInputProps{Value: literalBV("2026-01-01"), EnableDate: true}},
		"multi":        {ID: "multi", Type: a2ui.ComponentMultipleChoice, MultipleChoice: &a2ui.MultipleChoiceProps{Options: []a2ui.ChoiceOption{{Label: literalBV("A"), Value: "a"}}}},
		"icon":         {ID: "icon", Type: a2ui.ComponentIcon, Icon: &a2ui.IconProps{Name: literalBV("home")}},
		"divider":      {ID: "divider", Type: a2ui.ComponentDivider, Divider: &a2ui.DividerProps{Axis: "horizontal"}},
		"list":         {ID: "list", Type: a2ui.ComponentList, List: &a2ui.ListProps{Children: a2ui.Children{ExplicitList: []string{"text"}}}},
		"cardChild":    {ID: "cardChild", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("inside card")}},
		"card":         {ID: "card", Type: a2ui.ComponentCard, Card: &a2ui.CardProps{Child: "cardChild"}},
		"modalEntry":   {ID: "modalEntry", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("open")}},
		"modalContent": {ID: "modalContent", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("content")}},
		"modal":        {ID: "modal", Type: a2ui.ComponentModal, Modal: &a2ui.ModalProps{EntryPointChild: "modalEntry", ContentChild: "modalContent"}},
		"tabContent":   {ID: "tabContent", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: literalBV("tab body")}},
		"tabs":         {ID: "tabs", Type: a2ui.ComponentTabs, Tabs: &a2ui.TabsProps{TabItems: []a2ui.TabItem{{Title: literalBV("Tab 1"), Child: "tabContent"}}}},
		"input":        {ID: "input", Type: a2ui.ComponentInput, Input: &a2ui.InputProps{Name: "n", Label: "N"}},
		"select":       {ID: "select", Type: a2ui.ComponentSelect, Select: &a2ui.SelectProps{Name: "s", Options: []a2ui.SelectOption{{Value: "v", Label: "V"}}}},
		"table":        {ID: "table", Type: a2ui.ComponentTable, Table: &a2ui.TableProps{Headers: []string{"H"}, Rows: [][]string{{"R"}}}},
		"form":         {ID: "form", Type: a2ui.ComponentForm, Form: &a2ui.FormProps{Action: "/submit", Method: "post", SubmitLabel: "Save"}},
		"itemText":     {ID: "itemText", Type: a2ui.ComponentText, Text: &a2ui.TextProps{Text: &a2ui.BoundValue{Path: "/name"}}},
		"templated":    {ID: "templated", Type: a2ui.ComponentList, List: &a2ui.ListProps{Children: a2ui.Children{Template: &a2ui.ChildTemplate{DataBinding: "/items", ComponentID: "itemText"}}}},
	}

	html, err := r.RenderSurface(components, dm, "root")
	if err != nil {
		t.Fatalf("RenderSurface error: %v", err)
	}
	out := string(html)
	for _, want := range []string{"hello", "inside card", "tab body", "first", "second"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered html missing %q", want)
		}
	}
}

func TestRegistryAndRendererErrorPaths(t *testing.T) {
	if _, err := NewRegistry("does-not-exist"); err == nil {
		t.Fatalf("expected NewRegistry to fail for missing template dir")
	}

	reg, err := NewRegistry("templates")
	if err != nil {
		reg, err = NewRegistry("../renderer/templates")
		if err != nil {
			t.Fatalf("registry: %v", err)
		}
	}

	names := reg.TemplateNames()
	if len(names) == 0 {
		t.Fatalf("expected template names")
	}
	names[0] = "mutated"
	names2 := reg.TemplateNames()
	if names2[0] == "mutated" {
		t.Fatalf("TemplateNames should return a copy")
	}

	r := New(reg)
	if _, err := r.RenderComponent(map[string]*a2ui.Component{}, a2ui.DataModel{}, &a2ui.Component{ID: "x", Type: a2ui.ComponentType("Unknown")}); err == nil {
		t.Fatalf("expected error for unknown template mapping")
	}

	html, err := r.RenderSurface(map[string]*a2ui.Component{}, a2ui.DataModel{}, "missing")
	if err != nil {
		t.Fatalf("RenderSurface with missing root should not error: %v", err)
	}
	if !strings.Contains(string(html), "<!doctype html>") {
		t.Fatalf("expected page shell even with missing root")
	}
}

func boolPtr(v bool) *bool { return &v }
