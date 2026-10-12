package inboundreplyservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendRequest(f *fixture, key string) *SendRequest {
	return &SendRequest{
		TenantInfo:     testTenant,
		UserID:         testUserID,
		MessageID:      f.message.ID,
		Subject:        "Re: Where is my load?",
		Body:           "Hi Dana,\n\nS-104251 is running about two hours behind.",
		IdempotencyKey: key,
	}
}

func TestSendThreadsReplyFromMailbox(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.email.result.Status = email.MessageStatusSent
	f.email.result.SentAt = 1_791_000_100
	f.email.result.ProviderMessageID = "provider-1"

	reply, err := f.service.Send(t.Context(), sendRequest(f, "key-1"))
	require.NoError(t, err)

	assert.Equal(t, triage.ReplySent, reply.Status)
	assert.Equal(t, "dispatch@trenova.example", reply.FromAddress)
	assert.Equal(t, "edi@partner.example", reply.ToAddress)
	assert.Equal(t, int64(1_791_000_100), reply.SentAt)
	assert.Equal(t, "provider-1", reply.ProviderMessageID)
	assert.True(t, strings.HasSuffix(reply.MessageKey, "@trenova.example"))

	require.Len(t, f.email.requests, 1)
	req := f.email.requests[0]
	assert.Equal(t, "dispatch@trenova.example", req.FromEmail)
	assert.Equal(t, []string{"edi@partner.example"}, req.To)
	assert.Equal(t, email.PurposeOperations, req.Purpose)
	assert.Equal(t, inboundmessageports.InboundReplyIdempotencyPrefix+reply.ID.String(), req.IdempotencyKey)
	assert.Equal(t, "<abc123@partner.example>", req.Headers["In-Reply-To"])
	assert.Contains(t, req.Headers["References"], "<root@partner.example>")
	assert.Contains(t, req.Headers["References"], "<abc123@partner.example>")
	assert.True(t, strings.HasSuffix(req.Headers["Message-ID"], "@trenova.example>"))
	assert.NotEmpty(t, req.HTML)

	assert.True(t, containsKind(f.recorder.kinds(), triage.ActivityReplied))
	assert.Contains(t, f.notifier.changed, f.message.ID)
}

func TestSendQueuedStaysSending(t *testing.T) {
	t.Parallel()

	f := newFixture()

	reply, err := f.service.Send(t.Context(), sendRequest(f, ""))
	require.NoError(t, err)

	assert.Equal(t, triage.ReplySending, reply.Status)
	assert.False(t, containsKind(f.recorder.kinds(), triage.ActivityReplied))
}

func TestSendClearsTheDraft(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Body:       "Draft body",
	})
	require.NoError(t, err)
	require.Len(t, f.replies.byStatus(triage.ReplyDraft), 1)

	_, err = f.service.Send(t.Context(), sendRequest(f, "key-draft"))
	require.NoError(t, err)

	assert.Empty(t, f.replies.byStatus(triage.ReplyDraft))
}

func TestSendRetryWithSameKeyReturnsExisting(t *testing.T) {
	t.Parallel()

	f := newFixture()

	first, err := f.service.Send(t.Context(), sendRequest(f, "key-retry"))
	require.NoError(t, err)
	second, err := f.service.Send(t.Context(), sendRequest(f, "key-retry"))
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, f.email.requests, 1)
}

func TestSendRefusedRecipientSuggestsAlternate(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.work.alternates = []string{"dispatch@partner.example"}
	f.email.err = errortypes.NewBusinessError("Recipient edi@partner.example was rejected by the provider")

	reply, err := f.service.Send(t.Context(), sendRequest(f, "key-refused"))
	require.NoError(t, err)

	assert.Equal(t, triage.ReplyFailed, reply.Status)
	assert.NotEmpty(t, reply.FailureText)
	assert.Equal(t, "dispatch@partner.example", reply.SuggestedAddress)
	require.NotNil(t, f.work.asked)
	assert.Equal(t, "partner.example", f.work.asked.Domain)
	assert.Equal(t, "edi@partner.example", f.work.asked.Exclude)
	assert.Equal(t, 1, f.work.asked.Limit)
	assert.True(t, containsKind(f.recorder.kinds(), triage.ActivityReplyFailed))
}

func TestSendUnqueuedFailureDoesNotSuggest(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.work.alternates = []string{"dispatch@partner.example"}
	f.email.err = errors.New("temporal unavailable")

	reply, err := f.service.Send(t.Context(), sendRequest(f, ""))
	require.NoError(t, err)

	assert.Equal(t, triage.ReplyFailed, reply.Status)
	assert.Equal(t, failureUnqueued, reply.FailureText)
	assert.Empty(t, reply.SuggestedAddress)
	assert.Nil(t, f.work.asked)
}

func TestSendWithoutOutboundEmailFails(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.service.email = nil

	reply, err := f.service.Send(t.Context(), sendRequest(f, ""))
	require.NoError(t, err)

	assert.Equal(t, triage.ReplyFailed, reply.Status)
	assert.Equal(t, failureNoOutbound, reply.FailureText)
}

func TestSendToAlternateAddress(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := sendRequest(f, "")
	req.ToAddress = " Dispatch@Partner.example "

	reply, err := f.service.Send(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, "dispatch@partner.example", reply.ToAddress)
	assert.Equal(t, []string{"dispatch@partner.example"}, f.email.requests[0].To)
}

func TestSendRejectsInvalidAddress(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := sendRequest(f, "")
	req.ToAddress = "not an address"

	_, err := f.service.Send(t.Context(), req)
	require.Error(t, err)
	assert.Empty(t, f.email.requests)
}

func TestSendRequiresBody(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := sendRequest(f, "")
	req.Body = "   "

	_, err := f.service.Send(t.Context(), req)
	require.Error(t, err)
	assert.Empty(t, f.email.requests)
}

func TestEmailMessageChangedBounceFailsReply(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.work.alternates = []string{"ops@partner.example"}
	reply, err := f.service.Send(t.Context(), sendRequest(f, "key-bounce"))
	require.NoError(t, err)
	require.Equal(t, triage.ReplySending, reply.Status)

	f.service.EmailMessageChanged(t.Context(), emailports.EmailMessageChange{
		Message: &email.Message{
			OrganizationID: testOrgID,
			BusinessUnitID: testBUID,
			Status:         email.MessageStatusBounced,
			IdempotencyKey: inboundmessageports.InboundReplyIdempotencyPrefix + reply.ID.String(),
		},
		Event: &email.Event{Raw: map[string]any{
			"bounce": map[string]any{"description": "550 5.1.1 mailbox unavailable"},
		}},
	})

	failed := f.replies.byStatus(triage.ReplyFailed)
	require.Len(t, failed, 1)
	assert.Equal(t, reply.ID, failed[0].ID)
	assert.NotEmpty(t, failed[0].FailureText)
	assert.Equal(t, "ops@partner.example", failed[0].SuggestedAddress)
}

func TestEmailMessageChangedDeliveredMarksSent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	reply, err := f.service.Send(t.Context(), sendRequest(f, "key-delivered"))
	require.NoError(t, err)

	change := emailports.EmailMessageChange{Message: &email.Message{
		OrganizationID: testOrgID,
		BusinessUnitID: testBUID,
		Status:         email.MessageStatusDelivered,
		IdempotencyKey: inboundmessageports.InboundReplyIdempotencyPrefix + reply.ID.String(),
	}}
	f.service.EmailMessageChanged(t.Context(), change)
	f.service.EmailMessageChanged(t.Context(), change)

	sent := f.replies.byStatus(triage.ReplySent)
	require.Len(t, sent, 1)
	assert.Equal(t, int64(1_791_000_000), sent[0].SentAt)

	replied := 0
	for _, kind := range f.recorder.kinds() {
		if kind == triage.ActivityReplied {
			replied++
		}
	}
	assert.Equal(t, 1, replied)
}

func TestEmailMessageChangedIgnoresOtherMail(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.service.EmailMessageChanged(t.Context(), emailports.EmailMessageChange{Message: &email.Message{
		Status:         email.MessageStatusBounced,
		IdempotencyKey: "invoice:abc",
	}})
	f.service.EmailMessageChanged(t.Context(), emailports.EmailMessageChange{})

	assert.Empty(t, f.recorder.kinds())
}

func TestReplySubject(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Re: Load 123", replySubject("Load 123"))
	assert.Equal(t, "RE: Load 123", replySubject("RE: Load 123"))
	assert.Equal(t, "Re: your message", replySubject("  "))
}
