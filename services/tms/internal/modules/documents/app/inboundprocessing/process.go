package inboundprocessing

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
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

// ErrClassificationUnavailable is a message left unsettled because the model
// could not be asked and the caller would rather ask again.
var ErrClassificationUnavailable = errors.New("the message could not be classified just now")

// ProcessOption changes how ProcessMessage treats a model it could not ask.
type ProcessOption func(*processOptions)

type processOptions struct {
	retryModel func(error) bool
}

// RetryModelFailures leaves a message unsettled, and returns
// ErrClassificationUnavailable, when the model could not be asked and retry
// says the failure is worth another attempt. Without it, or when retry says
// no, the message goes to review as Other, as it always has.
func RetryModelFailures(retry func(error) bool) ProcessOption {
	return func(o *processOptions) { o.retryModel = retry }
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
	opts ...ProcessOption,
) (*ProcessResult, error) {
	var options processOptions
	for _, opt := range opts {
		opt(&options)
	}

	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
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

	classification, modelErr := s.classify(ctx, message, tenantInfo)
	if modelErr != nil && options.retryModel != nil && options.retryModel(modelErr) {
		return nil, fmt.Errorf("%w: %w", ErrClassificationUnavailable, modelErr)
	}
	match := s.Match(ctx, message, tenantInfo)
	outcome := Settle(message.Mailbox, classification)

	message.Classification = classification.Class
	message.Confidence = classification.Confidence
	message.Facts = classification.Facts
	message.NovaRead = classification.Narrated
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

	targets, err := s.queue.Targets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	ApplyDue(message, targets)
	applied := s.applyRules(ctx, message)
	ApplyDue(message, targets)

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	s.l.Info("settled an inbound message",
		zap.String("messageId", updated.ID.String()),
		zap.String("classification", string(updated.Classification)),
		zap.String("status", string(updated.Status)),
		zap.Bool("handled", outcome.Handle))

	s.recordReading(ctx, updated, applied)
	s.notify.MessagesChanged(ctx, updated)
	s.notifyNeedsReview(ctx, updated)
	s.queueForRetrieval(ctx, updated)
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
	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         messageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if message.Status.Terminal() {
		return nil
	}

	targets, err := s.queue.Targets(ctx, tenantInfo)
	if err != nil {
		return err
	}

	message.Status = inboundmessage.StatusInReview
	message.FailureCode = code
	message.FailureText = reason
	message.ReviewNote = "This message could not be processed, so it is waiting on a person."
	message.UpdatedAt = timeutils.NowUnix()
	ApplyDue(message, targets)

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return err
	}

	s.notify.MessagesChanged(ctx, updated)
	s.notifyNeedsReview(ctx, updated)
	s.queueForRetrieval(ctx, updated)

	return nil
}

func ApplyDue(message *inboundmessage.InboundMessage, targets triage.Targets) {
	if message.NeedsReview() {
		if message.DueAt == 0 {
			message.ApplyResponseTarget(targets.For(message.Classification))
		}

		return
	}

	message.DueAt = 0
	message.Unpark()
}

func (s *Service) applyRules(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) *inboundmessageports.RuleApplication {
	if s.ruleEvaluator == nil || !message.IsWaiting() {
		return nil
	}

	applied, err := s.ruleEvaluator.ApplyRules(ctx, message)
	if err != nil {
		s.l.Warn("could not apply the organization's rules; the message waits on a person",
			zap.String("messageId", message.ID.String()), zap.Error(err))

		return nil
	}

	return applied
}

func (s *Service) recordReading(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	applied *inboundmessageports.RuleApplication,
) {
	activities := make([]*triage.Activity, 0, 2)
	if message.NovaRead {
		read := triage.NewActivity(message, triage.ActivityNovaRead).
			ByNova().
			About(string(message.Classification), "")
		read.ToClassification = message.Classification
		activities = append(activities, read)
	}
	if applied != nil && applied.Rule != nil {
		rule := triage.NewActivity(message, triage.ActivityRuleApplied).
			ByNova().
			About(string(applied.Rule.Action), message.SenderDomain)
		rule.RecordID = applied.Rule.ID
		activities = append(activities, rule)
	}
	if len(activities) == 0 {
		return
	}

	if err := s.activityRepo.Insert(ctx, activities); err != nil {
		s.l.Warn("could not record what Nova did with a message",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
}
