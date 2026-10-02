package emailhandler

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/stretchr/testify/require"
)

func TestPostmarkEventTypeMapsSupportedRecordTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  postmarkWebhookPayload
		expected email.EventType
		known    bool
	}{
		{name: "delivery", payload: postmarkWebhookPayload{RecordType: "Delivery"}, expected: email.EventTypeDelivered, known: true},
		{name: "hard bounce", payload: postmarkWebhookPayload{RecordType: "Bounce", Type: "HardBounce"}, expected: email.EventTypeBounced, known: true},
		{name: "soft bounce is not a failure", payload: postmarkWebhookPayload{RecordType: "Bounce", Type: "SoftBounce"}},
		{name: "spam complaint", payload: postmarkWebhookPayload{RecordType: "SpamComplaint"}, expected: email.EventTypeComplained, known: true},
		{name: "open", payload: postmarkWebhookPayload{RecordType: "Open"}, expected: email.EventTypeOpened, known: true},
		{name: "click", payload: postmarkWebhookPayload{RecordType: "Click"}, expected: email.EventTypeClicked, known: true},
		{name: "unknown is ignored", payload: postmarkWebhookPayload{RecordType: "SubscriptionChange"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, known := postmarkEventType(tt.payload)
			require.Equal(t, tt.known, known)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestPostmarkProviderEventIDFallsBackToDeterministicFields(t *testing.T) {
	t.Parallel()

	payload := postmarkWebhookPayload{
		RecordType:  "Delivery",
		MessageID:   "message-id",
		Recipient:   "ops@example.com",
		DeliveredAt: "2026-05-29T12:00:00Z",
	}

	require.Equal(
		t,
		"Delivery:message-id:ops@example.com:2026-05-29T12:00:00Z",
		postmarkProviderEventID(payload),
	)
}

func TestPostmarkProviderEventIDUsesProviderIDWhenPresent(t *testing.T) {
	t.Parallel()

	require.Equal(t, "12345", postmarkProviderEventID(postmarkWebhookPayload{ID: float64(12345)}))
}

func TestPostmarkSuppressionReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  postmarkWebhookPayload
		expected email.SuppressionReason
	}{
		{
			name: "hard bounce",
			payload: postmarkWebhookPayload{
				RecordType: "Bounce",
				Type:       "HardBounce",
			},
			expected: email.SuppressionReasonHardBounce,
		},
		{
			name: "soft bounce",
			payload: postmarkWebhookPayload{
				RecordType: "Bounce",
				Type:       "SoftBounce",
			},
			expected: "",
		},
		{
			name: "spam complaint",
			payload: postmarkWebhookPayload{
				RecordType: "SpamComplaint",
			},
			expected: email.SuppressionReasonComplaint,
		},
		{
			name: "delivery",
			payload: postmarkWebhookPayload{
				RecordType: "Delivery",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, postmarkSuppressionReason(tt.payload))
		})
	}
}
