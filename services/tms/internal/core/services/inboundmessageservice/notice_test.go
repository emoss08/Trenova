package inboundmessageservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type recordingRealtime struct {
	services.RealtimeService

	published []*services.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *services.PublishResourceInvalidationRequest,
) error {
	r.published = append(r.published, req)
	return nil
}

func noticed(repo *settleRepo, reply string) (*Service, *recordingNotifier, *recordingRealtime) {
	notifier := &recordingNotifier{}
	realtime := &recordingRealtime{}
	svc := settler(repo, reply)
	svc.notifier = notifier
	svc.realtime = realtime

	return svc, notifier, realtime
}

func TestProcessMessage_TellsTheInboxReadersAMessageIsWaiting(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAlways)}
	svc, notifier, realtime := noticed(repo,
		`{"classification":"Tender","confidence":0.9,"reasoning":"Offers a load."}`)

	_, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	require.Len(t, notifier.requests, 1)
	req := notifier.requests[0]
	assert.Equal(t, EventNeedsReview, req.Notification.EventType)
	assert.Equal(t, permission.ResourceInboundMessage, req.Resource)
	assert.Equal(t, permission.OpRead, req.Operation)
	assert.Positive(t, req.DedupeSince, "a reprocessed message must not tell everyone twice")
	require.NotNil(t, req.Notification.CorrelationID)
	assert.Equal(t, repo.message.ID.String(), *req.Notification.CorrelationID)
	assert.Contains(t, req.Notification.Data["link"], repo.message.ID.String())

	require.NotEmpty(t, realtime.published, "an open inbox moves")
	assert.Equal(t, "inbound_message", realtime.published[0].Resource)
	assert.Equal(t, repo.message.ID, realtime.published[0].RecordID)
}

/*
A message the mailbox handles without review is work that happened, not work
owed, and telling people about it would train them to ignore the notice that
matters. A quarantined one is critical on the watchtower, which already tells
the same people; this notice would be the second.
*/
func TestProcessMessage_SaysNothingAboutAMessageNobodyHasToLookAt(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle)}
	svc, notifier, realtime := noticed(repo,
		`{"classification":"StatusRequest","confidence":0.95,"reasoning":"Asks where a load is."}`)

	_, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Empty(t, notifier.requests)
	assert.NotEmpty(t, realtime.published, "the inbox still moves")

	quarantined := staged(inboundmessage.StatusQuarantined, inboundmessage.ReviewAlways)
	svc.notifyNeedsReview(t.Context(), quarantined)
	assert.Empty(t, notifier.requests)
}

func TestReview_MovesAnOpenInbox(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{message: staged(inboundmessage.StatusInReview, inboundmessage.ReviewAlways)}
	svc, notifier, realtime := noticed(repo, "")

	_, err := svc.Review(t.Context(), ReviewRequest{
		MessageID: repo.message.ID,
		Status:    inboundmessage.StatusActioned,
	})
	require.NoError(t, err)

	require.Len(t, realtime.published, 1)
	assert.Equal(t, repo.message.ID, realtime.published[0].RecordID)
	assert.Empty(t, notifier.requests, "settling a message is not news")
}
