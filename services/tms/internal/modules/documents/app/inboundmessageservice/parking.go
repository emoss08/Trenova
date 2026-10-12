package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SnoozeRequest struct {
	TenantInfo     pagination.TenantInfo
	ActorID        pulid.ID
	Selection      Selection
	Until          int64
	Event          inboundmessage.SnoozeEvent
	DocumentTypeID pulid.ID
}

type MessageActionRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
}

type WaitForReplyRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
	Until      int64
}

type LinkedRecord string

const (
	LinkedShipment = LinkedRecord("Shipment")
	LinkedCustomer = LinkedRecord("Customer")
	LinkedCarrier  = LinkedRecord("Carrier")
)

type UnlinkRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
	Record     LinkedRecord
}

func ValidateSnooze(req *SnoozeRequest, now int64) error {
	switch {
	case req.Until == 0 && req.Event == "":
		return errortypes.NewValidationError("until", errortypes.ErrRequired,
			"Say when to bring it back, or what to wait for")
	case req.Until != 0 && req.Event != "":
		return errortypes.NewValidationError("event", errortypes.ErrInvalid,
			"Snooze until a time or until something happens, not both")
	case req.Until != 0 && req.Until <= now:
		return errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"A snooze has to end in the future")
	case req.Until != 0 && req.Until-now > inboundmessage.MaxSnoozeSeconds:
		return errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"A snooze can last at most 90 days")
	case req.Event != "" && !req.Event.IsValid():
		return errortypes.NewValidationError("event", errortypes.ErrInvalid,
			"That is not something a message can wait for")
	case req.Event == inboundmessage.SnoozeEventDocumentAttached && req.DocumentTypeID.IsNil():
		return errortypes.NewValidationError("documentTypeId", errortypes.ErrRequired,
			"Say which document to wait for")
	case req.Event != inboundmessage.SnoozeEventDocumentAttached && req.DocumentTypeID.IsNotNil():
		return errortypes.NewValidationError("documentTypeId", errortypes.ErrInvalid,
			"A document type is only named when waiting for a document")
	default:
		return nil
	}
}

func checkSnoozable(event inboundmessage.SnoozeEvent) func([]*inboundmessageports.QueueState) error {
	return func(states []*inboundmessageports.QueueState) error {
		for _, state := range states {
			if !state.NeedsReview() {
				return errortypes.NewBusinessError(
					"Only a message waiting on a person can be snoozed",
				)
			}
			if event != "" && state.MatchedShipmentID.IsNil() {
				return errortypes.NewValidationError("event", errortypes.ErrInvalid,
					"Link every message to a shipment before waiting on it")
			}
		}

		return nil
	}
}

func (s *Service) Snooze(
	ctx context.Context,
	req *SnoozeRequest,
) (*inboundmessageports.BulkResult, error) {
	if err := ValidateSnooze(req, s.now()); err != nil {
		return nil, err
	}

	documentTypeName := ""
	if req.DocumentTypeID.IsNotNil() {
		names, err := s.documentTypes.DocumentTypeNames(
			ctx, req.TenantInfo, []pulid.ID{req.DocumentTypeID},
		)
		if err != nil {
			return nil, err
		}
		name, ok := names[req.DocumentTypeID]
		if !ok {
			return nil, errortypes.NewValidationError("documentTypeId", errortypes.ErrInvalid,
				"No document type with that id exists")
		}
		documentTypeName = name
	}

	ids, err := s.resolveSelection(ctx, req.TenantInfo, req.Selection)
	if err != nil {
		return nil, err
	}

	result, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    ids,
		check:  checkSnoozable(req.Event),
		patch: inboundmessageports.QueuePatch{
			Park: &inboundmessageports.ParkPatch{
				Parking:        inboundmessage.ParkingSnoozed,
				SnoozedUntil:   req.Until,
				SnoozeEvent:    req.Event,
				WatchShipment:  req.Event != "",
				DocumentTypeID: req.DocumentTypeID,
				By:             req.ActorID,
			},
		},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			activity := activityOf(after, triage.ActivitySnoozed).
				About(string(req.Event), documentTypeName)
			activity.Until = req.Until
			activity.RecordID = after.SnoozeWatchID

			return byPerson(activity, req.ActorID)
		},
	})
	if err != nil {
		return nil, err
	}

	return s.bulkResult(ctx, req.TenantInfo, result, nil)
}

func (s *Service) Wake(
	ctx context.Context,
	req *MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	if _, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    []pulid.ID{req.MessageID},
		check: func(states []*inboundmessageports.QueueState) error {
			if err := requireFound(states, req.MessageID); err != nil {
				return err
			}
			if states[0].Parked != inboundmessage.ParkingSnoozed {
				return errortypes.NewBusinessError("This message is not snoozed")
			}

			return nil
		},
		patch: inboundmessageports.QueuePatch{Unpark: true},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			return byPerson(activityOf(after, triage.ActivityWoken), req.ActorID)
		},
	}); err != nil {
		return nil, err
	}

	return s.messageAfter(ctx, req.TenantInfo, req.MessageID)
}

func ValidateAwaitUntil(until, now int64) error {
	if until <= now {
		return errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"Say when to bring it back if nobody replies, in the future")
	}
	if until-now > inboundmessage.MaxAwaitSeconds {
		return errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"A message can wait on a reply for at most 30 days")
	}

	return nil
}

func (s *Service) WaitForReply(
	ctx context.Context,
	req *WaitForReplyRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := ValidateAwaitUntil(req.Until, s.now()); err != nil {
		return nil, err
	}

	if _, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    []pulid.ID{req.MessageID},
		check: func(states []*inboundmessageports.QueueState) error {
			if err := requireFound(states, req.MessageID); err != nil {
				return err
			}
			if !states[0].NeedsReview() {
				return errortypes.NewBusinessError(
					"Only a message waiting on a person can wait on a reply",
				)
			}

			return nil
		},
		patch: inboundmessageports.QueuePatch{
			Park: &inboundmessageports.ParkPatch{
				Parking:       inboundmessage.ParkingAwaitingReply,
				AwaitingUntil: req.Until,
			},
		},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			activity := activityOf(after, triage.ActivityWaitingOnReply).About(after.FromAddress, "")
			activity.Until = req.Until

			return byPerson(activity, req.ActorID)
		},
	}); err != nil {
		return nil, err
	}

	return s.messageAfter(ctx, req.TenantInfo, req.MessageID)
}

func (s *Service) StopWaiting(
	ctx context.Context,
	req *MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	if _, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    []pulid.ID{req.MessageID},
		check: func(states []*inboundmessageports.QueueState) error {
			if err := requireFound(states, req.MessageID); err != nil {
				return err
			}
			if states[0].Parked != inboundmessage.ParkingAwaitingReply {
				return errortypes.NewBusinessError("This message is not waiting on a reply")
			}

			return nil
		},
		patch: inboundmessageports.QueuePatch{Unpark: true},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			return byPerson(activityOf(after, triage.ActivityStoppedWaiting), req.ActorID)
		},
	}); err != nil {
		return nil, err
	}

	return s.messageAfter(ctx, req.TenantInfo, req.MessageID)
}

func linkedID(state *inboundmessageports.QueueState, record LinkedRecord) pulid.ID {
	switch record {
	case LinkedShipment:
		return state.MatchedShipmentID
	case LinkedCustomer:
		return state.MatchedCustomerID
	case LinkedCarrier:
		return state.MatchedCarrierID
	default:
		return pulid.Nil
	}
}

func UnlinkPatch(state *inboundmessageports.QueueState, record LinkedRecord) inboundmessageports.QueuePatch {
	patch := inboundmessageports.QueuePatch{
		ClearShipment: record == LinkedShipment,
		ClearCustomer: record == LinkedCustomer,
		ClearCarrier:  record == LinkedCarrier,
	}
	remaining := 0
	for _, kind := range []LinkedRecord{LinkedShipment, LinkedCustomer, LinkedCarrier} {
		if kind != record && linkedID(state, kind).IsNotNil() {
			remaining++
		}
	}
	patch.ClearReason = remaining == 0

	return patch
}

func (s *Service) Unlink(
	ctx context.Context,
	req *UnlinkRequest,
) (*inboundmessage.InboundMessage, error) {
	switch req.Record {
	case LinkedShipment, LinkedCustomer, LinkedCarrier:
	default:
		return nil, errortypes.NewValidationError("record", errortypes.ErrInvalid,
			"Say which record to unlink")
	}

	if _, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    []pulid.ID{req.MessageID},
		check: func(states []*inboundmessageports.QueueState) error {
			if err := requireFound(states, req.MessageID); err != nil {
				return err
			}
			if linkedID(states[0], req.Record).IsNil() {
				return errortypes.NewBusinessError("This message is not linked to a {0}",
					string(req.Record))
			}

			return nil
		},
		patchFor: func(states []*inboundmessageports.QueueState) inboundmessageports.QueuePatch {
			return UnlinkPatch(states[0], req.Record)
		},
		activity: func(before, after *inboundmessageports.QueueState) *triage.Activity {
			activity := activityOf(after, triage.ActivityUnlinked).About(string(req.Record), "")
			activity.RecordID = linkedID(before, req.Record)

			return byPerson(activity, req.ActorID)
		},
	}); err != nil {
		return nil, err
	}

	return s.messageAfter(ctx, req.TenantInfo, req.MessageID)
}
