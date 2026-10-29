package tooldispatch

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

type ScriptBrokerParams struct {
	Req           *agentruntimeports.RunRequest
	ParentKey     string
	ParentCallID  string
	Taint         *agent.RunTaint
	AfterExternal bool
	// Writes lets the script prepare changes, which the code-mode gate
	// allows only for a model the batch-write bench earned it for.
	Writes bool
	Reach  map[string]agent.AutonomyTier
}
