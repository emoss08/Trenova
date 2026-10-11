package ratesimcontract

import (
	"github.com/emoss08/trenova/shared/pulid"
)

// RunSimulationPayload names the simulation to carry out.
//
// It carries an id rather than the simulation itself: a Temporal payload is
// serialized and replayed, and a stale copy of a run's own state is the last
// thing that should decide what it does.
type RunSimulationPayload struct {
	OrgID            pulid.ID `json:"orgId"`
	BuID             pulid.ID `json:"buId"`
	UserID           pulid.ID `json:"userId"`
	RateSimulationID pulid.ID `json:"rateSimulationId"`
}

const RunRateSimulationWorkflowName = "RunRateSimulationWorkflow"
