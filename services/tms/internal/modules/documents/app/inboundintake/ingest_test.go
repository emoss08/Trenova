package inboundintake

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func postmarkReply(subject, inReplyTo string) *ReceiveWebhookRequest {
	body := []byte(`{
		"MessageID": "pm-reply-1",
		"From": "dispatch@bigshipper.com",
		"FromName": "Dispatch",
		"To": "tenders@acme-logistics.com",
		"Subject": "` + subject + `",
		"TextBody": "Here you go.",
		"Headers": [
			{"Name": "Message-ID", "Value": "<reply-1@bigshipper.com>"},
			{"Name": "In-Reply-To", "Value": "` + inReplyTo + `"},
			{"Name": "References", "Value": "` + inReplyTo + `"}
		]
	}`)

	return &ReceiveWebhookRequest{
		MailboxToken: inboundqueuetest.TestToken,
		Body:         body,
		Authorization: "Basic " + base64.StdEncoding.EncodeToString(
			[]byte(testPostmarkCredentials),
		),
		ReceivedAt: time.Unix(inboundqueuetest.TestNow, 0),
	}
}

func ingestService(mailbox *inboundmessage.Mailbox, states ...*inboundmessageports.QueueState) (
	*testService, *stubMessageRepo, *stubWorkflows, *inboundqueuetest.QueueFakes,
) {
	messages := &stubMessageRepo{}
	workflows := &stubWorkflows{}
	svc, fakes := withQueue(&testService{
		l:           zap.NewNop(),
		mailboxRepo: &stubMailboxRepo{mailbox: mailbox},
		messageRepo: messages,
		storage:     &stubStorage{},
		encryption:  inboundqueuetest.PlainSecrets{},
		workflows:   workflows,
	}, states...)

	return svc, messages, workflows, fakes
}

func TestReceiveWebhook_AReplyJoinsItsThreadAndAnswersTheMessageWaitingOnIt(t *testing.T) {
	t.Parallel()

	mailbox := postmarkMailbox()
	awaiting := &inboundmessageports.QueueState{
		ID:             pulid.MustNew("imsg_"),
		OrganizationID: mailbox.OrganizationID,
		BusinessUnitID: mailbox.BusinessUnitID,
		Status:         inboundmessage.StatusInReview,
		ThreadKey:      "original@acme-logistics.com",
		Parked:         inboundmessage.ParkingAwaitingReply,
		AwaitingUntil:  inboundqueuetest.TestNow + 86400,
	}
	svc, messages, workflows, fakes := ingestService(mailbox, awaiting)
	fakes.Queue.ThreadKeys["original@acme-logistics.com"] = "original@acme-logistics.com"

	_, err := svc.intake().ReceiveWebhook(t.Context(), postmarkReply(
		"RE: Load 88213 tender", "<original@acme-logistics.com>",
	))
	require.NoError(t, err)

	require.NotNil(t, messages.created)
	assert.Equal(t, "original@acme-logistics.com", messages.created.ThreadKey)
	answered := fakes.Queue.State(awaiting.ID)
	assert.Equal(t, inboundmessage.StatusActioned, answered.Status)
	assert.Empty(t, answered.Parked)
	assert.Equal(t, []pulid.ID{awaiting.ID}, fakes.Wakes.Cancelled)
	assert.Contains(t, fakes.Activities.Kinds(), triage.ActivityTheyReplied)
	require.Len(t, workflows.started, 1, "the reply itself lands in the queue to be read")
	require.Len(t, fakes.Mailboxes.Received, 1)
	assert.Equal(t, messages.created.ReceivedAt, fakes.Mailboxes.Received[0].ReceivedAt)
}

func TestReceiveWebhook_ABlockedSenderIsIgnoredBeforeAnythingReadsIt(t *testing.T) {
	t.Parallel()

	mailbox := postmarkMailbox()
	mailbox.BlockedSenders = []string{"bigshipper.com"}
	svc, messages, workflows, fakes := ingestService(mailbox)

	_, err := svc.intake().ReceiveWebhook(t.Context(), postmarkDelivery(testPostmarkCredentials))
	require.NoError(t, err)

	require.NotNil(t, messages.created)
	assert.Equal(t, inboundmessage.StatusIgnored, messages.created.Status)
	assert.Empty(t, workflows.started, "no model is asked about a blocked sender")
	assert.Equal(t, []triage.ActivityKind{triage.ActivityIgnored}, fakes.Activities.Kinds())
	assert.Len(t, fakes.Mailboxes.Received, 1, "the source is still alive")
}

func TestReceiveWebhook_ATestMessageMarksTheSourceAndLandsLikeAnyOther(t *testing.T) {
	t.Parallel()

	mailbox := postmarkMailbox()
	mailbox.TestToken = "trv-test-0a1b2c3d4e5f6a7b"
	svc, messages, workflows, fakes := ingestService(mailbox)

	_, err := svc.intake().ReceiveWebhook(t.Context(), postmarkReply(
		"Trenova test message trv-test-0a1b2c3d4e5f6a7b", "",
	))
	require.NoError(t, err)

	require.NotNil(t, messages.created)
	require.Len(t, fakes.Mailboxes.TestReceived, 1)
	assert.Equal(t, mailbox.TestToken, fakes.Mailboxes.TestReceived[0].Token)
	assert.Equal(t, []triage.ActivityKind{triage.ActivityTestReceived}, fakes.Activities.Kinds())
	assert.Len(t, workflows.started, 1)
}
