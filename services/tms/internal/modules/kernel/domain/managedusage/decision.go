package managedusage

import "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

type Reason string

const (
	ReasonNotManaged = Reason("not_managed")
	ReasonWithin     = Reason("within_allowance")
	ReasonNoWindow   = Reason("no_usage_window")
	ReasonStopped    = Reason("stopped")
	ReasonAllowance  = Reason("allowance_reached")
)

type Decision struct {
	Meter       platformcatalog.MeterKey
	Managed     bool
	Allowed     bool
	Reason      Reason
	Limit       int64
	Used        int64
	Requested   int64
	WindowStart int64
	WindowEnd   int64
}

func (d *Decision) Remaining() int64 {
	if !d.Managed || d.Used >= d.Limit {
		return 0
	}
	return d.Limit - d.Used
}

func Decide(s *State, meter platformcatalog.MeterKey, requested int64) Decision {
	d := Decision{Meter: meter, Requested: requested, Allowed: true, Reason: ReasonNotManaged}
	if s == nil {
		return d
	}
	m, ok := s.Meter(meter)
	if !ok {
		return d
	}
	d.Managed = true
	d.Limit = m.Allowance
	d.Used = m.Quantity
	d.WindowStart = s.WindowStart
	d.WindowEnd = s.WindowEnd
	switch {
	case !s.HasWindow():
		d.Allowed, d.Reason = false, ReasonNoWindow
	case m.Stopped:
		d.Allowed, d.Reason = false, ReasonStopped
	case (m.HardCap || !s.Overage) && requested > 0 && m.Quantity+requested > m.Allowance:
		d.Allowed, d.Reason = false, ReasonAllowance
	default:
		d.Reason = ReasonWithin
	}
	return d
}
