package triageworkservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	maxExceptionQuantity = 100000
	maxExceptionUnit     = 50
	maxExceptionNote     = 500
	maxBlockedSenders    = 500
	blockedReviewNote    = "Blocked the sender"
)

var exceptionWords = map[inboundmessage.ExceptionKind]string{
	inboundmessage.ExceptionShortage: "short",
	inboundmessage.ExceptionOverage:  "over",
	inboundmessage.ExceptionDamage:   "damaged",
}

type LogExceptionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageID  pulid.ID
	Kind       inboundmessage.ExceptionKind
	Quantity   int
	Unit       string
	Note       string
}

type ExceptionPlan struct {
	Message *inboundmessage.InboundMessage
	After   *inboundmessage.InboundMessage
	Hold    *inboundmessageports.LogTriageExceptionRequest
	Summary string
}

func (s *Service) LogException(
	ctx context.Context,
	req *LogExceptionRequest,
) (*inboundmessage.InboundMessage, error) {
	plan, err := s.PlanLogException(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = s.exceptions.LogException(ctx, plan.Hold); err != nil {
		return nil, err
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			ApplyException(m, req.Kind, now)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	activity := personActivity(updated, triage.ActivityExceptionLogged, req.UserID).
		About(plan.Summary, strings.TrimSpace(req.Note))
	activity.RecordID = plan.Message.MatchedShipmentID
	s.record(ctx, activity)
	s.changed(ctx, updated)

	return updated, nil
}

func (s *Service) PlanLogException(
	ctx context.Context,
	req *LogExceptionRequest,
) (*ExceptionPlan, error) {
	if s.exceptions == nil {
		return nil, errortypes.NewBusinessError("Exceptions cannot be logged on this installation")
	}
	if err := validateException(req); err != nil {
		return nil, err
	}

	message, err := s.message(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}
	if message.MatchedShipmentID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"Link the message to its shipment before logging an exception",
		)
	}

	actorTenant := req.TenantInfo
	actorTenant.UserID = req.UserID
	after := *message
	ApplyException(&after, req.Kind, s.now())

	return &ExceptionPlan{
		Message: message,
		After:   &after,
		Hold: &inboundmessageports.LogTriageExceptionRequest{
			TenantInfo: req.TenantInfo,
			Actor:      permissionports.UserActor(actorTenant),
			ShipmentID: message.MatchedShipmentID,
			Kind:       req.Kind,
			Quantity:   req.Quantity,
			Unit:       strings.TrimSpace(req.Unit),
			Note:       strings.TrimSpace(req.Note),
			Source: fmt.Sprintf(
				"From an email from %s: %s",
				message.FromAddress,
				message.Subject,
			),
		},
		Summary: exceptionSummary(req),
	}, nil
}

func ApplyException(
	message *inboundmessage.InboundMessage,
	kind inboundmessage.ExceptionKind,
	now int64,
) {
	message.ActionState.ExceptionKind = kind
	message.ActionState.ExceptionLoggedAt = now
}

func validateException(req *LogExceptionRequest) error {
	multiErr := errortypes.NewMultiError()
	if !req.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Choose shortage, overage or damage")
	}
	if req.Quantity < 1 || req.Quantity > maxExceptionQuantity {
		multiErr.Add("quantity", errortypes.ErrInvalid, "Say how many, at least one")
	}
	unit := strings.TrimSpace(req.Unit)
	switch {
	case unit == "":
		multiErr.Add("unit", errortypes.ErrRequired, "Say what was counted: pallets, cases, pieces")
	case len([]rune(unit)) > maxExceptionUnit:
		multiErr.Add(
			"unit",
			errortypes.ErrInvalid,
			"Keep the unit under {0} characters",
			maxExceptionUnit,
		)
	}
	if len([]rune(req.Note)) > maxExceptionNote {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"Keep the note under {0} characters",
			maxExceptionNote,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func exceptionSummary(req *LogExceptionRequest) string {
	return fmt.Sprintf(
		"%d %s %s",
		req.Quantity,
		strings.TrimSpace(req.Unit),
		exceptionWords[req.Kind],
	)
}

type BlockSenderRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageID  pulid.ID
}

type SenderPlan struct {
	Message *inboundmessage.InboundMessage
	After   *inboundmessage.InboundMessage
	Mailbox *inboundmessage.Mailbox
	Sender  string
	Blocked []string
	Changed bool
}

func SenderOf(message *inboundmessage.InboundMessage) (string, error) {
	sender, err := emailutils.Normalize(message.FromAddress)
	if err != nil {
		sender = strings.ToLower(strings.TrimSpace(message.FromAddress))
	}
	if sender == "" {
		return "", errortypes.NewBusinessError("This message has no sender to block")
	}

	return sender, nil
}

func (s *Service) BlockSender(
	ctx context.Context,
	req *BlockSenderRequest,
) (*inboundmessage.InboundMessage, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}

	sender, err := SenderOf(message)
	if err != nil {
		return nil, err
	}

	if _, err = s.changeBlockedSenders(
		ctx,
		req.TenantInfo,
		message.MailboxID,
		blocking(sender),
	); err != nil {
		return nil, err
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			ApplyBlockSender(m, req.UserID, now)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	s.record(ctx,
		personActivity(updated, triage.ActivitySenderBlocked, req.UserID).About(sender, ""),
		personActivity(updated, triage.ActivityIgnored, req.UserID).About(blockedReviewNote, ""),
	)
	s.changed(ctx, updated)

	return updated, nil
}

func (s *Service) PlanBlockSender(
	ctx context.Context,
	req *BlockSenderRequest,
) (*SenderPlan, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}

	sender, err := SenderOf(message)
	if err != nil {
		return nil, err
	}

	plan, err := s.planBlockedSenders(ctx, req.TenantInfo, message.MailboxID, blocking(sender))
	if err != nil {
		return nil, err
	}

	after := *message
	ApplyBlockSender(&after, req.UserID, s.now())
	plan.Message = message
	plan.After = &after
	plan.Sender = sender

	return plan, nil
}

func ApplyBlockSender(message *inboundmessage.InboundMessage, userID pulid.ID, now int64) {
	message.ActionState.SenderBlockedAt = now
	if message.Status == inboundmessage.StatusIgnored ||
		message.Status == inboundmessage.StatusActioned {
		return
	}
	message.Status = inboundmessage.StatusIgnored
	message.ReviewedBy = userID
	message.ReviewedAt = now
	message.ReviewNote = blockedReviewNote
	message.DueAt = 0
	message.Unpark()
}

type UnblockSenderRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MailboxID  pulid.ID
	Sender     string
}

func (s *Service) UnblockSender(
	ctx context.Context,
	req *UnblockSenderRequest,
) (*inboundmessage.Mailbox, error) {
	sender, err := unblockedSender(req)
	if err != nil {
		return nil, err
	}

	return s.changeBlockedSenders(ctx, req.TenantInfo, req.MailboxID, unblocking(sender))
}

func (s *Service) PlanUnblockSender(
	ctx context.Context,
	req *UnblockSenderRequest,
) (*SenderPlan, error) {
	sender, err := unblockedSender(req)
	if err != nil {
		return nil, err
	}

	plan, err := s.planBlockedSenders(ctx, req.TenantInfo, req.MailboxID, unblocking(sender))
	if err != nil {
		return nil, err
	}
	plan.Sender = sender

	return plan, nil
}

func unblockedSender(req *UnblockSenderRequest) (string, error) {
	sender := strings.ToLower(strings.TrimSpace(req.Sender))
	if sender == "" {
		return "", errortypes.NewValidationError(
			"sender",
			errortypes.ErrRequired,
			"Say which sender to unblock",
		)
	}

	return sender, nil
}

func blocking(sender string) func([]string) []string {
	return func(senders []string) []string {
		if slices.Contains(senders, sender) {
			return senders
		}

		return append(senders, sender)
	}
}

func unblocking(sender string) func([]string) []string {
	return func(senders []string) []string {
		return slices.DeleteFunc(
			senders,
			func(blocked string) bool { return blocked == sender },
		)
	}
}

func nextBlockedSenders(current []string, change func([]string) []string) ([]string, bool, error) {
	next := change(slices.Clone(current))
	if slices.Equal(next, current) {
		return current, false, nil
	}
	if len(next) > maxBlockedSenders {
		return nil, false, errortypes.NewBusinessError(
			"A source blocks at most {0} senders. Unblock some first.", maxBlockedSenders,
		)
	}

	return next, true, nil
}

func (s *Service) planBlockedSenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	mailboxID pulid.ID,
	change func([]string) []string,
) (*SenderPlan, error) {
	mailbox, err := s.mailboxes.GetByID(ctx, inboundmessageports.GetMailboxByIDRequest{
		ID:         mailboxID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	next, changed, err := nextBlockedSenders(mailbox.BlockedSenders, change)
	if err != nil {
		return nil, err
	}

	return &SenderPlan{Mailbox: mailbox, Blocked: next, Changed: changed}, nil
}

func (s *Service) changeBlockedSenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	mailboxID pulid.ID,
	change func([]string) []string,
) (*inboundmessage.Mailbox, error) {
	var lastErr error
	for range maxMessageWriteAttempts {
		mailbox, err := s.mailboxes.GetByID(ctx, inboundmessageports.GetMailboxByIDRequest{
			ID:         mailboxID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return nil, err
		}

		next, changed, err := nextBlockedSenders(mailbox.BlockedSenders, change)
		if err != nil {
			return nil, err
		}
		if !changed {
			return mailbox, nil
		}
		mailbox.BlockedSenders = next

		updated, err := s.mailboxes.Update(ctx, mailbox)
		if err == nil {
			return updated, nil
		}
		if !errortypes.IsVersionMismatchError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

type DecideStopsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Actor      *permissionports.RequestActor
	MessageID  pulid.ID
	Decision   inboundmessage.StopsDecision
}

type StopsPlan struct {
	Message   *inboundmessage.InboundMessage
	After     *inboundmessage.InboundMessage
	Unchanged bool
}

func (s *Service) DecideStops(
	ctx context.Context,
	req *DecideStopsRequest,
) (*inboundmessage.InboundMessage, error) {
	plan, err := s.PlanDecideStops(ctx, req)
	if err != nil {
		return nil, err
	}
	message := plan.Message
	if plan.Unchanged {
		return s.refreshed(ctx, req.TenantInfo, message.ID)
	}

	settled := &inboundmessageports.SettledTriageDetention{}
	if s.detention != nil {
		settled, err = s.detention.SettleDetention(
			ctx,
			&inboundmessageports.SettleTriageDetentionRequest{
				TenantInfo: req.TenantInfo,
				Actor:      req.Actor,
				ShipmentID: message.MatchedShipmentID,
				Decision:   req.Decision,
				Note: fmt.Sprintf(
					"Decided in Triage after checking stop times (%s)",
					message.Subject,
				),
				UserID: req.UserID,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			ApplyStopsDecision(m, req.Decision, now, settled)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	activity := personActivity(updated, triage.ActivityStopsDecided, req.UserID).
		About(string(req.Decision), stopsDetail(settled))
	activity.RecordID = message.MatchedShipmentID
	s.record(ctx, activity)

	if _, err = s.replies.DraftReplyFor(ctx, updated, triagenova.Interactive); err != nil {
		s.l.Warn("recorded a stops decision but Nova could not draft the reply",
			zap.String("messageId", updated.ID.String()), zap.Error(err))
	}

	return s.refreshed(ctx, req.TenantInfo, updated.ID)
}

func (s *Service) PlanDecideStops(
	ctx context.Context,
	req *DecideStopsRequest,
) (*StopsPlan, error) {
	if !req.Decision.IsValid() {
		return nil, errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalid,
			"Choose waive or stand by",
		)
	}

	message, err := s.message(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}
	if message.Classification != inboundmessage.ClassificationDetentionDispute {
		return nil, errortypes.NewBusinessError(
			"Stop times are checked only for detention disputes",
		)
	}
	if message.MatchedShipmentID.IsNil() {
		return nil, errortypes.NewBusinessError("Link the message to its shipment first")
	}
	pending := message.ActionState.CreditPending()
	if message.ActionState.StopsDecision == req.Decision && !pending {
		return &StopsPlan{Message: message, After: message, Unchanged: true}, nil
	}
	if pending && req.Decision == inboundmessage.StopsDecisionStandBy {
		return nil, errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalidOperation,
			"The credit for this detention on {0} is waiting for approval in billing. Reject it there before standing by the charge.",
			strings.Join(message.ActionState.DetentionWaiver.InvoiceNumbers, ", "),
		)
	}

	after := *message
	ApplyStopsDecision(&after, req.Decision, s.now(), nil)

	return &StopsPlan{Message: message, After: &after}, nil
}

func ApplyStopsDecision(
	message *inboundmessage.InboundMessage,
	decision inboundmessage.StopsDecision,
	now int64,
	settled *inboundmessageports.SettledTriageDetention,
) {
	message.ActionState.StopsDecision = decision
	message.ActionState.StopsDecidedAt = now
	message.ActionState.DetentionWaiver = waiverOf(decision, settled)
}

func waiverOf(
	decision inboundmessage.StopsDecision,
	settled *inboundmessageports.SettledTriageDetention,
) *inboundmessage.DetentionWaiver {
	if decision != inboundmessage.StopsDecisionWaive || settled == nil || !settled.Settled {
		return nil
	}

	return &inboundmessage.DetentionWaiver{
		Amount:            settled.Amount,
		Currency:          settled.Currency,
		InvoiceNumbers:    settled.InvoiceNumbers,
		CreditMemoNumbers: settled.CreditMemoNumbers,
		CreditPending:     settled.CreditPending,
	}
}

func stopsDetail(settled *inboundmessageports.SettledTriageDetention) string {
	if settled == nil || !settled.Settled || settled.Amount == "" {
		return ""
	}

	amount := strings.TrimSpace(settled.Amount + " " + settled.Currency)
	if !settled.Credited() {
		return amount
	}

	invoices := strings.Join(settled.InvoiceNumbers, ", ")
	if settled.CreditPending {
		return amount + ", credit on " + invoices + " waits for approval"
	}
	if len(settled.CreditMemoNumbers) > 0 {
		return amount + ", credited on " + invoices + " by " +
			strings.Join(settled.CreditMemoNumbers, ", ")
	}

	return amount + ", credited on " + invoices
}
