package inboundmessageservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// ProcessResult is what became of a message, for the activity's log and for a
// test to assert on without reading the row back.
type ProcessResult struct {
	MessageID      pulid.ID
	Status         inboundmessage.Status
	Classification inboundmessage.Classification
	Handled        bool
	Matched        bool
}

// ProcessMessage reads a staged message and settles it.
//
// It is idempotent by status: a message that has already been dealt with is
// left alone, so a workflow retried after a crash cannot reclassify a message a
// person has since reviewed and reverse their decision.
func (s *Service) ProcessMessage(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*ProcessResult, error) {
	message, err := s.messageRepo.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
	if err != nil {
		return nil, err
	}

	if message.Status != inboundmessage.StatusReceived &&
		message.Status != inboundmessage.StatusProcessing {
		return &ProcessResult{
			MessageID:      message.ID,
			Status:         message.Status,
			Classification: message.Classification,
		}, nil
	}

	if message.Mailbox == nil {
		return nil, fmt.Errorf("message %s has no mailbox to read its policy from", message.ID)
	}

	classification := s.Classify(ctx, message, tenantInfo)
	match := s.Match(ctx, message, tenantInfo)
	outcome := Settle(message.Mailbox, classification)

	message.Classification = classification.Class
	message.Confidence = classification.Confidence
	message.Status = outcome.Status
	message.MatchedCustomerID = match.CustomerID
	message.MatchedCarrierID = match.CarrierID
	message.MatchedShipmentID = match.ShipmentID
	message.MatchReason = match.Reason
	message.ReviewNote = outcome.Note

	if outcome.Status == inboundmessage.StatusInReview {
		// The clock starts when a person is asked, not when the mail arrived,
		// so an inbox can show how long something has actually been waiting on
		// somebody rather than how long ago it was sent.
		message.ReviewedAt = 0
	}

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	s.l.Info("settled an inbound message",
		zap.String("messageId", updated.ID.String()),
		zap.String("classification", string(updated.Classification)),
		zap.String("status", string(updated.Status)),
		zap.Bool("handled", outcome.Handle))

	s.project(ctx, updated)
	s.publishMessage(ctx, updated, inboxRealtimeAction)
	s.notifyNeedsReview(ctx, updated)
	s.announce(ctx, updated)

	return &ProcessResult{
		MessageID:      updated.ID,
		Status:         updated.Status,
		Classification: updated.Classification,
		Handled:        outcome.Handle,
		Matched:        match.Reason != "",
	}, nil
}

// MarkFailed records that the pipeline could not finish.
//
// A message stuck mid-pipeline is worse than one that failed visibly: the
// first looks like it is still being worked on, and nobody goes to look.
func (s *Service) MarkFailed(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
	code string,
	reason string,
) error {
	message, err := s.messageRepo.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:         messageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if message.Status.Terminal() {
		return nil
	}

	message.Status = inboundmessage.StatusInReview
	message.FailureCode = code
	message.FailureText = reason
	message.ReviewNote = "This message could not be processed, so it is waiting on a person."
	message.UpdatedAt = timeutils.NowUnix()

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return err
	}

	s.project(ctx, updated)
	s.publishMessage(ctx, updated, inboxRealtimeAction)
	s.notifyNeedsReview(ctx, updated)

	return nil
}
