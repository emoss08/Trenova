package billingqueuejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/zap"
)

func (a *Activities) RecheckItemActivity(
	ctx context.Context,
	payload *billingqueuecontract.RecheckPayload,
) (*billingqueueports.RecheckResult, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	result, err := a.review.Recheck(ctx, tenant, payload.ItemID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &billingqueueports.RecheckResult{ItemID: payload.ItemID}, nil
		}
		a.l.Warn("billing queue recheck failed",
			zap.String("billingQueueItemId", payload.ItemID.String()),
			zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (a *Activities) RequestStaleRechecksActivity(
	ctx context.Context,
	staleBefore int64,
) (int, error) {
	candidates, err := a.queueRepo.ListRecheckCandidates(
		ctx,
		&billingqueueports.ListRecheckCandidatesRequest{
			StaleBefore: staleBefore,
			Limit:       recheckSweepBatch,
		},
	)
	if err != nil {
		return 0, temporaltype.ToDomainRefusal(err)
	}

	for _, candidate := range candidates {
		a.rechecks.RequestRecheck(ctx, candidate.TenantInfo(), candidate.ID)
	}

	return len(candidates), nil
}

func (a *Activities) AgentReviewActivity(
	ctx context.Context,
	payload *billingqueuecontract.RecheckPayload,
) (*billingqueueports.AgentReviewOutcome, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.ReviewItem(ctx, tenant, payload.ItemID)
}

func (a *Activities) AutoApproveActivity(
	ctx context.Context,
	payload *billingqueuecontract.RecheckPayload,
) (*billingqueueports.AutoApproveOutcome, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	outcome, err := a.assist.AutoApprove(ctx, tenant, payload.ItemID)
	if err != nil && isRefusal(err) {
		return &billingqueueports.AutoApproveOutcome{Reason: err.Error()}, nil
	}

	return outcome, err
}

func (a *Activities) RequestBriefActivity(
	ctx context.Context,
	payload *billingqueuecontract.RecheckPayload,
) error {
	a.briefs.RequestBrief(ctx, payload.Tenant())

	return nil
}

func (a *Activities) RefreshBriefActivity(
	ctx context.Context,
	payload *billingqueuecontract.BriefPayload,
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
