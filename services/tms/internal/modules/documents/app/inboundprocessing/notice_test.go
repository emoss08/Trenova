package inboundprocessing

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingNotifier struct {
	requests []notificationservice.NotifyPermittedRequest
}

func (r *recordingNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	r.requests = append(r.requests, req)
	return 1, nil
}

func noticed(
	repo *inboundqueuetest.SettleRepo,
	reply string,
) (*testService, *recordingNotifier, *inboundqueuetest.RecordingRealtime) {
	notifier := &recordingNotifier{}
	realtime := &inboundqueuetest.RecordingRealtime{}
	svc := settler(repo, reply)
	svc.notifier = notifier
	svc.realtime = realtime
	svc.notify = inboundqueue.NewNotifier(
		inboundqueue.NotifierParams{Logger: zap.NewNop(), Realtime: realtime},
	)

	return svc, notifier, realtime
}

func TestProcessMessage_TellsTheInboxReadersAMessageIsWaiting(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAlways,
		),
	}
	svc, notifier, realtime := noticed(repo,
		`{"classification":"Tender","confidence":0.9,"reasoning":"Offers a load."}`)

	_, err := svc.processing().ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	require.Len(t, notifier.requests, 1)
	req := notifier.requests[0]
	assert.Equal(t, EventNeedsReview, req.Notification.EventType)
	assert.Equal(t, permission.ResourceInboundMessage, req.Resource)
	assert.Equal(t, permission.OpRead, req.Operation)
	assert.Positive(t, req.DedupeSince, "a reprocessed message must not tell everyone twice")
	require.NotNil(t, req.Notification.CorrelationID)
	assert.Equal(t, repo.Message.ID.String(), *req.Notification.CorrelationID)
	assert.Contains(t, req.Notification.Data["link"], repo.Message.ID.String())

	require.NotEmpty(t, realtime.Published, "an open inbox moves")
	assert.Equal(t, "inbound_message", realtime.Published[0].Resource)
	assert.Equal(t, repo.Message.ID, realtime.Published[0].RecordID)
}

/*
A message the mailbox handles without review is work that happened, not work
owed, and telling people about it would train them to ignore the notice that
matters. A quarantined one is critical on the watchtower, which already tells
the same people; this notice would be the second.
*/
func TestProcessMessage_SaysNothingAboutAMessageNobodyHasToLookAt(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	svc, notifier, realtime := noticed(repo,
		`{"classification":"StatusRequest","confidence":0.95,"reasoning":"Asks where a load is."}`)

	_, err := svc.processing().ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Empty(t, notifier.requests)
	assert.NotEmpty(t, realtime.Published, "the inbox still moves")

	quarantined := inboundqueuetest.Staged(
		inboundmessage.StatusQuarantined,
		inboundmessage.ReviewAlways,
	)
	svc.processing().notifyNeedsReview(t.Context(), quarantined)
	assert.Empty(t, notifier.requests)
}
