package inboundmessageservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const mailboxTestNow int64 = 1_784_131_200

type testMailer struct {
	emailports.EmailService

	sent []*emailports.SendEmailRequest
	errs []error
}

func (m *testMailer) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	m.sent = append(m.sent, req)
	if len(m.errs) > 0 {
		err := m.errs[0]
		m.errs = m.errs[1:]
		if err != nil {
			return nil, err
		}
	}

	return &email.Message{}, nil
}

type mailboxTestHarness struct {
	svc    *Service
	state  *fakeMailboxState
	mailer *testMailer
	req    MailboxActionRequest
}

func newMailboxTestHarness(lastTestAt int64, sendErrs ...error) *mailboxTestHarness {
	mailbox := testMailbox()
	mailbox.TestSentAt = lastTestAt
	state := &fakeMailboxState{lastTestAt: lastTestAt}
	mailer := &testMailer{errs: sendErrs}

	return &mailboxTestHarness{
		svc: &Service{
			l:            zap.NewNop(),
			mailboxRepo:  &adminMailboxRepo{stored: mailbox},
			mailboxState: state,
			email:        mailer,
			now:          func() int64 { return mailboxTestNow },
		},
		state:  state,
		mailer: mailer,
		req: MailboxActionRequest{
			Actor: adminActor(mailbox.OrganizationID, mailbox.BusinessUnitID),
			ID:    mailbox.ID,
		},
	}
}

func TestSendMailboxTest_SendsOneTestAndStartsTheCooldown(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(0)

	updated, err := h.svc.SendMailboxTest(t.Context(), h.req)
	require.NoError(t, err)

	require.Len(t, h.mailer.sent, 1)
	assert.Equal(t, []string{"tenders@acme-logistics.com"}, h.mailer.sent[0].To)
	require.Len(t, h.state.claims, 1)
	assert.Equal(t, mailboxTestNow, h.state.claims[0].At)
	assert.Equal(t, mailboxTestNow-60, h.state.claims[0].LastSentBefore)
	assert.Equal(t, mailboxTestNow, updated.TestSentAt)
	assert.Equal(t, h.state.claims[0].Token, h.mailer.sent[0].IdempotencyKey)
}

func TestSendMailboxTest_RefusesASecondTestInsideTheCooldownWithoutSending(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(mailboxTestNow - 20)

	_, err := h.svc.SendMailboxTest(t.Context(), h.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsRateLimitError(err))
	retryAfter, ok := errortypes.RetryAfterOf(err)
	require.True(t, ok)
	assert.Equal(t, 40*time.Second, retryAfter)
	rateLimited, ok := errors.AsType[*errortypes.RateLimitError](err)
	require.True(t, ok)
	assert.Equal(t, []any{int64(40)}, rateLimited.Args)
	assert.Empty(t, h.mailer.sent)
	assert.Empty(t, h.state.claims)
}

func TestSendMailboxTest_RefusesBackToBackRequests(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(0)

	_, err := h.svc.SendMailboxTest(t.Context(), h.req)
	require.NoError(t, err)
	_, err = h.svc.SendMailboxTest(t.Context(), h.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsRateLimitError(err))
	retryAfter, ok := errortypes.RetryAfterOf(err)
	require.True(t, ok)
	assert.Equal(t, 60*time.Second, retryAfter,
		"a test claimed after the mailbox was read waits the whole cooldown")
	assert.Len(t, h.mailer.sent, 1)
}

func TestSendMailboxTest_SendsAgainOnceTheCooldownHasPassed(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(mailboxTestNow - 60)

	_, err := h.svc.SendMailboxTest(t.Context(), h.req)

	require.NoError(t, err)
	assert.Len(t, h.mailer.sent, 1)
}

func TestSendMailboxTest_AFailedSendDoesNotHoldTheCooldown(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(0, errors.New("smtp unreachable"))

	_, err := h.svc.SendMailboxTest(t.Context(), h.req)
	require.Error(t, err)
	require.Len(t, h.state.testsSent, 1)
	assert.Zero(t, h.state.testsSent[0].At, "the failed claim is rolled back to the previous test")

	_, err = h.svc.SendMailboxTest(t.Context(), h.req)

	require.NoError(t, err)
	assert.Len(t, h.mailer.sent, 2)
}

func TestSendMailboxTest_RefusesAPausedMailboxBeforeClaiming(t *testing.T) {
	t.Parallel()

	h := newMailboxTestHarness(0)
	h.svc.mailboxRepo.(*adminMailboxRepo).stored.Status = inboundmessage.MailboxInactive

	_, err := h.svc.SendMailboxTest(t.Context(), h.req)

	require.Error(t, err)
	assert.False(t, errortypes.IsRateLimitError(err))
	assert.Empty(t, h.state.claims)
	assert.Empty(t, h.mailer.sent)
}
