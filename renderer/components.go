package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/dmundt/au2ui-go/a2ui"
)

// ---------- View types (all values pre-resolved from BoundValues) ----------

type rowView struct {
	Distribution string
	Alignment    string
}

type columnView struct {
	Distribution string
	Alignment    string
}

type listView struct {
	Direction string
	Alignment string
}

type textView struct {
	Value     string
	UsageHint string
}

type imageView struct {
	URL       string
	Fit       string
	UsageHint string
}

type buttonView struct {
	ChildHTML  template.HTML
	Primary    bool
	ActionName string
	ActionJSON string // JSON context object for hx-vals
	LinkURL    string
}

type textFieldView struct {
	Label            string
	Value            string
	TextFieldType    string
	ValidationRegexp string
}

type checkBoxView struct {
	Label string
	Value bool
}

type sliderView struct {
	Value    string
	MinValue float64
	MaxValue float64
}

type dateTimeInputView struct {
	Value      string
	EnableDate bool
	EnableTime bool
}

type choiceOptionView struct {
	Label string
	Value string
}

type multipleChoiceView struct {
	Options              []choiceOptionView
	MaxAllowedSelections int
}

type iconView struct {
	Name string
}

type dividerView struct {
	Axis string
}

type cardView struct {
	ChildHTML template.HTML
}

type modalView struct {
	EntryPointHTML template.HTML
	ContentHTML    template.HTML
}

type tabItemView struct {
	Title   string
	Content template.HTML
}

type tabsView struct {
	Items []tabItemView
}

// componentView is passed to every HTML template.
type componentView struct {
	ID    string
	Class string

	// Pre-rendered children (Row, Column, List)
	Children template.HTML

	// v0.8 component views
	Row            *rowView
	Column         *columnView
	List           *listView
	Text           *textView
	Image          *imageView
	Button         *buttonView
	TextField      *textFieldView
	CheckBox       *checkBoxView
	Slider         *sliderView
	DateTimeInput  *dateTimeInputView
	MultipleChoice *multipleChoiceView
	Icon           *iconView
	Divider        *dividerView
	Card           *cardView
	Modal          *modalView
	Tabs           *tabsView

	// Legacy components (unchanged props)
	Input  *a2ui.InputProps
	Select *a2ui.SelectProps
	Table  *a2ui.TableProps
	Form   *a2ui.FormProps
}

// RenderComponent renders a single typed component with its children resolved from the buffer.
func (r *Renderer) RenderComponent(components map[string]*a2ui.Component, dm a2ui.DataModel, c *a2ui.Component) (template.HTML, error) {
	tmplName, ok := r.registry.TemplateNameFor(c.Type)
	if !ok {
		return "", fmt.Errorf("no template for component type %s", c.Type)
	}

	view, err := r.buildView(components, dm, c)
	if err != nil {
		return "", fmt.Errorf("component %s: %w", c.ID, err)
	}

	var out bytes.Buffer
	if err := r.registry.Templates().ExecuteTemplate(&out, tmplName, view); err != nil {
		return "", fmt.Errorf("template %s: %w", tmplName, err)
	}
	return template.HTML(out.String()), nil
}

// buildView constructs a componentView from a typed Component, resolving all BoundValues and child IDs.
func (r *Renderer) buildView(components map[string]*a2ui.Component, dm a2ui.DataModel, c *a2ui.Component) (componentView, error) {
	view := componentView{ID: c.ID}

	switch c.Type {
	case a2ui.ComponentRow:
		if c.Row != nil {
			ch, err := r.renderChildren(components, dm, c.Row.Children)
			if err != nil {
				return view, err
			}
			view.Children = ch
			view.Row = &rowView{Distribution: c.Row.Distribution, Alignment: c.Row.Alignment}
		}

	case a2ui.ComponentColumn:
		if c.Column != nil {
			ch, err := r.renderChildren(components, dm, c.Column.Children)
			if err != nil {
				return view, err
			}
			view.Children = ch
			view.Column = &columnView{Distribution: c.Column.Distribution, Alignment: c.Column.Alignment}
		}

	case a2ui.ComponentList:
		if c.List != nil {
			ch, err := r.renderChildren(components, dm, c.List.Children)
			if err != nil {
				return view, err
			}
			view.Children = ch
			view.List = &listView{Direction: c.List.Direction, Alignment: c.List.Alignment}
		}

	case a2ui.ComponentText:
		if c.Text != nil {
			view.Text = &textView{Value: c.Text.Text.Str(dm), UsageHint: c.Text.UsageHint}
		}

	case a2ui.ComponentImage:
		if c.Image != nil {
			view.Image = &imageView{URL: c.Image.URL.Str(dm), Fit: c.Image.Fit, UsageHint: c.Image.UsageHint}
		}

	case a2ui.ComponentButton:
		if c.Button != nil {
			var childHTML template.HTML
			if c.Button.Child != "" {
				var err error
				childHTML, err = r.renderByID(components, dm, c.Button.Child)
				if err != nil {
					return view, err
				}
			}
			actionName := ""
			actionJSON := "{}"
			linkURL := ""
			if c.Button.Action != nil {
				actionName = c.Button.Action.Name
				if strings.HasPrefix(actionName, "link:") {
					linkURL = strings.TrimPrefix(actionName, "link:")
					actionName = ""
				}
				ctx := make(map[string]string)
				for _, e := range c.Button.Action.Context {
					ctx[e.Key] = e.Value.Str(dm)
				}
				if b, err := json.Marshal(ctx); err == nil {
					actionJSON = string(b)
				}
			}
			view.Button = &buttonView{
				ChildHTML:  childHTML,
				Primary:    c.Button.Primary,
				ActionName: actionName,
				ActionJSON: actionJSON,
				LinkURL:    linkURL,
			}
		}

	case a2ui.ComponentTextField:
		if c.TextField != nil {
			view.TextField = &textFieldView{
				Label:            c.TextField.Label.Str(dm),
				Value:            c.TextField.Text.Str(dm),
				TextFieldType:    c.TextField.TextFieldType,
				ValidationRegexp: c.TextField.ValidationRegexp,
			}
		}

	case a2ui.ComponentCheckBox:
		if c.CheckBox != nil {
			view.CheckBox = &checkBoxView{
				Label: c.CheckBox.Label.Str(dm),
				Value: c.CheckBox.Value.Bool(dm),
			}
		}

	case a2ui.ComponentSlider:
		if c.Slider != nil {
			view.Slider = &sliderView{
				Value:    c.Slider.Value.Str(dm),
				MinValue: c.Slider.MinValue,
				MaxValue: c.Slider.MaxValue,
			}
		}

	case a2ui.ComponentDateTimeInput:
		if c.DateTimeInput != nil {
			view.DateTimeInput = &dateTimeInputView{
				Value:      c.DateTimeInput.Value.Str(dm),
				EnableDate: c.DateTimeInput.EnableDate,
				EnableTime: c.DateTimeInput.EnableTime,
			}
		}

	case a2ui.ComponentMultipleChoice:
		if c.MultipleChoice != nil {
			opts := make([]choiceOptionView, len(c.MultipleChoice.Options))
			for i, o := range c.MultipleChoice.Options {
				opts[i] = choiceOptionView{Label: o.Label.Str(dm), Value: o.Value}
			}
			view.MultipleChoice = &multipleChoiceView{
				Options:              opts,
				MaxAllowedSelections: c.MultipleChoice.MaxAllowedSelections,
			}
		}

	case a2ui.ComponentIcon:
		if c.Icon != nil {
			view.Icon = &iconView{Name: c.Icon.Name.Str(dm)}
		}

	case a2ui.ComponentDivider:
		if c.Divider != nil {
			view.Divider = &dividerView{Axis: c.Divider.Axis}
		}

	case a2ui.ComponentCard:
		if c.Card != nil {
			childHTML, err := r.renderByID(components, dm, c.Card.Child)
			if err != nil {
				return view, err
			}
			view.Card = &cardView{ChildHTML: childHTML}
		}

	case a2ui.ComponentModal:
		if c.Modal != nil {
			epHTML, err := r.renderByID(components, dm, c.Modal.EntryPointChild)
			if err != nil {
				return view, err
			}
			contentHTML, err := r.renderByID(components, dm, c.Modal.ContentChild)
			if err != nil {
				return view, err
			}
			view.Modal = &modalView{EntryPointHTML: epHTML, ContentHTML: contentHTML}
		}

	case a2ui.ComponentTabs:
		if c.Tabs != nil {
			items := make([]tabItemView, len(c.Tabs.TabItems))
			for i, ti := range c.Tabs.TabItems {
				content, err := r.renderByID(components, dm, ti.Child)
				if err != nil {
					return view, err
				}
				items[i] = tabItemView{Title: ti.Title.Str(dm), Content: content}
			}
			view.Tabs = &tabsView{Items: items}
		}

	// Legacy components - pass props unchanged
	case a2ui.ComponentInput:
		view.Input = c.Input
	case a2ui.ComponentSelect:
		view.Select = c.Select
	case a2ui.ComponentTable:
		view.Table = c.Table
	case a2ui.ComponentForm:
		view.Form = c.Form
	}

	return view, nil
}

// renderChildren pre-renders all children defined by a Children spec.
func (r *Renderer) renderChildren(components map[string]*a2ui.Component, dm a2ui.DataModel, ch a2ui.Children) (template.HTML, error) {
	var buf strings.Builder

	if ch.Template != nil {
		items := dm.GetList(ch.Template.DataBinding)
		tmplComp, ok := components[ch.Template.ComponentID]
		if ok {
			for _, item := range items {
				// Create a sub-DataModel with the item merged at root
				subDM := make(a2ui.DataModel)
				for k, v := range dm {
					subDM[k] = v
				}
				for k, v := range item {
					subDM[k] = v
				}
				h, err := r.RenderComponent(components, subDM, tmplComp)
				if err != nil {
					return "", err
				}
				buf.WriteString(string(h))
			}
		}
		return template.HTML(buf.String()), nil
	}

	for _, id := range ch.ExplicitList {
		h, err := r.renderByID(components, dm, id)
		if err != nil {
			return "", err
		}
		buf.WriteString(string(h))
	}
	return template.HTML(buf.String()), nil
}
