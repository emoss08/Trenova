package inboundprocessing

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/stretchr/testify/assert"
)

func TestApplyDue_StartsTheClockOnlyForWorkOwed(t *testing.T) {
	t.Parallel()

	targets := triage.ResolveTargets(nil)
	message := &inboundmessage.InboundMessage{
		ReceivedAt:     inboundqueuetest.TestNow,
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationStatusRequest,
	}
	ApplyDue(message, targets)
	assert.Equal(t, inboundqueuetest.TestNow+60*60, message.DueAt)

	message.Status = inboundmessage.StatusActioned
	message.Parked = inboundmessage.ParkingSnoozed
	ApplyDue(message, targets)
	assert.Zero(t, message.DueAt)
	assert.Empty(t, message.Parked)
}
