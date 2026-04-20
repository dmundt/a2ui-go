package a2ui

import "sort"

// VersionV08 is the only supported A2UI protocol version.
const VersionV08 = "0.8"

// StandardCatalogID is the identifier for the v0.8 standard component catalog.
const StandardCatalogID = "https://a2ui.org/specification/v0_8/standard_catalog_definition.json"

// ComponentType identifies a catalog component.
type ComponentType string

const (
	// Official reference components.
	ComponentButton         ComponentType = "Button"
	ComponentCard           ComponentType = "Card"
	ComponentCheckBox       ComponentType = "CheckBox"
	ComponentColumn         ComponentType = "Column"
	ComponentDateTimeInput  ComponentType = "DateTimeInput"
	ComponentDivider        ComponentType = "Divider"
	ComponentIcon           ComponentType = "Icon"
	ComponentImage          ComponentType = "Image"
	ComponentList           ComponentType = "List"
	ComponentModal          ComponentType = "Modal"
	ComponentMultipleChoice ComponentType = "MultipleChoice"
	ComponentRow            ComponentType = "Row"
	ComponentSlider         ComponentType = "Slider"
	ComponentTabs           ComponentType = "Tabs"
	ComponentText           ComponentType = "Text"
	ComponentTextField      ComponentType = "TextField"

	// Legacy renderer-specific components kept for backward compatibility.
	ComponentInput  ComponentType = "Input"
	ComponentSelect ComponentType = "Select"
	ComponentTable  ComponentType = "Table"
	ComponentForm   ComponentType = "Form"
)

var officialCatalogTypes = map[ComponentType]struct{}{
	ComponentButton:         {},
	ComponentCard:           {},
	ComponentCheckBox:       {},
	ComponentColumn:         {},
	ComponentDateTimeInput:  {},
	ComponentDivider:        {},
	ComponentIcon:           {},
	ComponentImage:          {},
	ComponentList:           {},
	ComponentModal:          {},
	ComponentMultipleChoice: {},
	ComponentRow:            {},
	ComponentSlider:         {},
	ComponentTabs:           {},
	ComponentText:           {},
	ComponentTextField:      {},
}

var validComponentTypes = map[ComponentType]struct{}{
	ComponentButton:         {},
	ComponentCard:           {},
	ComponentCheckBox:       {},
	ComponentColumn:         {},
	ComponentDateTimeInput:  {},
	ComponentDivider:        {},
	ComponentIcon:           {},
	ComponentImage:          {},
	ComponentList:           {},
	ComponentModal:          {},
	ComponentMultipleChoice: {},
	ComponentRow:            {},
	ComponentSlider:         {},
	ComponentTabs:           {},
	ComponentText:           {},
	ComponentTextField:      {},
	ComponentInput:          {},
	ComponentSelect:         {},
	ComponentTable:          {},
	ComponentForm:           {},
}

// CatalogComponents returns the official base component catalog in deterministic order.
func CatalogComponents() []ComponentType {
	out := make([]ComponentType, 0, len(officialCatalogTypes))
	for t := range officialCatalogTypes {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
