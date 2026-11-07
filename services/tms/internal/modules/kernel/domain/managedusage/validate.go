package managedusage

import (
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MaxMeters      = 100
	maxMeterKeyLen = 100
)

func (s *State) Validate() error {
	me := errortypes.NewMultiError()
	switch {
	case s.WindowStart < 0 || s.WindowEnd < 0:
		me.Add("window", errortypes.ErrInvalid, "The window cannot start or end before 1970")
	case (s.WindowStart == 0) != (s.WindowEnd == 0):
		me.Add("window", errortypes.ErrInvalid, "A window needs both a start and an end")
	case s.HasWindow() && s.WindowEnd <= s.WindowStart:
		me.Add("window", errortypes.ErrInvalid, "The window must end after it starts")
	}
	if len(s.Meters) > MaxMeters {
		me.Add("meters", errortypes.ErrInvalid, "At most %d meters", MaxMeters)
	}
	seen := make(map[platformcatalog.MeterKey]struct{}, len(s.Meters))
	for i := range s.Meters {
		m := &s.Meters[i]
		field := me.WithIndex("meters", i)
		if m.Meter == "" || len(m.Meter) > maxMeterKeyLen {
			field.Add("meter", errortypes.ErrRequired, "A meter key of 1 to 100 characters")
		}
		if _, dup := seen[m.Meter]; dup {
			field.Add("meter", errortypes.ErrDuplicate, "The meter is listed twice")
		}
		seen[m.Meter] = struct{}{}
		if !m.Unit.IsValid() {
			field.Add("unit", errortypes.ErrInvalid, "The unit is count, cents or bytes")
		}
		if m.Quantity < 0 {
			field.Add("quantity", errortypes.ErrInvalid, "The quantity cannot be negative")
		}
		if m.Allowance < 0 {
			field.Add("allowance", errortypes.ErrInvalid, "The allowance cannot be negative")
		}
	}
	if me.HasErrors() {
		return me
	}
	return nil
}
