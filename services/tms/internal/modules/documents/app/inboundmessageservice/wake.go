package inboundmessageservice

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

const reconcileGraceSeconds = 5 * 60

func (s *Service) WakeSchedule(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*triagewakecontract.Schedule, error) {
	states, err := s.queueRepo.QueueStates(ctx, inboundmessageports.QueueStatesRequest{
		TenantInfo: tenantInfo,
		IDs:        []pulid.ID{id},
	})
	if err != nil {
		return nil, err
	}

	return scheduleOf(states, id), nil
}

func scheduleOf(states []*inboundmessageports.QueueState, id pulid.ID) *triagewakecontract.Schedule {
	for _, state := range states {
		if state.ID == id && state.HasTimer() {
			return &triagewakecontract.Schedule{Open: true, WakeAt: state.WakeAt()}
		}
	}

	return &triagewakecontract.Schedule{}
}

func dueToWake(now int64) func(*inboundmessageports.QueueState) bool {
	return func(state *inboundmessageports.QueueState) bool {
		return state.HasTimer() && state.WakeAt() <= now
	}
}

func (s *Service) WakeIfDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	now int64,
) (*triagewakecontract.WakeResult, error) {
	result, err := s.runTransition(ctx, &transition{
		tenant:   tenantInfo,
		ids:      []pulid.ID{id},
		eligible: dueToWake(now),
		patch:    inboundmessageports.QueuePatch{Unpark: true},
		activity: func(before, after *inboundmessageports.QueueState) *triage.Activity {
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
		return nil, err
	}
	if len(result.changed) > 0 {
		return &triagewakecontract.WakeResult{Woken: true}, nil
	}

	schedule, err := s.WakeSchedule(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	return &triagewakecontract.WakeResult{Schedule: *schedule}, nil
}

func (s *Service) ReconcileOverdueWakes(ctx context.Context, now int64) (int, error) {
	woken := 0
	for {
		refs, err := s.queueRepo.ListOverdueParked(ctx, inboundmessageports.ListOverdueParkedRequest{
			Now:   now - reconcileGraceSeconds,
			Limit: inboundmessageports.ReconcileBatchSize,
		})
		if err != nil {
			return woken, err
		}

		changed := 0
		for _, ref := range refs {
			tenantInfo := ref.TenantInfo()
			tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
			result, wErr := s.WakeIfDue(tenantCtx, tenantInfo, ref.ID, now)
			if wErr != nil {
				s.l.Error("could not wake an overdue inbound message",
					zap.String("messageId", ref.ID.String()), zap.Error(wErr))
				continue
			}
			if result.Woken {
				changed++
			}
		}
		woken += changed
		if len(refs) < inboundmessageports.ReconcileBatchSize || changed == 0 {
			return woken, nil
		}
	}
}
