package a2ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------- Wire-level message types (v0.8) ----------

// Message is one JSONL line in A2UI v0.8.
// Exactly one field must be non-nil.
type Message struct {
	SurfaceUpdate   *SurfaceUpdatePayload   `json:"surfaceUpdate,omitempty"`
	DataModelUpdate *DataModelUpdatePayload `json:"dataModelUpdate,omitempty"`
	BeginRendering  *BeginRenderingPayload  `json:"beginRendering,omitempty"`
	DeleteSurface   *DeleteSurfacePayload   `json:"deleteSurface,omitempty"`
}

// SurfaceUpdatePayload carries component definitions for a surface.
type SurfaceUpdatePayload struct {
	SurfaceID  string            `json:"surfaceId"`
	Components []RawComponentDef `json:"components"`
}

// RawComponentDef is the wire representation before type-dispatch.
type RawComponentDef struct {
	ID        string                     `json:"id"`
	Component map[string]json.RawMessage `json:"component"`
}

// DataModelUpdatePayload updates the data model for a surface.
type DataModelUpdatePayload struct {
	SurfaceID string      `json:"surfaceId"`
	Path      string      `json:"path,omitempty"`
	Contents  []DataEntry `json:"contents"`
}

// DataEntry is one keyed entry in a dataModelUpdate contents array.
type DataEntry struct {
	Key          string      `json:"key"`
	ValueString  *string     `json:"valueString,omitempty"`
	ValueNumber  *float64    `json:"valueNumber,omitempty"`
	ValueBoolean *bool       `json:"valueBoolean,omitempty"`
	ValueMap     []DataEntry `json:"valueMap,omitempty"`
}

// BeginRenderingPayload signals the client to render from the given root.
type BeginRenderingPayload struct {
	SurfaceID string `json:"surfaceId"`
	Root      string `json:"root"`
	CatalogID string `json:"catalogId,omitempty"`
}

// DeleteSurfacePayload removes a surface.
type DeleteSurfacePayload struct {
	SurfaceID string `json:"surfaceId"`
}

// ---------- BoundValue ----------

// BoundValue holds a literal value or a reference to a path in the data model.
type BoundValue struct {
	LiteralString  *string  `json:"literalString,omitempty"`
	LiteralNumber  *float64 `json:"literalNumber,omitempty"`
	LiteralBoolean *bool    `json:"literalBoolean,omitempty"`
	Path           string   `json:"path,omitempty"`
}

// Str resolves the BoundValue to a string using dm for path lookups.
func (bv *BoundValue) Str(dm DataModel) string {
	if bv == nil {
		return ""
	}
	if bv.LiteralString != nil {
		return *bv.LiteralString
	}
	if bv.LiteralNumber != nil {
		return fmt.Sprintf("%g", *bv.LiteralNumber)
	}
	if bv.LiteralBoolean != nil {
		if *bv.LiteralBoolean {
			return "true"
		}
		return "false"
	}
	if bv.Path != "" {
		if v := dm.Get(bv.Path); v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// Bool resolves the BoundValue to a boolean using dm for path lookups.
func (bv *BoundValue) Bool(dm DataModel) bool {
	if bv == nil {
		return false
	}
	if bv.LiteralBoolean != nil {
		return *bv.LiteralBoolean
	}
	if bv.Path != "" {
		if v := dm.Get(bv.Path); v != nil {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return false
}

// ---------- DataModel ----------

// DataModel is the per-surface application state store.
type DataModel map[string]interface{}

// Get resolves a JSON-pointer-style path (e.g. "/user/name") against the model.
func (dm DataModel) Get(path string) interface{} {
	if dm == nil || path == "" || path == "/" {
		return nil
	}
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	var cur interface{} = map[string]interface{}(dm)
	for _, p := range parts {
		if p == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

// GetList resolves a path and returns a slice of map[string]interface{} for template iteration.
func (dm DataModel) GetList(path string) []map[string]interface{} {
	v := dm.Get(path)
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	case []map[string]interface{}:
		return t
	}
	return nil
}

// ApplyUpdate applies a DataModelUpdatePayload to this model.
func (dm DataModel) ApplyUpdate(path string, contents []DataEntry) {
	target := map[string]interface{}(dm)
	if path != "" && path != "/" {
		path = strings.TrimPrefix(path, "/")
		parts := strings.Split(path, "/")
		cur := map[string]interface{}(dm)
		for _, part := range parts[:len(parts)-1] {
			if next, ok := cur[part]; ok {
				if m, ok2 := next.(map[string]interface{}); ok2 {
					cur = m
				} else {
					m2 := make(map[string]interface{})
					cur[part] = m2
					cur = m2
				}
			} else {
				m2 := make(map[string]interface{})
				cur[part] = m2
				cur = m2
			}
		}
		last := parts[len(parts)-1]
		if m, ok := cur[last].(map[string]interface{}); ok {
			target = m
		} else {
			m2 := make(map[string]interface{})
			cur[last] = m2
			target = m2
		}
	}
	applyDataEntries(target, contents)
}

func applyDataEntries(target map[string]interface{}, entries []DataEntry) {
	for _, e := range entries {
		switch {
		case e.ValueString != nil:
			target[e.Key] = *e.ValueString
		case e.ValueNumber != nil:
			target[e.Key] = *e.ValueNumber
		case e.ValueBoolean != nil:
			target[e.Key] = *e.ValueBoolean
		case e.ValueMap != nil:
			sub, ok := target[e.Key].(map[string]interface{})
			if !ok {
				sub = make(map[string]interface{})
			}
			applyDataEntries(sub, e.ValueMap)
			target[e.Key] = sub
		}
	}
}

// ---------- Children ----------

// Children defines list/row/column layout children.
type Children struct {
	ExplicitList []string       `json:"explicitList,omitempty"`
	Template     *ChildTemplate `json:"template,omitempty"`
}

// ChildTemplate defines dynamic children from a data-bound list.
type ChildTemplate struct {
	DataBinding string `json:"dataBinding"`
	ComponentID string `json:"componentId"`
}

// ---------- Action ----------

// ActionDef defines the action fired when a user interacts with a component.
type ActionDef struct {
	Name    string         `json:"name"`
	Context []ContextEntry `json:"context,omitempty"`
}

// ContextEntry is one key-value pair in an action context.
type ContextEntry struct {
	Key   string     `json:"key"`
	Value BoundValue `json:"value"`
}

// ---------- Component prop types (v0.8) ----------

// RowProps configures Row.
type RowProps struct {
	Children     Children `json:"children"`
	Distribution string   `json:"distribution,omitempty"`
	Alignment    string   `json:"alignment,omitempty"`
}

// ColumnProps configures Column.
type ColumnProps struct {
	Children     Children `json:"children"`
	Distribution string   `json:"distribution,omitempty"`
	Alignment    string   `json:"alignment,omitempty"`
}

// ListProps configures List.
type ListProps struct {
	Children  Children `json:"children"`
	Direction string   `json:"direction,omitempty"`
	Alignment string   `json:"alignment,omitempty"`
}

// TextProps configures Text.
type TextProps struct {
	Text      *BoundValue `json:"text"`
	UsageHint string      `json:"usageHint,omitempty"`
}

// ImageProps configures Image.
type ImageProps struct {
	URL       *BoundValue `json:"url"`
	Fit       string      `json:"fit,omitempty"`
	UsageHint string      `json:"usageHint,omitempty"`
}

// ButtonProps configures Button.
type ButtonProps struct {
	Child   string     `json:"child,omitempty"`
	Primary bool       `json:"primary,omitempty"`
	Action  *ActionDef `json:"action,omitempty"`
}

// TextFieldProps configures TextField.
type TextFieldProps struct {
	Label            *BoundValue `json:"label,omitempty"`
	Text             *BoundValue `json:"text,omitempty"`
	TextFieldType    string      `json:"textFieldType,omitempty"`
	ValidationRegexp string      `json:"validationRegexp,omitempty"`
}

// CheckBoxProps configures CheckBox.
type CheckBoxProps struct {
	Label *BoundValue `json:"label,omitempty"`
	Value *BoundValue `json:"value,omitempty"`
}

// SliderProps configures Slider.
type SliderProps struct {
	Value    *BoundValue `json:"value,omitempty"`
	MinValue float64     `json:"minValue,omitempty"`
	MaxValue float64     `json:"maxValue,omitempty"`
}

// DateTimeInputProps configures DateTimeInput.
type DateTimeInputProps struct {
	Value      *BoundValue `json:"value,omitempty"`
	EnableDate bool        `json:"enableDate,omitempty"`
	EnableTime bool        `json:"enableTime,omitempty"`
}

// ChoiceOption is one MultipleChoice option.
type ChoiceOption struct {
	Label *BoundValue `json:"label"`
	Value string      `json:"value"`
}

// MultipleChoiceProps configures MultipleChoice.
type MultipleChoiceProps struct {
	Options              []ChoiceOption `json:"options"`
	Selections           *BoundValue    `json:"selections,omitempty"`
	MaxAllowedSelections int            `json:"maxAllowedSelections,omitempty"`
}

// IconProps configures Icon.
type IconProps struct {
	Name *BoundValue `json:"name"`
}

// DividerProps configures Divider.
type DividerProps struct {
	Axis string `json:"axis,omitempty"`
}

// CardProps configures Card.
type CardProps struct {
	Child string `json:"child,omitempty"`
}

// ModalProps configures Modal.
type ModalProps struct {
	EntryPointChild string `json:"entryPointChild,omitempty"`
	ContentChild    string `json:"contentChild,omitempty"`
}

// TabItem defines one tab entry.
type TabItem struct {
	Title *BoundValue `json:"title"`
	Child string      `json:"child"`
}

// TabsProps configures Tabs.
type TabsProps struct {
	TabItems []TabItem `json:"tabItems"`
}

// InputProps configures legacy Input component.
type InputProps struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	Type        string `json:"type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// SelectProps configures legacy Select component.
type SelectProps struct {
	Name     string         `json:"name"`
	Label    string         `json:"label,omitempty"`
	Value    string         `json:"value,omitempty"`
	Options  []SelectOption `json:"options"`
	Required bool           `json:"required,omitempty"`
}

// SelectOption represents one option in Select.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// TableProps configures legacy Table component.
type TableProps struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// FormProps configures legacy Form component.
type FormProps struct {
	Action      string `json:"action,omitempty"`
	Method      string `json:"method,omitempty"`
	SubmitLabel string `json:"submit_label,omitempty"`
}

// ---------- Typed Component ----------

// Component is a parsed, typed catalog component instance stored in the surface buffer.
type Component struct {
	ID   string
	Type ComponentType

	Row            *RowProps
	Column         *ColumnProps
	List           *ListProps
	Text           *TextProps
	Image          *ImageProps
	Button         *ButtonProps
	TextField      *TextFieldProps
	CheckBox       *CheckBoxProps
	Slider         *SliderProps
	DateTimeInput  *DateTimeInputProps
	MultipleChoice *MultipleChoiceProps
	Icon           *IconProps
	Divider        *DividerProps
	Card           *CardProps
	Modal          *ModalProps
	Tabs           *TabsProps
	Input          *InputProps
	Select         *SelectProps
	Table          *TableProps
	Form           *FormProps
}

// ParseComponentDef parses a raw wire ComponentDef into a typed Component.
func ParseComponentDef(raw RawComponentDef) (Component, error) {
	if len(raw.Component) != 1 {
		return Component{}, fmt.Errorf("component %q: expected exactly one type key, got %d", raw.ID, len(raw.Component))
	}
	var typeName string
	var propsRaw json.RawMessage
	for k, v := range raw.Component {
		typeName = k
		propsRaw = v
	}
	c := Component{ID: raw.ID, Type: ComponentType(typeName)}
	switch c.Type {
	case ComponentRow:
		var p RowProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Row props: %w", err)
		}
		c.Row = &p
	case ComponentColumn:
		var p ColumnProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Column props: %w", err)
		}
		c.Column = &p
	case ComponentList:
		var p ListProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("List props: %w", err)
		}
		c.List = &p
	case ComponentText:
		var p TextProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Text props: %w", err)
		}
		c.Text = &p
	case ComponentImage:
		var p ImageProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Image props: %w", err)
		}
		c.Image = &p
	case ComponentButton:
		var p ButtonProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Button props: %w", err)
		}
		c.Button = &p
	case ComponentTextField:
		var p TextFieldProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("TextField props: %w", err)
		}
		c.TextField = &p
	case ComponentCheckBox:
		var p CheckBoxProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("CheckBox props: %w", err)
		}
		c.CheckBox = &p
	case ComponentSlider:
		var p SliderProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Slider props: %w", err)
		}
		c.Slider = &p
	case ComponentDateTimeInput:
		var p DateTimeInputProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("DateTimeInput props: %w", err)
		}
		c.DateTimeInput = &p
	case ComponentMultipleChoice:
		var p MultipleChoiceProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("MultipleChoice props: %w", err)
		}
		c.MultipleChoice = &p
	case ComponentIcon:
		var p IconProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Icon props: %w", err)
		}
		c.Icon = &p
	case ComponentDivider:
		var p DividerProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Divider props: %w", err)
		}
		c.Divider = &p
	case ComponentCard:
		var p CardProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Card props: %w", err)
		}
		c.Card = &p
	case ComponentModal:
		var p ModalProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Modal props: %w", err)
		}
		c.Modal = &p
	case ComponentTabs:
		var p TabsProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Tabs props: %w", err)
		}
		c.Tabs = &p
	case ComponentInput:
		var p InputProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Input props: %w", err)
		}
		c.Input = &p
	case ComponentSelect:
		var p SelectProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Select props: %w", err)
		}
		c.Select = &p
	case ComponentTable:
		var p TableProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Table props: %w", err)
		}
		c.Table = &p
	case ComponentForm:
		var p FormProps
		if err := json.Unmarshal(propsRaw, &p); err != nil {
			return Component{}, fmt.Errorf("Form props: %w", err)
		}
		c.Form = &p
	default:
		return Component{}, fmt.Errorf("unknown component type %q", typeName)
	}
	return c, nil
}
