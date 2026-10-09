package agent

import "github.com/emoss08/trenova/pkg/errortypes"

type Surface string

const (
	SurfaceDesk      = Surface("Desk")
	SurfaceAssistant = Surface("Assistant")
)

func (s Surface) IsValid() bool {
	switch s {
	case "", SurfaceDesk, SurfaceAssistant:
		return true
	default:
		return false
	}
}

func (s Surface) Validate(field string, multiErr *errortypes.MultiError) {
	if !s.IsValid() {
		multiErr.Add(field, errortypes.ErrInvalid, "Surface must be Desk or Assistant")
	}
}
