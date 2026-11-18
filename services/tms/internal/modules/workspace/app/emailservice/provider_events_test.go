package emailservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
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

	eventType, known := resendEventType(&payload)
	require.True(t, known)
	require.Equal(t, email.EventTypeBounced, eventType)
	require.Equal(t, "ap@customer.example.com", resendRecipient(&payload))
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
			got, known := resendEventType(&payload)
			require.Equal(t, tt.known, known)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestResendRecipientIsBlankWhenTheEventCoversSeveralRecipients(t *testing.T) {
	t.Parallel()

	var payload resendWebhookPayload
	payload.Data.To = []string{"a@example.com", "b@example.com"}
	require.Empty(t, resendRecipient(&payload))
}

func TestParseResendEventTakesTheDeliveryIDAndTheMessage(t *testing.T) {
	t.Parallel()

	event, known, err := ParseResendEvent([]byte(`{
		"type": "email.complained",
		"data": {"email_id": "re_123", "to": ["ap@customer.example.com"]}
	}`), "msg_2Kq")

	require.NoError(t, err)
	require.True(t, known)
	require.Equal(t, "msg_2Kq", event.Event.ProviderEventID)
	require.Equal(t, email.ProviderResend, event.Event.Provider)
	require.Equal(t, email.EventTypeComplained, event.Event.Type)
	require.Equal(t, "ap@customer.example.com", event.Event.Recipient)
	require.Equal(t, "re_123", event.ProviderMessageID)
	require.Equal(t, email.SuppressionReasonComplaint, event.SuppressionReason)
	require.Equal(t, "email.complained", event.Event.Raw["type"])
}

func TestParseProviderEventRefusesWhatIsNotJSON(t *testing.T) {
	t.Parallel()

	for _, provider := range []email.Provider{email.ProviderResend, email.ProviderPostmark, "SES"} {
		_, known, err := ParseProviderEvent(provider, []byte("not json"), "id")
		require.ErrorIs(t, err, ErrMalformedProviderEvent, provider)
		require.False(t, known, provider)
	}
}

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

			got, known := postmarkEventType(&tt.payload)
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
		postmarkProviderEventID(&payload),
	)
}

func TestPostmarkProviderEventIDUsesProviderIDWhenPresent(t *testing.T) {
	t.Parallel()

	require.Equal(t, "12345", postmarkProviderEventID(&postmarkWebhookPayload{ID: float64(12345)}))
}

func TestPostmarkSuppressionReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  postmarkWebhookPayload
		expected email.SuppressionReason
	}{
		{
			name:     "hard bounce",
			payload:  postmarkWebhookPayload{RecordType: "Bounce", Type: "HardBounce"},
			expected: email.SuppressionReasonHardBounce,
		},
		{
			name:    "soft bounce",
			payload: postmarkWebhookPayload{RecordType: "Bounce", Type: "SoftBounce"},
		},
		{
			name:     "spam complaint",
			payload:  postmarkWebhookPayload{RecordType: "SpamComplaint"},
			expected: email.SuppressionReasonComplaint,
		},
		{
			name:    "delivery",
			payload: postmarkWebhookPayload{RecordType: "Delivery"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, postmarkSuppressionReason(&tt.payload))
		})
	}
}
