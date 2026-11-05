package managedusage

import (
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

const CurrentStateID = "current"

var ErrNoState = errors.New("the control plane has not set a usage state yet")

type Unit string

const (
	UnitCount = Unit("count")
	UnitCents = Unit("cents")
	UnitBytes = Unit("bytes")
)

func (u Unit) IsValid() bool {
	switch u {
	case UnitCount, UnitCents, UnitBytes:
		return true
	default:
		return false
	}
}

type Meter struct {
	Meter     platformcatalog.MeterKey `json:"meter"`
	Unit      Unit                     `json:"unit"`
	Quantity  int64                    `json:"quantity"`
	Allowance int64                    `json:"allowance"`
	HardCap   bool                     `json:"hardCap"`
	Stopped   bool                     `json:"stopped"`
}

type State struct {
	keel.Table `keel:"managed_usage_states"`

	ID          string  `json:"id"          keel:"id,pk"`
	WindowStart int64   `json:"windowStart" keel:"window_start,nullzero"`
	WindowEnd   int64   `json:"windowEnd"   keel:"window_end,nullzero"`
	Overage     bool    `json:"overage"     keel:"overage"`
	Meters      []Meter `json:"meters"      keel:"meters,jsonb"`
	ReceivedAt  int64   `json:"receivedAt"  keel:"received_at"`
}

func (s *State) HasWindow() bool {
	return s.WindowEnd > 0
}

func (s *State) Meter(key platformcatalog.MeterKey) (Meter, bool) {
	for _, m := range s.Meters {
		if m.Meter == key {
			return m, true
		}
	}
	return Meter{}, false
}
