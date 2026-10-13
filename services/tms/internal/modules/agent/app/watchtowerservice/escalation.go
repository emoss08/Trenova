package watchtowerservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const attentionBatch = 200

var _ watchtowerports.ExceptionSweeper = (*Exceptions)(nil)

func AsExceptionSweeper(e *Exceptions) watchtowerports.ExceptionSweeper { return e }

func (s *Exceptions) SweepAttention(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
) (*watchtowerports.ExceptionSweepResult, error) {
	result := &watchtowerports.ExceptionSweepResult{}

	waking, err := s.repo.ListSnoozeExpired(
		ctx,
		watchtowerports.ListAttentionWatchtowerItemsRequest{
			TenantInfo: tenant,
			Now:        now,
			Limit:      attentionBatch,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, item := range waking {
		if s.wake(ctx, item) {
			result.Woke++
		} else {
			result.Failed++
		}
	}

	policy := s.router.Policy(ctx, tenant)
	if policy.MaxEscalations <= 0 {
		return result, nil
	}
	overdue, err := s.repo.ListOverdue(ctx, watchtowerports.ListAttentionWatchtowerItemsRequest{
		TenantInfo:         tenant,
		Now:                now,
		Limit:              attentionBatch,
		BelowEscalationCap: policy.MaxEscalations,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range overdue {
		if s.escalate(ctx, item, policy, now) {
			result.Escalated++
		} else {
			result.Failed++
		}
	}

	return result, nil
}

func (s *Exceptions) wake(ctx context.Context, item *watchtower.Item) bool {
	woken := *item
	until := formatOptionalTime(item.SnoozedUntil)
	if !woken.Wake() {
		return true
	}
	saved, err := s.repo.UpdateWorkflow(ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &woken,
		ExpectedVersion: item.Version,
		Events: []*watchtower.Event{
			watchtower.NewEvent(&woken, watchtower.EventWoke, pulid.Nil).
				WithChange(until, woken.Status.String()),
		},
	})
	if err != nil {
		s.l.Warn("could not wake a snoozed exception",
			zap.String("item", item.ID.String()), zap.Error(err))
		return false
	}
	s.projector.publish(ctx, saved, "updated")
	if saved.AssigneeID.IsNotNil() && s.notifications != nil {
		sendExceptionNotice(ctx, s.notifications, s.l, &exceptionNotice{
			item:      saved,
			userID:    saved.AssigneeID,
			eventType: EventAssigned,
			title:     "Snoozed exception is back: " + saved.Title,
			priority:  noticePriority(saved.Severity),
		})
	}
	return true
}

func (s *Exceptions) escalate(
	ctx context.Context,
	item *watchtower.Item,
	policy dispatchcontrol.ExceptionPolicy,
	now int64,
) bool {
	escalated := *item
	escalated.EscalationLevel++
	escalated.LastEscalatedAt = &now
	escalated.Severity = item.Severity.Raised()
	due := now + int64(policy.EscalationMinutes)*60
	escalated.DueAt = &due

	saved, err := s.repo.UpdateWorkflow(ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &escalated,
		ExpectedVersion: item.Version,
		Events: []*watchtower.Event{
			watchtower.NewEvent(&escalated, watchtower.EventEscalated, pulid.Nil).
				WithChange(item.Severity.String(), escalated.Severity.String()).
				WithNote("Escalation " + strconv.Itoa(int(escalated.EscalationLevel)) +
					" of " + strconv.Itoa(int(policy.MaxEscalations))),
		},
	})
	if err != nil {
		s.l.Warn("could not escalate an overdue exception",
			zap.String("item", item.ID.String()), zap.Error(err))
		return false
	}
	s.projector.publish(ctx, saved, "updated")
	s.notifyEscalation(ctx, saved, policy.EscalationContact)
	return true
}

func (s *Exceptions) notifyEscalation(
	ctx context.Context,
	item *watchtower.Item,
	contact pulid.ID,
) {
	if s.notifications == nil {
		return
	}
	title := "Overdue exception escalated: " + item.Title
	recipients := make([]pulid.ID, 0, 2)
	if item.AssigneeID.IsNotNil() {
		recipients = append(recipients, item.AssigneeID)
	}
	if contact.IsNotNil() && contact != item.AssigneeID {
		ok, err := mayWorkException(ctx, s.access, item, pagination.TenantInfo{
			OrgID:  item.OrganizationID,
			BuID:   item.BusinessUnitID,
			UserID: contact,
		})
		if err != nil {
			s.l.Warn("could not check the escalation contact",
				zap.String("item", item.ID.String()), zap.Error(err))
		}
		if ok {
			recipients = append(recipients, contact)
		}
	}
	for _, recipient := range recipients {
		sendExceptionNotice(ctx, s.notifications, s.l, &exceptionNotice{
			item:      item,
			userID:    recipient,
			eventType: EventEscalated,
			title:     title,
			priority:  notification.PriorityHigh,
		})
	}
}
