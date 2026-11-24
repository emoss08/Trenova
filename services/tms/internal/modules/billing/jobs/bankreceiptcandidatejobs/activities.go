package bankreceiptcandidatejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/bankreceiptcandidatejobs/bankreceiptcandidatecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Entitlements entitlementports.Filter
	Refresher    bankreceiptports.CandidateRefresher
	Candidates   bankreceiptports.CandidateRepository
	Queue        workqueueports.Queue
	Requester    *bankreceiptcandidatecontract.Requester
	Logger       *zap.Logger
}

type Activities struct {
	entitlements entitlementports.Filter
	refresher    bankreceiptports.CandidateRefresher
	candidates   bankreceiptports.CandidateRepository
	queue        workqueueports.Queue
	requester    *bankreceiptcandidatecontract.Requester
	l            *zap.Logger
}

//nolint:gocritic // fx passes its In struct by value
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		entitlements: p.Entitlements,
		refresher:    p.Refresher,
		candidates:   p.Candidates,
		queue:        p.Queue,
		requester:    p.Requester,
		l:            p.Logger.Named("temporal.bank-receipt-candidate-activities"),
	}
}

func (a *Activities) DrainBankReceiptCandidatesActivity(
	ctx context.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainPage, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return temporaljobs.DrainDue(ctx, &temporaljobs.DrainDueParams{
		Need:         entitlement.For(platformcatalog.FeatureAccounting),
		Entitlements: a.entitlements,
		Queue:        a.queue,
		Kind:         bankreceiptcandidatecontract.DrainKind,
		Tenant:       tenant,
		PageSize:     drainPageSize,
		RetryAfter:   drainRetryAfter,
		Handle: func(ctx context.Context, subjects []pulid.ID, _ map[pulid.ID]string) map[pulid.ID]error {
			if _, err := a.refresher.Refresh(ctx, tenant, subjects); err != nil {
				a.l.Warn("bank receipt candidates could not be refreshed; the page is tried again",
					zap.String("organizationId", tenant.OrgID.String()),
					zap.Int("subjects", len(subjects)),
					zap.Error(err))

				return allFailed(subjects, err)
			}

			return nil
		},
	})
}

func (a *Activities) WakeDueBankReceiptCandidateDrainsActivity(ctx context.Context) (int, error) {
	return a.requester.WakeDue(ctx, drainWakeLimit)
}

func (a *Activities) QueueUnappliedBankReceiptsActivity(ctx context.Context) (int64, error) {
	return a.candidates.QueueUnapplied(ctx, &bankreceiptports.QueueUnappliedRequest{
		Kind: bankreceiptcandidatecontract.DrainKind,
	})
}

func allFailed(ids []pulid.ID, err error) map[pulid.ID]error {
	failed := make(map[pulid.ID]error, len(ids))
	for _, id := range ids {
		failed[id] = err
	}

	return failed
}
