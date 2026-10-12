package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"go.uber.org/fx"
)

var _ inboundmessageports.TriageActivityRecorder = (*ActivityRecorder)(nil)

type ActivityRecorderParams struct {
	fx.In

	Repo inboundmessageports.TriageActivityRepository
}

type ActivityRecorder struct {
	repo inboundmessageports.TriageActivityRepository
}

func NewActivityRecorder(p ActivityRecorderParams) *ActivityRecorder {
	return &ActivityRecorder{repo: p.Repo}
}

func (r *ActivityRecorder) RecordActivity(
	ctx context.Context,
	activities ...*triage.Activity,
) error {
	return r.repo.Insert(ctx, activities)
}

func activityOf(
	state *inboundmessageports.QueueState,
	kind triage.ActivityKind,
) *triage.Activity {
	return &triage.Activity{
		OrganizationID: state.OrganizationID,
		BusinessUnitID: state.BusinessUnitID,
		MessageID:      state.ID,
		Kind:           kind,
		ActorKind:      triage.ActorSystem,
	}
}

func (s *Service) recordActivity(ctx context.Context, activities ...*triage.Activity) {
	if err := s.activityRepo.Insert(ctx, activities); err != nil {
		s.l.Warn("could not record a line of a message's activity", zap.Error(err))
	}
}

func queueStateOf(message *inboundmessage.InboundMessage) *inboundmessageports.QueueState {
	return &inboundmessageports.QueueState{
		ID:                   message.ID,
		OrganizationID:       message.OrganizationID,
		BusinessUnitID:       message.BusinessUnitID,
		MailboxID:            message.MailboxID,
		Status:               message.Status,
		Classification:       message.Classification,
		ReceivedAt:           message.ReceivedAt,
		FromAddress:          message.FromAddress,
		FromName:             message.FromName,
		Subject:              message.Subject,
		SenderDomain:         message.SenderDomain,
		ThreadKey:            message.ThreadKey,
		OwnerID:              message.OwnerID,
		DueAt:                message.DueAt,
		Parked:               message.Parked,
		SnoozedUntil:         message.SnoozedUntil,
		SnoozeEvent:          message.SnoozeEvent,
		SnoozeWatchID:        message.SnoozeWatchID,
		SnoozeDocumentTypeID: message.SnoozeDocumentTypeID,
		AwaitingUntil:        message.AwaitingUntil,
		MatchedShipmentID:    message.MatchedShipmentID,
		MatchedCustomerID:    message.MatchedCustomerID,
		MatchedCarrierID:     message.MatchedCarrierID,
		MatchReason:          message.MatchReason,
		FailureText:          message.FailureText,
		ReviewNote:           message.ReviewNote,
		NovaRead:             message.NovaRead,
	}
}
