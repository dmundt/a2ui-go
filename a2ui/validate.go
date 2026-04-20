package a2ui

import (
	"fmt"
	"strings"
)

// ValidateMessage validates one A2UI v0.8 message.
// Exactly one of the payload fields must be set.
func ValidateMessage(m Message) error {
	count := 0
	if m.SurfaceUpdate != nil {
		count++
	}
	if m.DataModelUpdate != nil {
		count++
	}
	if m.BeginRendering != nil {
		count++
	}
	if m.DeleteSurface != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("message must have exactly one type key, got %d", count)
	}

	switch {
	case m.SurfaceUpdate != nil:
		return validateSurfaceUpdate(m.SurfaceUpdate)
	case m.DataModelUpdate != nil:
		return validateDataModelUpdate(m.DataModelUpdate)
	case m.BeginRendering != nil:
		return validateBeginRendering(m.BeginRendering)
	case m.DeleteSurface != nil:
		return validateDeleteSurface(m.DeleteSurface)
	}
	return nil
}

func validateSurfaceUpdate(p *SurfaceUpdatePayload) error {
	if strings.TrimSpace(p.SurfaceID) == "" {
		return fmt.Errorf("surfaceUpdate.surfaceId is required")
	}
	if len(p.Components) == 0 {
		return fmt.Errorf("surfaceUpdate.components must not be empty")
	}
	for i, raw := range p.Components {
		if strings.TrimSpace(raw.ID) == "" {
			return fmt.Errorf("surfaceUpdate.components[%d].id is required", i)
		}
		if len(raw.Component) == 0 {
			return fmt.Errorf("surfaceUpdate.components[%d].component is required", i)
		}
	}
	return nil
}

func validateDataModelUpdate(p *DataModelUpdatePayload) error {
	if strings.TrimSpace(p.SurfaceID) == "" {
		return fmt.Errorf("dataModelUpdate.surfaceId is required")
	}
	return nil
}

func validateBeginRendering(p *BeginRenderingPayload) error {
	if strings.TrimSpace(p.SurfaceID) == "" {
		return fmt.Errorf("beginRendering.surfaceId is required")
	}
	if strings.TrimSpace(p.Root) == "" {
		return fmt.Errorf("beginRendering.root is required")
	}
	return nil
}

func validateDeleteSurface(p *DeleteSurfacePayload) error {
	if strings.TrimSpace(p.SurfaceID) == "" {
		return fmt.Errorf("deleteSurface.surfaceId is required")
	}
	return nil
}
