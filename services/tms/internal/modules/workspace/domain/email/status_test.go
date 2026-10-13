package email

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextMessageStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		current MessageStatus
		event   EventType
		want    MessageStatus
		changed bool
	}{
		{"sent moves a queued message on", MessageStatusSending, EventTypeSent, MessageStatusSent, true},
		{"delivered after sent", MessageStatusSent, EventTypeDelivered, MessageStatusDelivered, true},
		{"a late delivered never undoes opened", MessageStatusOpened, EventTypeDelivered, MessageStatusOpened, false},
		{"a late sent never undoes delivered", MessageStatusDelivered, EventTypeSent, MessageStatusDelivered, false},
		{"opened after a bounce stays bounced", MessageStatusBounced, EventTypeOpened, MessageStatusBounced, false},
		{"a bounce after delivery is recorded", MessageStatusDelivered, EventTypeBounced, MessageStatusBounced, true},
		{"a complaint always wins", MessageStatusOpened, EventTypeComplained, MessageStatusComplained, true},
		{"a bounce never replaces a complaint", MessageStatusComplained, EventTypeBounced, MessageStatusComplained, false},
		{"failed before delivery", MessageStatusSent, EventTypeFailed, MessageStatusFailed, true},
		{"failed after delivery is ignored", MessageStatusDelivered, EventTypeFailed, MessageStatusDelivered, false},
		{"an unknown event changes nothing", MessageStatusSent, EventType("Unknown"), MessageStatusSent, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, changed := NextMessageStatus(tc.current, tc.event)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.changed, changed)
		})
	}
}
