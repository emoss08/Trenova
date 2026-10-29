package inboundqueue

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// maxReviewNoteLength bounds what a person may write on a message. It is the
// column's length, checked here so a long note is a field error rather than a
// database error the reader cannot act on.
const maxReviewNoteLength = 2000

// ReviewRequest is a person's decision about a message.
type ReviewRequest struct {
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
	ReviewerID pulid.ID
	Status     inboundmessage.Status
	Note       string
}

// LinkRequest says what a message is about, by hand.
type LinkRequest struct {
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
	ReviewerID pulid.ID
	ShipmentID pulid.ID
	CustomerID pulid.ID
	CarrierID  pulid.ID
	Reason     string
}

// Review records what a person decided.
//
// Only Actioned and Ignored are a person's to choose. The rest are states the
// pipeline moves through, and letting a reader set one would put a message
// back into a queue nothing is going to pick it up from.
func (s *Service) Review(
	ctx context.Context,
	req *ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	if _, err := reviewNote(req); err != nil {
		return nil, err
	}

	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	before := queueStateOf(message)
	if err = ApplyReview(message, req, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	message.DueAt = 0

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, byPerson(
		triage.NewActivity(updated, settleActivityKind(updated.Status)).
			About("", updated.ReviewNote),
		req.ReviewerID,
	))
	s.afterQueueChange(ctx, req.TenantInfo, &TransitionResult{
		Before:  map[pulid.ID]*inboundmessageports.QueueState{updated.ID: before},
		Changed: []*inboundmessageports.QueueState{queueStateOf(updated)},
	})

	return updated, nil
}

// Link says what a message is about when the reading did not work it out.
//
// The reason is required rather than optional. A hand-made link with no reason
// is indistinguishable on the page from one the classifier guessed at, and the
// whole point of storing the reason is that somebody can check it.
func (s *Service) Link(
	ctx context.Context,
	req *LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := s.CheckLink(ctx, req); err != nil {
		return nil, err
	}

	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	ApplyLink(message, req)

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	linked := byPerson(
		triage.NewActivity(updated, triage.ActivityLinked).About(updated.MatchReason, ""),
		req.ReviewerID,
	)
	linked.RecordID = pulid.FirstNotNil(req.ShipmentID, req.CustomerID, req.CarrierID)
	s.recordActivity(ctx, linked)
	s.notify.MessagesChanged(ctx, updated)

	return updated, nil
}

// CheckLink says whether a link would be accepted, without making it. A desk
// asks before proposing, so a link to a record that is not there is refused
// while the call can still be fixed rather than after a person approved it.
func (s *Service) CheckLink(ctx context.Context, req *LinkRequest) error {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"Say why this is the right record")
	}
	if utf8.RuneCountInString(reason) > inboundmessage.MaxMatchReasonLength {
		return errortypes.NewValidationError("reason", errortypes.ErrInvalid,
			fmt.Sprintf("A reason may be at most %d characters",
				inboundmessage.MaxMatchReasonLength))
	}
	if req.ShipmentID.IsNil() && req.CustomerID.IsNil() && req.CarrierID.IsNil() {
		return errortypes.NewValidationError("shipmentId", errortypes.ErrRequired,
			"Name at least one record this message is about")
	}

	return s.verifyLinkTargets(ctx, req)
}

// verifyLinkTargets confirms each record a link names exists in the tenant.
//
// The columns carry no foreign key — a message can outlive the load it was
// about — so this is the only thing standing between a link and an id from
// another tenant, or one that was never real. A link that cannot be checked is
// refused rather than stored on trust.
func (s *Service) verifyLinkTargets(ctx context.Context, req *LinkRequest) error {
	checks := []struct {
		field  string
		id     pulid.ID
		label  string
		exists func(context.Context, pagination.TenantInfo, pulid.ID) (bool, error)
	}{
		{field: "shipmentId", id: req.ShipmentID, label: "shipment"},
		{field: "customerId", id: req.CustomerID, label: "customer"},
		{field: "carrierId", id: req.CarrierID, label: "carrier"},
	}
	if s.shipments != nil {
		checks[0].exists = s.shipments.ShipmentExists
	}
	if s.parties != nil {
		checks[1].exists = s.parties.CustomerExists
		checks[2].exists = s.parties.CarrierExists
	}

	multiErr := errortypes.NewMultiError()
	for _, check := range checks {
		if check.id.IsNil() {
			continue
		}
		if check.exists == nil {
			return errortypes.NewBusinessError(
				"Linked records cannot be verified on this installation",
			)
		}

		found, err := check.exists(ctx, req.TenantInfo, check.id)
		if err != nil {
			return err
		}
		if !found {
			multiErr.Add(check.field, errortypes.ErrInvalid,
				fmt.Sprintf("No %s with that id exists", check.label))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return s.messageRepo.GetByID(ctx, req)
}
