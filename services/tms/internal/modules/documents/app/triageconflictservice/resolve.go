package triageconflictservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type ResolveRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
	ConflictID string
	Resolution inboundmessage.ConflictResolution
}

type ReopenRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
	ConflictID string
}

func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (*inboundmessage.InboundMessage, error) {
	if !req.Resolution.FitsKind(inboundmessage.ConflictDuplicateDocument) &&
		!req.Resolution.FitsKind(inboundmessage.ConflictOverlappingLoad) {
		return nil, errortypes.NewValidationError(
			"resolution", errortypes.ErrInvalid, "That is not a way to settle a conflict",
		)
	}

	return s.change(ctx, changeRequest{
		tenantInfo: req.TenantInfo,
		messageID:  req.MessageID,
		conflictID: req.ConflictID,
		apply: func(message *inboundmessage.InboundMessage, conflict *inboundmessage.Conflict) error {
			if !req.Resolution.FitsKind(conflict.Kind) {
				return errortypes.NewValidationError(
					"resolution", errortypes.ErrInvalid, "That does not settle this kind of conflict",
				)
			}
			if conflict.Kind == inboundmessage.ConflictDuplicateDocument &&
				attachmentFiled(message, conflict.AttachmentID) {
				return errortypes.NewBusinessError(
					"The attachment is already filed. Unfile it before settling the conflict.",
				)
			}
			conflict.Resolution = req.Resolution
			conflict.ResolvedAt = timeutils.NowUnix()
			conflict.ResolvedBy = req.ActorID

			return nil
		},
		activity: func(message *inboundmessage.InboundMessage, conflict *inboundmessage.Conflict) *triage.Activity {
			activity := triage.NewActivity(message, triage.ActivityConflictResolved).
				ByPerson(req.ActorID).
				About(string(conflict.Kind), string(req.Resolution))
			activity.RecordID = conflict.ExistingShipmentID

			return activity
		},
	})
}

func (s *Service) Reopen(
	ctx context.Context,
	req ReopenRequest,
) (*inboundmessage.InboundMessage, error) {
	return s.change(ctx, changeRequest{
		tenantInfo: req.TenantInfo,
		messageID:  req.MessageID,
		conflictID: req.ConflictID,
		apply: func(message *inboundmessage.InboundMessage, conflict *inboundmessage.Conflict) error {
			if conflict.Resolution == "" {
				return errortypes.NewBusinessError("The conflict is still open")
			}
			if conflict.Kind == inboundmessage.ConflictDuplicateDocument &&
				attachmentFiled(message, conflict.AttachmentID) {
				return errortypes.NewBusinessError(
					"The attachment is already filed as settled. Unfile it to reopen the conflict.",
				)
			}
			conflict.Resolution = ""
			conflict.ResolvedAt = 0
			conflict.ResolvedBy = pulid.Nil

			return nil
		},
		activity: func(message *inboundmessage.InboundMessage, conflict *inboundmessage.Conflict) *triage.Activity {
			activity := triage.NewActivity(message, triage.ActivityConflictReopened).
				ByPerson(req.ActorID).
				About(string(conflict.Kind), "")
			activity.RecordID = conflict.ExistingShipmentID

			return activity
		},
	})
}

type changeRequest struct {
	tenantInfo pagination.TenantInfo
	messageID  pulid.ID
	conflictID string
	apply      func(*inboundmessage.InboundMessage, *inboundmessage.Conflict) error
	activity   func(*inboundmessage.InboundMessage, *inboundmessage.Conflict) *triage.Activity
}

func (s *Service) change(
	ctx context.Context,
	req changeRequest,
) (*inboundmessage.InboundMessage, error) {
	for attempt := range saveAttempts {
		message, err := s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
			ID:                 req.messageID,
			TenantInfo:         req.tenantInfo,
			IncludeAttachments: true,
		})
		if err != nil {
			return nil, err
		}

		conflicts := append([]inboundmessage.Conflict(nil), message.Conflicts...)
		index := conflictIndex(conflicts, req.conflictID)
		if index < 0 {
			return nil, errortypes.NewNotFoundError("That conflict is not on this message")
		}
		if err = req.apply(message, &conflicts[index]); err != nil {
			return nil, err
		}

		err = s.store.SaveTriageFields(ctx, triageports.SaveTriageFieldsRequest{
			TenantInfo: req.tenantInfo,
			MessageID:  message.ID,
			Version:    message.Version,
			Conflicts:  &conflicts,
		})
		if errors.Is(err, triageports.ErrMessageChanged) && attempt < saveAttempts-1 {
			continue
		}
		if err != nil {
			return nil, err
		}

		message.Version++
		message.Conflicts = conflicts
		if recordErr := s.activities.RecordActivity(
			ctx, req.activity(message, &conflicts[index]),
		); recordErr != nil {
			s.l.Warn("could not record a conflict decision",
				zap.String("messageId", message.ID.String()), zap.Error(recordErr))
		}
		s.notifier.MessagesChanged(ctx, message)

		return message, nil
	}

	return nil, errortypes.NewConflictError(
		"The message kept changing while the conflict was being settled. Try again.",
	)
}

func conflictIndex(conflicts []inboundmessage.Conflict, id string) int {
	for i := range conflicts {
		if conflicts[i].ID == id {
			return i
		}
	}

	return -1
}

func attachmentFiled(message *inboundmessage.InboundMessage, attachmentID pulid.ID) bool {
	for _, attachment := range message.Attachments {
		if attachment.ID == attachmentID {
			return attachment.Filed()
		}
	}

	return false
}
