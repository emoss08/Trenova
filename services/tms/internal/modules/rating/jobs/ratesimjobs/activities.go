package ratesimjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/jobs/ratesimjobs/ratesimcontract"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratesimulationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RunSimulationResult struct {
	RateSimulationID pulid.ID `json:"rateSimulationId"`
}

type ActivitiesParams struct {
	fx.In

	Service ratesimulationports.RateSimulationRunner
	Logger  *zap.Logger
}

type Activities struct {
	service ratesimulationports.RateSimulationRunner
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
	payload *ratesimcontract.RunSimulationPayload,
) (*RunSimulationResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrgID,
		BuID:   payload.BuID,
		UserID: payload.UserID,
	}

	err := a.service.Run(ctx, &ratesimulationports.RunRateSimulationRequest{
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
