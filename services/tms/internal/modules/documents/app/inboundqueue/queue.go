package inboundqueue

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
	"go.uber.org/zap"
)

type Selection struct {
	IDs    []pulid.ID
	Folder *inboundmessageports.ListInboundMessagesRequest
}

type Transition struct {
	Tenant   pagination.TenantInfo
	IDs      []pulid.ID
	Check    func(states []*inboundmessageports.QueueState) error
	Eligible func(state *inboundmessageports.QueueState) bool
	Patch    inboundmessageports.QueuePatch
	PatchFor func(states []*inboundmessageports.QueueState) inboundmessageports.QueuePatch
	Activity func(before, after *inboundmessageports.QueueState) *triage.Activity
}

type TransitionResult struct {
	Before  map[pulid.ID]*inboundmessageports.QueueState
	Changed []*inboundmessageports.QueueState
}

func (r *TransitionResult) ChangedIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.Changed))
	for _, state := range r.Changed {
		ids = append(ids, state.ID)
	}

	return ids
}

func (s *Service) RunTransition(ctx context.Context, t *Transition) (*TransitionResult, error) {
	result := &TransitionResult{}
	if len(t.IDs) == 0 {
		return result, nil
	}

	now := s.now()
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		states, err := s.queueRepo.QueueStates(txCtx, inboundmessageports.QueueStatesRequest{
			TenantInfo: t.Tenant,
			IDs:        t.IDs,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		if t.Check != nil {
			if err = t.Check(states); err != nil {
				return err
			}
		}

		result.Before = make(map[pulid.ID]*inboundmessageports.QueueState, len(states))
		eligible := make([]pulid.ID, 0, len(states))
		for _, state := range states {
			if t.Eligible != nil && !t.Eligible(state) {
				continue
			}
			result.Before[state.ID] = state
			eligible = append(eligible, state.ID)
		}
		if len(eligible) == 0 {
			return nil
		}

		patch := t.Patch
		if t.PatchFor != nil {
			patch = t.PatchFor(states)
		}
		result.Changed, err = s.queueRepo.UpdateQueueState(
			txCtx,
			inboundmessageports.UpdateQueueStateRequest{
				TenantInfo: t.Tenant,
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

	s.afterQueueChange(ctx, t.Tenant, result)

	return result, nil
}

func transitionActivities(t *Transition, result *TransitionResult) []*triage.Activity {
	if t.Activity == nil {
		return nil
	}

	activities := make([]*triage.Activity, 0, len(result.Changed))
	for _, after := range result.Changed {
		if activity := t.Activity(result.Before[after.ID], after); activity != nil {
			activities = append(activities, activity)
		}
	}

	return activities
}

func (s *Service) afterQueueChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	result *TransitionResult,
) {
	if len(result.Changed) == 0 {
		return
	}

	messages := make([]*inboundmessage.InboundMessage, 0, len(result.Changed))
	for _, after := range result.Changed {
		messages = append(messages, inboundmessageports.MessageFromState(after))
	}
	s.notify.MessagesChanged(ctx, messages...)

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, after := range result.Changed {
			s.syncWake(committed, tenantInfo, result.Before[after.ID], after)
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
	result *TransitionResult,
	ruleID *pulid.ID,
) (*inboundmessageports.BulkResult, error) {
	ids := result.ChangedIDs()
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
		Affected: len(result.Changed),
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
