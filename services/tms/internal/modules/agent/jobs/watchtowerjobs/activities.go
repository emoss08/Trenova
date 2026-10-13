package watchtowerjobs

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// retentionDays is how long a resolved item stays readable. A month covers
// looking back over a holiday or a leave; past that the source record is
// the record.
const retentionDays = 30

// retentionBatch bounds one delete so a first sweep over a long-running
// deployment does not hold a transaction open across millions of rows.
const retentionBatch = 5000

// retentionPasses bounds how many batches one run deletes, so a sweep that
// finds a very large backlog ends in a predictable time and the next run
// continues where it stopped.
const retentionPasses = 20

type ActivitiesParams struct {
	fx.In

	Watchtower watchtowerports.WatchtowerService
	Exceptions watchtowerports.ExceptionSweeper
	Repo       watchtowerports.WatchtowerRepository
	Tenants    tenantports.TenantSyncRepository
	Controls   dispatchcontrolports.DispatchControlRepository
	Logger     *zap.Logger
}

type Activities struct {
	watchtower watchtowerports.WatchtowerService
	exceptions watchtowerports.ExceptionSweeper
	repo       watchtowerports.WatchtowerRepository
	tenants    tenantports.TenantSyncRepository
	controls   dispatchcontrolports.DispatchControlRepository
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		watchtower: p.Watchtower,
		exceptions: p.Exceptions,
		repo:       p.Repo,
		tenants:    p.Tenants,
		controls:   p.Controls,
		l:          p.Logger.Named("job.watchtower"),
	}
}

// ReconcileActivity corrects every tenant's feed against its sources.
func (a *Activities) ReconcileActivity(ctx context.Context) (*WatchtowerSweepResult, error) {
	return a.sweepAll(ctx, a.watchtower.Reconcile)
}

// NoCheckCallSweepActivity corrects the no-check-call items of every
// organization that watches its brokered loads for check calls. A load goes
// overdue because time passed, not because anything was written, so nothing
// could project it live.
func (a *Activities) NoCheckCallSweepActivity(
	ctx context.Context,
) (*WatchtowerSweepResult, error) {
	tenants, err := a.controls.ListCheckCallMonitoringTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("list check call monitoring tenants: %w", err)
	}

	result := newSweepResult()
	for _, tenant := range tenants {
		activity.RecordHeartbeat(ctx, tenant.OrgID.String())

		tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
		swept, sErr := a.watchtower.ReconcileKind(
			tenantCtx,
			tenant,
			watchtower.SourceNoCheckCall,
		)
		if sErr != nil {
			a.l.Error("no check call sweep failed for organization",
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(sErr),
			)
			result.FailedOrganizations = append(
				result.FailedOrganizations,
				tenant.OrgID.String(),
			)

			continue
		}

		result.OrganizationsProcessed++
		absorb(result, swept)
	}

	a.l.Info("no check call sweep complete",
		zap.Int("organizations", result.OrganizationsProcessed),
		zap.Int("upserted", result.Upserted),
		zap.Int("resolved", result.Resolved),
		zap.Strings("failedSources", result.FailedSources),
	)

	return result, nil
}

// ExceptionAttentionActivity wakes every snoozed exception whose time has
// come and escalates every overdue one below its organization's cap. Both
// happen because time passed, so nothing written could trigger them.
func (a *Activities) ExceptionAttentionActivity(
	ctx context.Context,
) (*ExceptionAttentionResult, error) {
	now := timeutils.NowUnix()
	tenants, err := a.repo.ListTenantsNeedingAttention(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("list tenants needing exception attention: %w", err)
	}

	result := &ExceptionAttentionResult{FailedOrganizations: []string{}}
	for _, tenant := range tenants {
		activity.RecordHeartbeat(ctx, tenant.OrgID.String())

		tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
		swept, sErr := a.exceptions.SweepAttention(tenantCtx, tenant, now)
		if sErr != nil {
			a.l.Error("exception attention sweep failed for organization",
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(sErr),
			)
			result.FailedOrganizations = append(
				result.FailedOrganizations,
				tenant.OrgID.String(),
			)

			continue
		}

		result.OrganizationsProcessed++
		result.Woke += swept.Woke
		result.Escalated += swept.Escalated
		result.Failed += swept.Failed
	}

	if result.Woke > 0 || result.Escalated > 0 || result.Failed > 0 {
		a.l.Info("exception attention sweep complete",
			zap.Int("organizations", result.OrganizationsProcessed),
			zap.Int("woke", result.Woke),
			zap.Int("escalated", result.Escalated),
			zap.Int("failed", result.Failed),
		)
	}

	return result, nil
}

// BackfillActivity fills the named tenant's feed from what its sources
// report open, or every tenant's when no organization is named. It never
// resolves: a backfill adds what is missing and leaves the rest alone.
func (a *Activities) BackfillActivity(
	ctx context.Context,
	input *WatchtowerBackfillInput,
) (*WatchtowerSweepResult, error) {
	if input == nil || input.OrganizationID == "" {
		return a.sweepAll(ctx, a.watchtower.Backfill)
	}

	orgID, err := pulid.MustParse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("parse organization: %w", err)
	}
	buID, err := pulid.MustParse(input.BusinessUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse business unit: %w", err)
	}

	result := newSweepResult()
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
	sweep, err := a.watchtower.Backfill(tenantCtx, tenant)
	if err != nil {
		return nil, err
	}
	absorb(result, sweep)
	result.OrganizationsProcessed = 1

	return result, nil
}

// WatchtowerRetentionActivity removes items resolved more than a month ago, a batch
// at a time across every tenant.
func (a *Activities) WatchtowerRetentionActivity(
	ctx context.Context,
) (*WatchtowerRetentionResult, error) {
	before := timeutils.NowUnix() - retentionDays*24*60*60
	result := &WatchtowerRetentionResult{}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.repo.DeleteResolvedBefore(
			ctx,
			watchtowerports.DeleteResolvedWatchtowerItemsRequest{
				Before: before,
				Limit:  retentionBatch,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("delete resolved watchtower items: %w", err)
		}
		result.Deleted += deleted
		if deleted < retentionBatch {
			break
		}
	}

	a.l.Info("watchtower retention sweep complete", zap.Int("deleted", result.Deleted))

	return result, nil
}

// sweepAll runs one sweep per tenant. A tenant whose sweep fails outright
// is recorded and stepped over, the same way the insight refresh does it:
// the next run picks it up without anyone intervening.
func (a *Activities) sweepAll(
	ctx context.Context,
	sweep func(context.Context, pagination.TenantInfo) (*watchtowerports.WatchtowerSweepResult, error),
) (*WatchtowerSweepResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	result := newSweepResult()
	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
		tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
		swept, sErr := sweep(tenantCtx, tenant)
		if sErr != nil {
			a.l.Error("watchtower sweep failed for organization",
				zap.String("organization", org.ID.String()),
				zap.Error(sErr),
			)
			result.FailedOrganizations = append(result.FailedOrganizations, org.ID.String())

			continue
		}

		result.OrganizationsProcessed++
		absorb(result, swept)
	}

	a.l.Info("watchtower sweep complete",
		zap.Int("organizations", result.OrganizationsProcessed),
		zap.Int("upserted", result.Upserted),
		zap.Int("resolved", result.Resolved),
		zap.Strings("failedSources", result.FailedSources),
	)

	return result, nil
}

func newSweepResult() *WatchtowerSweepResult {
	return &WatchtowerSweepResult{
		FailedOrganizations: make([]string, 0),
		FailedSources:       make([]string, 0),
	}
}

func absorb(result *WatchtowerSweepResult, swept *watchtowerports.WatchtowerSweepResult) {
	if swept == nil {
		return
	}

	result.Upserted += swept.Upserted
	result.Resolved += swept.Resolved
	for _, failed := range swept.Failed {
		if !slices.Contains(result.FailedSources, failed) {
			result.FailedSources = append(result.FailedSources, failed)
		}
	}
}
