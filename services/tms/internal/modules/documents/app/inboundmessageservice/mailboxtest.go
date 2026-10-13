package inboundmessageservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const (
	testTokenPrefix = "trv-test-"
	testTokenBytes  = 8
	testSubject     = "Trenova test message "
	testBody        = "This is a test from Trenova to check that this address receives mail. " +
		"It lands in Triage like any other message; you can ignore it once it arrives."
)

func (s *Service) SendMailboxTest(
	ctx context.Context,
	req MailboxActionRequest,
) (*inboundmessage.Mailbox, error) {
	if s.email == nil {
		return nil, errortypes.NewBusinessError("Outbound email is not set up for this organization")
	}

	tenantInfo := req.Actor.TenantInfo()
	mailbox, err := s.getMailbox(ctx, req.ID, tenantInfo)
	if err != nil {
		return nil, err
	}
	if mailbox.Status != inboundmessage.MailboxActive {
		return nil, errortypes.NewBusinessError("Turn this address on before testing it")
	}

	suffix, err := tokenutils.RandomHex(testTokenBytes)
	if err != nil {
		return nil, err
	}
	token := testTokenPrefix + suffix

	updated, err := s.mailboxState.MarkTestSent(ctx, inboundmessageports.MailboxTestRequest{
		TenantInfo: tenantInfo,
		MailboxID:  mailbox.ID,
		Token:      token,
		At:         s.now(),
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.email.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeOperations,
		To:             []string{mailbox.Address},
		Subject:        testSubject + token,
		Text:           testBody,
		HTML:           "<p>" + testBody + "</p>",
		IdempotencyKey: token,
	}); err != nil {
		if _, restoreErr := s.mailboxState.MarkTestSent(ctx, inboundmessageports.MailboxTestRequest{
			TenantInfo: tenantInfo,
			MailboxID:  mailbox.ID,
			Token:      mailbox.TestToken,
			At:         mailbox.TestSentAt,
		}); restoreErr != nil {
			s.l.Warn("could not restore a mailbox's last test after a failed send",
				zap.String("mailboxId", mailbox.ID.String()), zap.Error(restoreErr))
		}
		if errors.Is(err, emailports.ErrNoEmailProfile) {
			return nil, errortypes.NewBusinessError(
				"No email profile is set up for this organization, so a test cannot be sent. " +
					"An administrator can add one in the email settings, or send any email to this address yourself.",
			).WithInternal(err)
		}

		return nil, err
	}

	return updated, nil
}

func carriesTestToken(mailbox *inboundmessage.Mailbox, message *inboundmessage.InboundMessage) bool {
	if mailbox.TestToken == "" {
		return false
	}
	token := strings.ToLower(mailbox.TestToken)

	return strings.Contains(strings.ToLower(message.Subject), token) ||
		strings.Contains(strings.ToLower(message.TextBody), token)
}

func (s *Service) noteTestReceived(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	message *inboundmessage.InboundMessage,
) *triage.Activity {
	if !carriesTestToken(mailbox, message) {
		return nil
	}

	matched, err := s.mailboxState.MarkTestReceived(ctx, inboundmessageports.MailboxTestRequest{
		TenantInfo: tenantOfMessage(message),
		MailboxID:  mailbox.ID,
		Token:      mailbox.TestToken,
		At:         message.ReceivedAt,
	})
	if err != nil {
		s.l.Warn("could not record that a mailbox test arrived",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return nil
	}
	if !matched {
		return nil
	}

	return triage.NewActivity(message, triage.ActivityTestReceived).About(mailbox.Address, "")
}
