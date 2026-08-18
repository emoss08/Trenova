package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RunRateSimulationRequest asks for one recorded simulation to be carried out.
type RunRateSimulationRequest struct {
	TenantInfo       pagination.TenantInfo
	RateSimulationID pulid.ID

	// Heartbeat is called between pages of shipments, so a run that takes
	// minutes can report progress and be cancelled part way through rather than
	// only between whole simulations.
	Heartbeat func(done int)
}

// RateSimulationRunner replays a recorded simulation.
//
// It is a port so the Temporal activity that drives a run does not depend on
// the service package that also enqueues one — which would be a cycle, and is
// the same shape the report runner already uses.
type RateSimulationRunner interface {
	Run(ctx context.Context, req *RunRateSimulationRequest) error
}
