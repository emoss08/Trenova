package benchscenario

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DecisionApprove = DecisionAction("approve")
	DecisionReject  = DecisionAction("reject")
)

type DecisionAction string

type Utterance struct {
	Content     string
	Page        *agentpagecontext.PageContext
	Mentions    []agentpagecontext.EntityRef
	Surface     agentpagecontext.Surface
	Directed    pulid.ID
	Attachments []string
}
