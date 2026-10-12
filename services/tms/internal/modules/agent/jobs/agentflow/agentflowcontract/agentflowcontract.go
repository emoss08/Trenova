package agentflowcontract

import (
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
)

const (
	failureStopped    = "cancelled"
	failureTimedOut   = "timeout"
	failureNoProvider = "no_provider"
	failureResting    = "providers_resting"
	failureRefused    = "refused"
	failureSchema     = "schema_invalid"
	failureOther      = "failed"
	failureHTTPPrefix = "http_"
)

// DrainWindow is how long a finished run waits for its reader to say it
// has the last event. A reader that is there says so within a poll or
// two; one that is not is not coming, and the run closes without it.
const DrainWindow = 15 * time.Second

const PriorityInteractive = 1

const PriorityOneShot = 2

const PriorityBackground = 3

// Priority keys order a queue's work: lower runs first. They matter only
// where kinds of work share a queue, and fairness keys by organization then
// share each priority out between tenants.
const (
	PriorityEvaluation = 5
)

// SteerSignal carries something the person said to a turn while it works.
// The turn reads it at its next step.
const SteerSignal = "assistant-turn-steer"

func FailureKind(failure *modelcallcontract.Failure) string {
	switch {
	case failure == nil:
		return ""
	case failure.Stopped:
		return failureStopped
	case failure.TimedOut:
		return failureTimedOut
	case failure.NoProvider:
		return failureNoProvider
	case failure.Resting:
		return failureResting
	case failure.Refusal != nil:
		return failureRefused
	case failure.SchemaInvalid:
		return failureSchema
	case failure.Status != 0:
		return failureHTTPPrefix + strconv.Itoa(failure.Status)
	default:
		return failureOther
	}
}
