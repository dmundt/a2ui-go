package renderer

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/dmundt/a2ui-go/a2ui"
)

// Registry stores deterministic template mappings.
type Registry struct {
	templateDir       string
	tmpl              *template.Template
	componentTemplate map[a2ui.ComponentType]string
	templateNames     []string
}

// NewRegistry builds the renderer template registry in fixed order.
func NewRegistry(templateDir string) (*Registry, error) {
	files := []string{
		"page.html",
		"row.html",
		"column.html",
		"image.html",
		"text.html",
		"button.html",
		"textfield.html",
		"checkbox.html",
		"slider.html",
		"datetimeinput.html",
		"multiplechoice.html",
		"icon.html",
		"divider.html",
		"list.html",
		"card.html",
		"modal.html",
		"tabs.html",
		"input.html",
		"select.html",
		"table.html",
		"form.html",
	}

	t := template.New("a2ui")
	for _, name := range files {
		p := filepath.Join(templateDir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if _, err := t.Parse(string(b)); err != nil {
			return nil, err
		}
	}

	return &Registry{
		templateDir: templateDir,
		tmpl:        t,
		componentTemplate: map[a2ui.ComponentType]string{
			a2ui.ComponentRow:            "component.row",
			a2ui.ComponentColumn:         "component.column",
			a2ui.ComponentImage:          "component.image",
			a2ui.ComponentText:           "component.text",
			a2ui.ComponentButton:         "component.button",
			a2ui.ComponentTextField:      "component.textfield",
			a2ui.ComponentCheckBox:       "component.checkbox",
			a2ui.ComponentSlider:         "component.slider",
			a2ui.ComponentDateTimeInput:  "component.datetimeinput",
			a2ui.ComponentMultipleChoice: "component.multiplechoice",
			a2ui.ComponentIcon:           "component.icon",
			a2ui.ComponentDivider:        "component.divider",
			a2ui.ComponentList:           "component.list",
			a2ui.ComponentCard:           "component.card",
			a2ui.ComponentModal:          "component.modal",
			a2ui.ComponentTabs:           "component.tabs",
			a2ui.ComponentInput:          "component.input",
			a2ui.ComponentSelect:         "component.select",
			a2ui.ComponentTable:          "component.table",
			a2ui.ComponentForm:           "component.form",
		},
		templateNames: files,
	}, nil
}

// TemplateNameFor returns the exact template name mapped to a component type.
func (r *Registry) TemplateNameFor(t a2ui.ComponentType) (string, bool) {
	name, ok := r.componentTemplate[t]
	return name, ok
}

// Templates returns the parsed templates.
func (r *Registry) Templates() *template.Template {
	return r.tmpl
}

// TemplateNames returns known template files in deterministic order.
func (r *Registry) TemplateNames() []string {
	out := make([]string, len(r.templateNames))
	copy(out, r.templateNames)
	return out
}
