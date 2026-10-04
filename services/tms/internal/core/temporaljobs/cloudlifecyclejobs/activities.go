package cloudlifecyclejobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/cloudlifecycleservice"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	purgeBatchSize      = 500
	purgeMaxBatches     = 40
	maxPurgeRowsPasses  = 2_000
	sweepHeartbeatLabel = "sweeping cloud subscriptions"
)

type ActivitiesParams struct {
	fx.In

	Lifecycle *cloudlifecycleservice.Service
	Logger    *zap.Logger
}

type Activities struct {
	lifecycle *cloudlifecycleservice.Service
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		lifecycle: p.Lifecycle,
		l:         p.Logger.Named("cloud-lifecycle-activities"),
	}
}

func (a *Activities) SweepCloudSubscriptionsActivity(
	ctx context.Context,
	input *SweepInput,
) (*cloudlifecycleservice.SweepResult, error) {
	recordHeartbeat(ctx, sweepHeartbeatLabel)

	result, err := a.lifecycle.Sweep(ctx, input.Now)
	if err != nil {
		return nil, fmt.Errorf("sweep cloud subscriptions: %w", err)
	}

	a.l.Info("cloud subscription sweep finished",
		zap.Int("examined", result.Examined),
		zap.Int("readOnly", result.ReadOnly),
		zap.Int("expired", result.Expired),
		zap.Int("failed", result.Failed),
		zap.Int("purgeTargets", len(result.PurgeTargets)),
	)

	return result, nil
}

func (a *Activities) CheckCloudTenantPurgeActivity(
	ctx context.Context,
	payload *PurgePayload,
) (*PurgeEligibility, error) {
	if err := a.lifecycle.RequireExpired(ctx, payload.ref()); err != nil {
		if errors.Is(err, cloudlifecycleservice.ErrNotExpired) {
			return &PurgeEligibility{Members: []*repositories.TenantMember{}}, nil
		}
		return nil, err
	}

	members, err := a.lifecycle.ListMembers(ctx, payload.ref())
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return &PurgeEligibility{Eligible: true, Members: members}, nil
}

func (a *Activities) PurgeCloudTenantRowsActivity(
	ctx context.Context,
	payload *PurgePayload,
) (*PurgeRowsResult, error) {
	result := &PurgeRowsResult{}
	for result.Passes < maxPurgeRowsPasses {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		pass, err := a.lifecycle.PurgeRows(ctx, payload.ref(), purgeBatchSize, purgeMaxBatches)
		if err != nil {
			return nil, fmt.Errorf("purge tenant rows: %w", err)
		}
		result.Passes++
		result.Deleted += pass.Deleted
		result.Retained = pass.Retained
		result.Blocked = pass.Blocked
		result.Complete = pass.Complete
		recordHeartbeat(ctx, result)

		if pass.Complete || pass.Deleted == 0 {
			break
		}
	}

	if !result.Complete {
		a.l.Warn("tenant purge left rows it could not delete",
			zap.String("organizationId", payload.OrganizationID.String()),
			zap.Strings("blocked", result.Blocked),
			zap.Strings("retained", result.Retained),
		)
	}

	return result, nil
}

func (a *Activities) PurgeCloudTenantStorageActivity(
	ctx context.Context,
	payload *PurgePayload,
) (*PurgeStorageResult, error) {
	recordHeartbeat(ctx, "deleting stored objects")

	deleted, err := a.lifecycle.PurgeStorage(ctx, payload.ref())
	if err != nil {
		if errors.Is(err, cloudlifecycleservice.ErrStorageUnsupported) {
			a.l.Warn("storage backend cannot delete by prefix; stored objects were left",
				zap.String("organizationId", payload.OrganizationID.String()),
			)
			return &PurgeStorageResult{Skipped: true}, nil
		}
		return nil, fmt.Errorf("purge stored objects: %w", err)
	}

	return &PurgeStorageResult{Deleted: deleted}, nil
}

func (a *Activities) PurgeCloudTenantUsersActivity(
	ctx context.Context,
	input *PurgeUsersInput,
) (*cloudlifecycleservice.PurgeUsersResult, error) {
	recordHeartbeat(ctx, "removing users")

	result, err := a.lifecycle.PurgeUsers(ctx, input.ref(), input.UserIDs)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Activities) FinalizeCloudTenantPurgeActivity(
	ctx context.Context,
	payload *PurgePayload,
) (*repositories.DeleteTenantResult, error) {
	result, err := a.lifecycle.Finalize(ctx, payload.ref())
	if err != nil {
		return nil, fmt.Errorf("delete tenant: %w", err)
	}

	if !result.OrganizationDeleted {
		a.l.Info("expired organization kept until its retained audit rows age out",
			zap.String("organizationId", payload.OrganizationID.String()),
			zap.String("reason", result.RetainedReason),
		)
	}

	return result, nil
}

func recordHeartbeat(ctx context.Context, details any) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, details)
	}
}
