package samsarajobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	SyncService *samsarasyncservice.Service
}

type Activities struct {
	syncService *samsarasyncservice.Service
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		syncService: p.SyncService,
	}
}

func (a *Activities) SyncWorkersToSamsaraActivity(
	ctx context.Context,
	payload *WorkersSyncWorkflowPayload,
) (*WorkersSyncWorkflowResult, error) {
	logger := activity.GetLogger(ctx)
	if payload == nil {
		logger.Error("Samsara sync payload is required")
		return nil, errInvalidPayload
	}

	result, err := a.syncService.SyncWorkersToSamsara(ctx, pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	})
	if err != nil {
		logger.Error("Samsara worker sync activity failed", "error", err)
		return nil, err
	}

	logger.Info("Samsara worker sync activity completed")
	return &WorkersSyncWorkflowResult{
		Result: result,
	}, nil
}
