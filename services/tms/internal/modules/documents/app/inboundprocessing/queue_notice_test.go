package inboundprocessing

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReview_MovesAnOpenInbox(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusInReview,
			inboundmessage.ReviewAlways,
		),
	}
	svc, notifier, realtime := noticed(repo, "")

	_, err := svc.queue().Review(t.Context(), &inboundqueue.ReviewRequest{
		MessageID: repo.Message.ID,
		Status:    inboundmessage.StatusActioned,
	})
	require.NoError(t, err)

	require.Len(t, realtime.Published, 1)
	assert.Equal(t, repo.Message.ID, realtime.Published[0].RecordID)
	assert.Empty(t, notifier.requests, "settling a message is not news")
}
