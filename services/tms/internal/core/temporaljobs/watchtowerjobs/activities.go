package watchtowerjobs

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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

	Watchtower services.WatchtowerService
	Repo       repositories.WatchtowerRepository
	Tenants    repositories.TenantSyncRepository
	Logger     *zap.Logger
}

type Activities struct {
	watchtower services.WatchtowerService
	repo       repositories.WatchtowerRepository
	tenants    repositories.TenantSyncRepository
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		watchtower: p.Watchtower,
		repo:       p.Repo,
		tenants:    p.Tenants,
		l:          p.Logger.Named("job.watchtower"),
	}
}

// ReconcileActivity corrects every tenant's feed against its sources.
func (a *Activities) ReconcileActivity(ctx context.Context) (*WatchtowerSweepResult, error) {
	return a.sweepAll(ctx, a.watchtower.Reconcile)
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
	sweep, err := a.watchtower.Backfill(ctx, tenant)
	if err != nil {
		return nil, err
	}
	absorb(result, sweep)
	result.OrganizationsProcessed = 1

	return result, nil
}

// RetentionActivity removes items resolved more than a month ago, a batch
// at a time across every tenant.
func (a *Activities) RetentionActivity(ctx context.Context) (*WatchtowerRetentionResult, error) {
	before := timeutils.NowUnix() - retentionDays*24*60*60
	result := &WatchtowerRetentionResult{}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.repo.DeleteResolvedBefore(
			ctx,
			repositories.DeleteResolvedWatchtowerItemsRequest{Before: before, Limit: retentionBatch},
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
	sweep func(context.Context, pagination.TenantInfo) (*services.WatchtowerSweepResult, error),
) (*WatchtowerSweepResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	result := newSweepResult()
	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
		swept, sErr := sweep(ctx, tenant)
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

func absorb(result *WatchtowerSweepResult, swept *services.WatchtowerSweepResult) {
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
