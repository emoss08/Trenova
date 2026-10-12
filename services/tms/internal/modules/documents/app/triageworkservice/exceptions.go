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

func (s *Service) LogException(
	ctx context.Context,
	req *LogExceptionRequest,
) (*inboundmessage.InboundMessage, error) {
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

	summary := exceptionSummary(req)
	actorTenant := req.TenantInfo
	actorTenant.UserID = req.UserID
	if err = s.exceptions.LogException(ctx, &inboundmessageports.LogTriageExceptionRequest{
		TenantInfo: req.TenantInfo,
		Actor:      permissionports.UserActor(actorTenant),
		ShipmentID: message.MatchedShipmentID,
		Kind:       req.Kind,
		Quantity:   req.Quantity,
		Unit:       strings.TrimSpace(req.Unit),
		Note:       strings.TrimSpace(req.Note),
		Source:     fmt.Sprintf("From an email from %s: %s", message.FromAddress, message.Subject),
	}); err != nil {
		return nil, err
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			m.ActionState.ExceptionKind = req.Kind
			m.ActionState.ExceptionLoggedAt = now

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	activity := personActivity(updated, triage.ActivityExceptionLogged, req.UserID).
		About(summary, strings.TrimSpace(req.Note))
	activity.RecordID = message.MatchedShipmentID
	s.record(ctx, activity)
	s.changed(ctx, updated)

	return updated, nil
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

func (s *Service) BlockSender(
	ctx context.Context,
	req *BlockSenderRequest,
) (*inboundmessage.InboundMessage, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}

	sender, err := emailutils.Normalize(message.FromAddress)
	if err != nil {
		sender = strings.ToLower(strings.TrimSpace(message.FromAddress))
	}
	if sender == "" {
		return nil, errortypes.NewBusinessError("This message has no sender to block")
	}

	if _, err = s.changeBlockedSenders(
		ctx,
		req.TenantInfo,
		message.MailboxID,
		func(senders []string) []string {
			if slices.Contains(senders, sender) {
				return senders
			}

			return append(senders, sender)
		},
	); err != nil {
		return nil, err
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			m.ActionState.SenderBlockedAt = now
			if m.Status != inboundmessage.StatusIgnored &&
				m.Status != inboundmessage.StatusActioned {
				m.Status = inboundmessage.StatusIgnored
				m.ReviewedBy = req.UserID
				m.ReviewedAt = now
				m.ReviewNote = blockedReviewNote
				m.DueAt = 0
				m.Unpark()
			}

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
	sender := strings.ToLower(strings.TrimSpace(req.Sender))
	if sender == "" {
		return nil, errortypes.NewValidationError(
			"sender",
			errortypes.ErrRequired,
			"Say which sender to unblock",
		)
	}

	return s.changeBlockedSenders(
		ctx,
		req.TenantInfo,
		req.MailboxID,
		func(senders []string) []string {
			return slices.DeleteFunc(
				senders,
				func(blocked string) bool { return blocked == sender },
			)
		},
	)
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

		next := change(slices.Clone(mailbox.BlockedSenders))
		if slices.Equal(next, mailbox.BlockedSenders) {
			return mailbox, nil
		}
		if len(next) > maxBlockedSenders {
			return nil, errortypes.NewBusinessError(
				"A source blocks at most {0} senders. Unblock some first.", maxBlockedSenders,
			)
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
	MessageID  pulid.ID
	Decision   inboundmessage.StopsDecision
}

func (s *Service) DecideStops(
	ctx context.Context,
	req *DecideStopsRequest,
) (*inboundmessage.InboundMessage, error) {
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

	detail := ""
	if s.detention != nil {
		settled, settleErr := s.detention.SettleDetention(
			ctx,
			&inboundmessageports.SettleTriageDetentionRequest{
				TenantInfo: req.TenantInfo,
				ShipmentID: message.MatchedShipmentID,
				Decision:   req.Decision,
				Note: fmt.Sprintf(
					"Decided in Triage after checking stop times (%s)",
					message.Subject,
				),
				UserID: req.UserID,
			},
		)
		if settleErr != nil {
			return nil, settleErr
		}
		if settled != nil && settled.Settled && settled.Amount != "" {
			detail = strings.TrimSpace(settled.Amount + " " + settled.Currency)
		}
	}

	now := s.now()
	updated, err := s.mutate(
		ctx,
		req.TenantInfo,
		req.MessageID,
		func(m *inboundmessage.InboundMessage) error {
			m.ActionState.StopsDecision = req.Decision
			m.ActionState.StopsDecidedAt = now

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	activity := personActivity(updated, triage.ActivityStopsDecided, req.UserID).
		About(string(req.Decision), detail)
	activity.RecordID = message.MatchedShipmentID
	s.record(ctx, activity)

	if _, err = s.replies.DraftReplyFor(ctx, updated, triagenova.Interactive); err != nil {
		s.l.Warn("recorded a stops decision but Nova could not draft the reply",
			zap.String("messageId", updated.ID.String()), zap.Error(err))
	}

	return s.refreshed(ctx, req.TenantInfo, updated.ID)
}
