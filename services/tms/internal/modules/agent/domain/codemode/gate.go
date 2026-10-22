package codemode

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

type Override string

const (
	OverrideDefault = Override("")
	OverrideOn      = Override("on")
	OverrideOff     = Override("off")
	// OverrideWrites forces code mode on with writes, for the bench's and the
	// red team's batch-write cases; OverrideOn forces it on to read only.
	OverrideWrites = Override("writes")
)

func (o Override) IsValid() bool {
	switch o {
	case OverrideDefault, OverrideOn, OverrideOff, OverrideWrites:
		return true
	default:
		return false
	}
}

// Gate decides whether a turn is offered the script tool and whether a call
// may run one, from the settings of the chat provider that answered.
// Forced lets the bench's override open it, in a development or test
// deployment only.
type Gate struct {
	Forced bool
}

type Decision struct {
	Allowed bool
	// Writes says the script may prepare changes as well as read.
	Writes bool
	Reason string
}

// Offered says whether a turn opening now is shown the script tool. A turn's
// tools are fixed when it opens, before any model has answered, so it is
// offered while any provider that may answer the turn has scripts on
// (offered); turning scripts on mid-turn adds nothing to a turn already open.
func (g Gate) Offered(override Override, offered bool) bool {
	switch override {
	case OverrideOff:
		return false
	case OverrideOn, OverrideWrites:
		return g.Forced
	case OverrideDefault:
		return offered
	default:
		return false
	}
}

// Decide is the gate for one call, read at dispatch from the settings the
// answering provider holds now, so turning scripts or their writes off
// applies at once, mid-turn included.
func (g Gate) Decide(override Override, model string, settings aiprovider.ScriptSettings) Decision {
	switch {
	case override == OverrideOff:
		return Decision{Reason: "code mode is turned off for this turn"}
	case (override == OverrideOn || override == OverrideWrites) && g.Forced:
		return Decision{Allowed: true, Writes: override == OverrideWrites}
	case override == OverrideOn || override == OverrideWrites:
		return Decision{Reason: "code mode can be forced on only in development and test"}
	case settings.Scripts:
		return Decision{Allowed: true, Writes: settings.Writes}
	default:
		name := strings.TrimSpace(model)
		if name == "" {
			name = "this model"
		}
		return Decision{
			Reason: "scripts are not turned on for the provider that answered with " + name,
		}
	}
}
