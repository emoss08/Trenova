package carrierintelligencejobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelmonitorservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	tenantPageSize       = 100
	recomputeBudget      = 4 * time.Minute
	refreshBatchPerSweep = 200
	sundayDriftCheckDay  = time.Sunday
)

type ListTenantsPayload struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type TenantPayload struct {
	temporaljobs.TenantWorkItem
}

type SweepTenantResult struct {
	Recomputed int  `json:"recomputed"`
	Enrolled   int  `json:"enrolled"`
	Removed    int  `json:"removed"`
	Changed    int  `json:"changed"`
	Refreshed  int  `json:"refreshed"`
	Paused     bool `json:"paused"`
}

func (r *SweepTenantResult) Total() int {
	return r.Recomputed + r.Enrolled + r.Removed + r.Refreshed
}

type ReconcileTenantPayload struct {
	temporaljobs.TenantWorkItem
	DriftCheck bool `json:"driftCheck"`
}

type ActivitiesParams struct {
	fx.In

	Intel      *carrierintelservice.Service
	Enrollment *carrierintelenrollmentservice.Service
	Monitor    *carrierintelmonitorservice.Service
	Logger     *zap.Logger
}

type Activities struct {
	intel      *carrierintelservice.Service
	enrollment *carrierintelenrollmentservice.Service
	monitor    *carrierintelmonitorservice.Service
	logger     *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		intel:      p.Intel,
		enrollment: p.Enrollment,
		monitor:    p.Monitor,
		logger:     p.Logger.Named("temporal.carrier-intelligence"),
	}
}

func heartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()
	activity.RecordHeartbeat(ctx, details...)
}

func (a *Activities) ListTenantsActivity(
	ctx context.Context,
	payload *ListTenantsPayload,
) (*temporaljobs.TenantPage, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, tenantPageSize)
	after := carrierintelports.CarrierIntelTenantCursor{}
	if payload.After != nil {
		after.OrganizationID = payload.After.OrganizationID
		after.BusinessUnitID = payload.After.BusinessUnitID
	}

	tenants, err := a.monitor.ListConfiguredTenants(ctx, after, limit)
	if err != nil {
		return nil, err
	}
	return &temporaljobs.TenantPage{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
		HasMore: len(tenants) == limit,
	}, nil
}

func (a *Activities) SweepTenantActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*SweepTenantResult, error) {
	tenantInfo := payload.TenantInfo()
	log := a.logger.With(
		zap.String("orgId", tenantInfo.OrgID.String()),
		zap.String("buId", tenantInfo.BuID.String()),
	)
	result := &SweepTenantResult{}
	beat := func(details ...any) { heartbeat(ctx, details...) }

	beat("recompute")
	recomputed, err := a.intel.RecomputeStalePolicy(
		ctx,
		tenantInfo,
		time.Now().Add(recomputeBudget),
	)
	if err != nil {
		log.Warn("carrier intelligence recompute failed", zap.Error(err))
	} else {
		result.Recomputed = recomputed.Evaluated
	}

	beat("enrollment-sync")
	synced, err := a.enrollment.SyncPendingEnrollments(ctx, tenantInfo)
	if err != nil {
		return result, err
	}
	result.Enrolled = synced.Enrolled
	result.Removed = synced.Removed
	result.Paused = synced.Paused

	beat("change-feed")
	polled, err := a.monitor.PollChangeFeed(ctx, tenantInfo, beat)
	if err != nil {
		log.Warn("carrier intelligence change feed poll failed", zap.Error(err))
	} else {
		result.Changed = polled.Changed
		result.Refreshed += polled.Refreshed
		result.Paused = result.Paused || polled.Paused
	}

	beat("snapshot-refresh")
	refreshed, err := a.monitor.RefreshDueSnapshots(ctx, tenantInfo, refreshBatchPerSweep, beat)
	if err != nil {
		log.Warn("carrier intelligence snapshot refresh failed", zap.Error(err))
	} else {
		result.Refreshed += refreshed.Refreshed
		result.Paused = result.Paused || refreshed.Paused
	}

	return result, nil
}

func (a *Activities) ReconcileTenantActivity(
	ctx context.Context,
	payload *ReconcileTenantPayload,
) (*carrierintelenrollmentservice.ReconcileResult, error) {
	tenantInfo := payload.TenantInfo()
	heartbeat(ctx, "reconcile")
	result, err := a.enrollment.ReconcileDesired(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !payload.DriftCheck {
		return result, nil
	}
	heartbeat(ctx, "drift-check")
	drift, err := a.enrollment.DriftCheck(ctx, tenantInfo)
	if err != nil {
		a.logger.Warn("carrier intelligence watchlist drift check failed", zap.Error(err),
			zap.String("orgId", tenantInfo.OrgID.String()))
		return result, nil
	}
	result.DriftAdded = drift.DriftAdded
	result.DriftRemove = drift.DriftRemove
	return result, nil
}

func (a *Activities) DigestTenantActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*carrierintelmonitorservice.DigestResult, error) {
	return a.monitor.SendDigest(ctx, payload.TenantInfo())
}

func (a *Activities) GlobalMaintenanceActivity(
	ctx context.Context,
) (*carrierintelmonitorservice.MaintenanceResult, error) {
	heartbeat(ctx, "maintenance")
	return a.monitor.RunGlobalMaintenance(ctx)
}

func (a *Activities) PruneTenantActivity(ctx context.Context, payload *TenantPayload) (int, error) {
	return a.monitor.PruneTenantHistory(ctx, payload.TenantInfo())
}
