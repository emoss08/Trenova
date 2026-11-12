package billingqueuejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// DrainRechecksActivity checks one page of the tenant's due items: the
// billing checks, then Nova's review, then auto-approval of an item found
// ready. Each item settles on its own, so a retry after a partial page picks
// up where it stopped. An item that fails is made due again later rather
// than failing the page.
func (a *Activities) DrainRechecksActivity(
	ctx context.Context,
	payload *billingqueuecontract.TenantPayload,
) (*DrainRechecksResult, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	due, err := a.queueRepo.ListRecheckDue(ctx, &billingqueueports.ListRecheckDueRequest{
		TenantInfo: tenant,
		Now:        timeutils.NowUnix(),
		Limit:      drainPageSize,
	})
	if err != nil {
		return nil, err
	}

	result := &DrainRechecksResult{Full: len(due) == drainPageSize}
	for _, item := range due {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		activity.RecordHeartbeat(ctx, item.ID.String())

		retryAt := a.recheckItem(ctx, tenant, item.ID, result)
		if err = a.queueRepo.SettleRecheck(ctx, &billingqueueports.SettleRecheckRequest{
			TenantInfo:   tenant,
			ItemID:       item.ID,
			ClaimedDueAt: item.DueAt,
			RetryAt:      retryAt,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// recheckItem runs one item's checks, review and auto-approval. It returns
// when the item should be due again, or nil when it is settled.
func (a *Activities) recheckItem(
	ctx context.Context,
	tenant pagination.TenantInfo,
	itemID pulid.ID,
	result *DrainRechecksResult,
) *int64 {
	log := a.l.With(zap.String("billingQueueItemId", itemID.String()))

	checked, err := a.review.Recheck(ctx, tenant, itemID)
	if errortypes.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		log.Warn("billing queue recheck failed; it will be checked again", zap.Error(err))
		result.Failed++

		return retryAfter(recheckRetryAfter.Seconds())
	}
	result.Checked++

	var retryAt *int64
	if _, err = a.assist.ReviewItem(ctx, tenant, itemID); err != nil {
		log.Warn("billing queue agent review failed; it will be reviewed again", zap.Error(err))
		result.Failed++
		retryAt = retryAfter(reviewRetryAfter.Seconds())
	}

	if checked.Ready {
		if _, err = a.autoApprove(ctx, tenant, itemID); err != nil {
			log.Warn("billing queue auto-approve failed", zap.Error(err))
		}
	}

	return retryAt
}

func retryAfter(seconds float64) *int64 {
	at := timeutils.NowUnix() + int64(seconds)

	return &at
}

// SweepRechecksActivity marks the items whose checks are missing or stale as
// due, then wakes each tenant that has due items.
func (a *Activities) SweepRechecksActivity(
	ctx context.Context,
	staleBefore int64,
) (*RecheckSweepResult, error) {
	now := timeutils.NowUnix()
	marked, err := a.queueRepo.MarkStaleRecheckDue(ctx, &billingqueueports.MarkStaleRecheckDueRequest{
		StaleBefore: staleBefore,
		Now:         now,
		Limit:       recheckSweepItems,
	})
	if err != nil {
		return nil, err
	}

	tenants, err := a.queueRepo.ListRecheckDueTenants(
		ctx,
		&billingqueueports.ListRecheckDueTenantsRequest{Now: now, Limit: recheckSweepTenants},
	)
	if err != nil {
		return nil, err
	}
	for _, tenant := range tenants {
		a.rechecks.WakeTenant(ctx, tenant)
	}

	return &RecheckSweepResult{Marked: marked, Tenants: len(tenants)}, nil
}

func (a *Activities) AgentReviewActivity(
	ctx context.Context,
	payload *billingqueuecontract.ItemPayload,
) (*billingqueueports.AgentReviewOutcome, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.ReviewItem(ctx, tenant, payload.ItemID)
}

func (a *Activities) AutoApproveActivity(
	ctx context.Context,
	payload *billingqueuecontract.ItemPayload,
) (*billingqueueports.AutoApproveOutcome, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.autoApprove(ctx, tenant, payload.ItemID)
}

// autoApprove approves an item found ready. A refusal is an outcome with its
// reason, not an error.
func (a *Activities) autoApprove(
	ctx context.Context,
	tenant pagination.TenantInfo,
	itemID pulid.ID,
) (*billingqueueports.AutoApproveOutcome, error) {
	outcome, err := a.assist.AutoApprove(ctx, tenant, itemID)
	if err != nil && isRefusal(err) {
		return &billingqueueports.AutoApproveOutcome{Reason: err.Error()}, nil
	}

	return outcome, err
}

func (a *Activities) RefreshBriefActivity(
	ctx context.Context,
	payload *billingqueuecontract.TenantPayload,
) error {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	written, err := a.assist.RefreshBrief(ctx, tenant)
	if err != nil || !written {
		return err
	}
	a.publishTenantChanged(ctx, tenant, permission.ResourceBillingQueue.String())

	return nil
}
