package delegation

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

// Turn is what handing a task to another agent reads of the turn
// that hands it: the agents it may hand to and what it has done so far.
type Turn interface {
	Delegates() []agentturn.RuntimeDelegate
	Result() *agentruntimeports.RunResult
}
