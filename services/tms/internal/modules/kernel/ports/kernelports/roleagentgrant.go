package kernelports

import (
	"github.com/emoss08/trenova/shared/pulid"
)

// AgentAudience narrows a read of agents, or of what they raised, to the
// agents a person may use: every agent open to everyone, and the restricted
// ones their roles grant.
type AgentAudience struct {
	GrantedAgentIDs []pulid.ID
}
