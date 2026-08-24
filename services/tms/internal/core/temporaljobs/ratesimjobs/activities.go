package ratesimjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

type RunSimulationResult struct {
	RateSimulationID pulid.ID `json:"rateSimulationId"`
}

type ActivitiesParams struct {
	fx.In

	Service services.RateSimulationRunner
	Logger  *zap.Logger
}

type Activities struct {
	service services.RateSimulationRunner
	logger  *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		service: p.Service,
		logger:  p.Logger.Named("temporal.rate-simulation"),
	}
}

// RunSimulationActivity replays one simulation.
//
// It heartbeats between pages of shipments, which is what lets a run that takes
// minutes be seen as alive and be cancelled part way through rather than only
// between whole simulations.
func (a *Activities) RunSimulationActivity(
	ctx context.Context,
	payload *RunSimulationPayload,
) (*RunSimulationResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrgID,
		BuID:   payload.BuID,
		UserID: payload.UserID,
	}

	err := a.service.Run(ctx, &services.RunRateSimulationRequest{
		TenantInfo:       tenantInfo,
		RateSimulationID: payload.RateSimulationID,
		Heartbeat: func(done int) {
			recordHeartbeat(ctx, "replaying-shipments", done)
		},
	})
	if err != nil {
		a.logger.Error("rate simulation failed",
			zap.String("rateSimulationId", payload.RateSimulationID.String()),
			zap.Error(err),
		)

		return nil, err
	}

	return &RunSimulationResult{RateSimulationID: payload.RateSimulationID}, nil
}

func recordHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
