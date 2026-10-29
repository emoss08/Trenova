package tooldispatch

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
)

// ScriptSettlement is how a script that may write ended, and what its whole
// run read: every intent is decided again with that, wherever in the script
// it was prepared.
type ScriptSettlement struct {
	Completed     bool
	Taint         *agent.RunTaint
	AfterExternal bool
	Beyond        agent.AutonomyTier
	MoneySpent    agenttooltrust.MoneyUse
}
