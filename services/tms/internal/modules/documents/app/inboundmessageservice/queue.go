package inboundmessageservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type Selection struct {
	IDs    []pulid.ID
	Folder *inboundmessageports.ListInboundMessagesRequest
}

type transition struct {
	tenant   pagination.TenantInfo
	ids      []pulid.ID
	check    func(states []*inboundmessageports.QueueState) error
	eligible func(state *inboundmessageports.QueueState) bool
	patch    inboundmessageports.QueuePatch
	patchFor func(states []*inboundmessageports.QueueState) inboundmessageports.QueuePatch
	activity func(before, after *inboundmessageports.QueueState) *triage.Activity
}

type transitionResult struct {
	before  map[pulid.ID]*inboundmessageports.QueueState
	changed []*inboundmessageports.QueueState
}

func (r *transitionResult) changedIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.changed))
	for _, state := range r.changed {
		ids = append(ids, state.ID)
	}

	return ids
}

func (s *Service) runTransition(ctx context.Context, t *transition) (*transitionResult, error) {
	result := &transitionResult{}
	if len(t.ids) == 0 {
		return result, nil
	}

	now := s.now()
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		states, err := s.queueRepo.QueueStates(txCtx, inboundmessageports.QueueStatesRequest{
			TenantInfo: t.tenant,
			IDs:        t.ids,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		if t.check != nil {
			if err = t.check(states); err != nil {
				return err
			}
		}

		result.before = make(map[pulid.ID]*inboundmessageports.QueueState, len(states))
		eligible := make([]pulid.ID, 0, len(states))
		for _, state := range states {
			if t.eligible != nil && !t.eligible(state) {
				continue
			}
			result.before[state.ID] = state
			eligible = append(eligible, state.ID)
		}
		if len(eligible) == 0 {
			return nil
		}

		patch := t.patch
		if t.patchFor != nil {
			patch = t.patchFor(states)
		}
		result.changed, err = s.queueRepo.UpdateQueueState(
			txCtx,
			inboundmessageports.UpdateQueueStateRequest{
				TenantInfo: t.tenant,
				IDs:        eligible,
				Patch:      patch,
				Now:        now,
			},
		)
		if err != nil {
			return err
		}

		return s.activityRepo.Insert(txCtx, transitionActivities(t, result))
	})
	if err != nil {
		return nil, err
	}

	s.afterQueueChange(ctx, t.tenant, result)

	return result, nil
}

func transitionActivities(t *transition, result *transitionResult) []*triage.Activity {
	if t.activity == nil {
		return nil
	}

	activities := make([]*triage.Activity, 0, len(result.changed))
	for _, after := range result.changed {
		if activity := t.activity(result.before[after.ID], after); activity != nil {
			activities = append(activities, activity)
		}
	}

	return activities
}

func (s *Service) afterQueueChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	result *transitionResult,
) {
	if len(result.changed) == 0 {
		return
	}

	messages := make([]*inboundmessage.InboundMessage, 0, len(result.changed))
	for _, after := range result.changed {
		messages = append(messages, messageFromState(after))
	}
	s.notify.MessagesChanged(ctx, messages...)

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, after := range result.changed {
			s.syncWake(committed, tenantInfo, result.before[after.ID], after)
		}
	})
}

func (s *Service) syncWake(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	before, after *inboundmessageports.QueueState,
) {
	if s.wakes == nil || !s.wakes.Enabled() {
		if after.HasTimer() {
			s.l.Warn("a parked inbound message has no worker to wake it",
				zap.String("messageId", after.ID.String()))
		}

		return
	}

	switch {
	case after.HasTimer():
		if before != nil && before.Parked == after.Parked && before.WakeAt() == after.WakeAt() {
			return
		}
		if err := s.wakes.ScheduleWake(ctx, tenantInfo, after.ID); err != nil {
			s.l.Error("could not schedule an inbound message's wake-up; the hourly reconcile will",
				zap.String("messageId", after.ID.String()), zap.Error(err))
		}
	case before != nil && before.HasTimer():
		if err := s.wakes.CancelWake(ctx, after.ID); err != nil {
			s.l.Warn("could not cancel an inbound message's wake-up; it will find nothing to do",
				zap.String("messageId", after.ID.String()), zap.Error(err))
		}
	}
}

func messageFromState(state *inboundmessageports.QueueState) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                   state.ID,
		OrganizationID:       state.OrganizationID,
		BusinessUnitID:       state.BusinessUnitID,
		MailboxID:            state.MailboxID,
		Status:               state.Status,
		Classification:       state.Classification,
		ReceivedAt:           state.ReceivedAt,
		FromAddress:          state.FromAddress,
		FromName:             state.FromName,
		Subject:              state.Subject,
		SenderDomain:         state.SenderDomain,
		ThreadKey:            state.ThreadKey,
		OwnerID:              state.OwnerID,
		DueAt:                state.DueAt,
		Parked:               state.Parked,
		SnoozedUntil:         state.SnoozedUntil,
		SnoozeEvent:          state.SnoozeEvent,
		SnoozeWatchID:        state.SnoozeWatchID,
		SnoozeDocumentTypeID: state.SnoozeDocumentTypeID,
		AwaitingUntil:        state.AwaitingUntil,
		MatchedShipmentID:    state.MatchedShipmentID,
		MatchedCustomerID:    state.MatchedCustomerID,
		MatchedCarrierID:     state.MatchedCarrierID,
		MatchReason:          state.MatchReason,
		FailureText:          state.FailureText,
		ReviewNote:           state.ReviewNote,
		NovaRead:             state.NovaRead,
	}
}

func (s *Service) resolveSelection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	selection Selection,
) ([]pulid.ID, error) {
	if len(selection.IDs) > 0 {
		ids := sliceutils.Dedupe(selection.IDs)
		if len(ids) > inboundmessageports.MaxBulkSelection {
			return nil, errortypes.NewValidationError(
				"selection.ids",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"Choose at most %d messages at once",
					inboundmessageports.MaxBulkSelection,
				),
			)
		}

		return ids, nil
	}
	if selection.Folder == nil {
		return nil, errortypes.NewValidationError("selection", errortypes.ErrRequired,
			"Choose some messages or a folder")
	}

	folder := *selection.Folder
	if folder.Filter == nil {
		folder.Filter = &pagination.QueryOptions{}
	}
	filter := *folder.Filter
	filter.TenantInfo = tenantInfo
	folder.Filter = &filter

	return s.queueRepo.SelectIDs(ctx, inboundmessageports.SelectInboundMessageIDsRequest{
		Folder: &folder,
		Limit:  inboundmessageports.MaxBulkSelection,
	})
}

func (s *Service) bulkResult(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	result *transitionResult,
	ruleID *pulid.ID,
) (*inboundmessageports.BulkResult, error) {
	ids := result.changedIDs()
	if len(ids) > inboundmessageports.MaxBulkResultMessages {
		ids = ids[:inboundmessageports.MaxBulkResultMessages]
	}

	messages, err := s.queueRepo.ListByIDs(ctx, inboundmessageports.ListInboundMessagesByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}

	return &inboundmessageports.BulkResult{
		Messages: messages,
		Affected: len(result.changed),
		RuleID:   ruleID,
	}, nil
}

func (s *Service) messageAfter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	return s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 id,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
}

func requireFound(states []*inboundmessageports.QueueState, id pulid.ID) error {
	if len(states) == 0 ||
		!slices.ContainsFunc(states, func(state *inboundmessageports.QueueState) bool {
			return state.ID == id
		}) {
		return errortypes.NewNotFoundError("Inbound message not found within your organization")
	}

	return nil
}

func byPerson(activity *triage.Activity, actorID pulid.ID) *triage.Activity {
	if actorID.IsNil() {
		return activity
	}

	return activity.ByPerson(actorID)
}
