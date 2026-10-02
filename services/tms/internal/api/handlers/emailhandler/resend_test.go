package emailhandler

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/stretchr/testify/require"
)

func TestResendWebhookPayloadReadsRecipientList(t *testing.T) {
	t.Parallel()

	var payload resendWebhookPayload
	require.NoError(t, sonic.Unmarshal([]byte(`{
		"type": "email.bounced",
		"created_at": "2026-10-01T12:00:00.000Z",
		"data": {
			"email_id": "re_123",
			"to": ["ap@customer.example.com"],
			"bounce": {"type": "Permanent", "subType": "General"}
		}
	}`), &payload))

	eventType, known := resendEventType(payload)
	require.True(t, known)
	require.Equal(t, email.EventTypeBounced, eventType)
	require.Equal(t, "ap@customer.example.com", resendRecipient(payload))
}

func TestResendEventTypeMapsKnownEventsAndIgnoresTheRest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType string
		bounce    string
		expected  email.EventType
		known     bool
	}{
		{eventType: "email.sent", expected: email.EventTypeSent, known: true},
		{eventType: "email.delivered", expected: email.EventTypeDelivered, known: true},
		{eventType: "email.opened", expected: email.EventTypeOpened, known: true},
		{eventType: "email.clicked", expected: email.EventTypeClicked, known: true},
		{eventType: "email.bounced", bounce: "Permanent", expected: email.EventTypeBounced, known: true},
		{eventType: "email.bounced", bounce: "Transient"},
		{eventType: "email.complained", expected: email.EventTypeComplained, known: true},
		{eventType: "email.failed", expected: email.EventTypeFailed, known: true},
		{eventType: "email.delivery_delayed"},
		{eventType: "contact.created"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType+tt.bounce, func(t *testing.T) {
			t.Parallel()

			var payload resendWebhookPayload
			payload.Type = tt.eventType
			payload.Data.Bounce.Type = tt.bounce
			got, known := resendEventType(payload)
			require.Equal(t, tt.known, known)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestResendRecipientIsBlankWhenTheEventCoversSeveralRecipients(t *testing.T) {
	t.Parallel()

	var payload resendWebhookPayload
	payload.Data.To = []string{"a@example.com", "b@example.com"}
	require.Empty(t, resendRecipient(payload))
}
