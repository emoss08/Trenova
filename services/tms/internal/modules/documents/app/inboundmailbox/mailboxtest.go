package inboundmailbox

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/mailtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
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
	testCooldown    = int64(60)
)

func (s *Service) SendMailboxTest(
	ctx context.Context,
	req MailboxActionRequest,
) (*inboundmessage.Mailbox, error) {
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

	now := s.now()
	updated, err := s.mailboxState.ClaimTestSend(ctx, inboundmessageports.ClaimMailboxTestRequest{
		TenantInfo:     tenantInfo,
		MailboxID:      mailbox.ID,
		Token:          token,
		At:             now,
		LastSentBefore: now - testCooldown,
	})
	if errors.Is(err, inboundmessageports.ErrMailboxTestTooSoon) {
		return nil, testTooSoon(mailbox, now)
	}
	if err != nil {
		return nil, err
	}

	rendered, err := s.renderer.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindMailboxTestEmail,
		Data: mailtemplate.MailboxTestEmailContext{
			MailboxAddress: mailbox.Address,
			Token:          token,
		},
	})
	if err == nil {
		_, err = s.email.Send(ctx, &emailports.SendEmailRequest{
			TenantInfo:     tenantInfo,
			Type:           email.TypeMailboxTest,
			Purpose:        email.PurposeOperations,
			To:             []string{mailbox.Address},
			Subject:        rendered.Subject,
			Text:           rendered.Text,
			HTML:           rendered.HTML,
			IdempotencyKey: token,
		})
	}
	if err != nil {
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

func testTooSoon(mailbox *inboundmessage.Mailbox, now int64) error {
	wait := mailbox.TestSentAt + testCooldown - now
	if wait <= 0 || wait > testCooldown {
		wait = testCooldown
	}

	return errortypes.NewRateLimitError(
		"mailbox",
		"A test email went to this address moments ago. Wait for it to arrive, "+
			"or send another in {0, plural, one {# second} other {# seconds}}.",
		wait,
	).WithRetryAfter(time.Duration(wait) * time.Second)
}

func carriesTestToken(
	mailbox *inboundmessage.Mailbox,
	message *inboundmessage.InboundMessage,
) bool {
	if mailbox.TestToken == "" {
		return false
	}
	token := strings.ToLower(mailbox.TestToken)

	return strings.Contains(strings.ToLower(message.Subject), token) ||
		strings.Contains(strings.ToLower(message.TextBody), token)
}

func (s *Service) NoteTestReceived(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	message *inboundmessage.InboundMessage,
) *triage.Activity {
	if !carriesTestToken(mailbox, message) {
		return nil
	}

	matched, err := s.mailboxState.MarkTestReceived(ctx, inboundmessageports.MailboxTestRequest{
		TenantInfo: inboundqueue.TenantOfMessage(message),
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
