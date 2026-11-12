package inboundqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var _ triagewakecontract.Worker = (*Service)(nil)

func dueToWake(now int64) func(*inboundmessageports.QueueState) bool {
	return func(state *inboundmessageports.QueueState) bool {
		return state.HasTimer() && state.WakeAt() <= now
	}
}

// WakeIfDue brings back the tenant's parked messages among ids whose time has
// come, in one transition, and reports how many it woke. A message woken
// already, or whose time moved on, is left as it is.
func (s *Service) WakeIfDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	now int64,
) (int, error) {
	result, err := s.RunTransition(ctx, &Transition{
		Tenant:   tenantInfo,
		IDs:      ids,
		Eligible: dueToWake(now),
		Patch:    inboundmessageports.QueuePatch{Unpark: true},
		Activity: func(before, after *inboundmessageports.QueueState) *triage.Activity {
			if before.Parked == inboundmessage.ParkingAwaitingReply {
				activity := activityOf(after, triage.ActivityStoppedWaiting).
					About(before.FromAddress, "no reply")
				activity.Until = before.AwaitingUntil

				return activity
			}
			activity := activityOf(after, triage.ActivityWoken)
			activity.Until = before.SnoozedUntil

			return activity
		},
	})
	if err != nil {
		return 0, err
	}

	return len(result.Changed), nil
}

// WakeDue wakes, in every tenant, the snoozed and waiting messages whose time
// has come. It is the only timer: the minute sweep calls it, reading the
// parked columns, so a message holds no workflow while it waits. Each page is
// one transition per tenant.
func (s *Service) WakeDue(ctx context.Context, now int64) (int, error) {
	woken := 0
	for {
		refs, err := s.queueRepo.ListOverdueParked(
			ctx,
			inboundmessageports.ListOverdueParkedRequest{
				Now:   now,
				Limit: inboundmessageports.ReconcileBatchSize,
			},
		)
		if err != nil {
			return woken, err
		}

		changed := 0
		for tenantInfo, ids := range parkedByTenant(refs) {
			tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
			n, wErr := s.WakeIfDue(tenantCtx, tenantInfo, ids, now)
			if wErr != nil {
				s.l.Error("could not wake a tenant's due inbound messages",
					zap.String("organizationId", tenantInfo.OrgID.String()),
					zap.Int("messages", len(ids)),
					zap.Error(wErr))
				continue
			}
			changed += n
		}
		woken += changed
		if len(refs) < inboundmessageports.ReconcileBatchSize || changed == 0 {
			return woken, nil
		}
	}
}

func parkedByTenant(refs []inboundmessageports.ParkedRef) map[pagination.TenantInfo][]pulid.ID {
	byTenant := make(map[pagination.TenantInfo][]pulid.ID)
	for _, ref := range refs {
		tenant := ref.TenantInfo()
		byTenant[tenant] = append(byTenant[tenant], ref.ID)
	}

	return byTenant
}
