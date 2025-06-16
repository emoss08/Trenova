package notification_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationHistory_Validate(t *testing.T) {
	tests := []struct {
		name      string
		history   *notification.NotificationHistory
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid notification history",
			history: &notification.NotificationHistory{
				NotificationID: pulid.MustNew("notif_"),
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Title:          "Test Notification",
				Message:        "This is a test notification",
				Priority:       notification.PriorityMedium,
				Channel:        notification.ChannelUser,
				EventType:      notification.EventEntityUpdated,
				DeliveryStatus: notification.DeliveryStatusPending,
			},
			wantError: false,
		},
		{
			name: "missing notification ID",
			history: &notification.NotificationHistory{
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Title:          "Test Notification",
				Message:        "This is a test notification",
				Priority:       notification.PriorityMedium,
				Channel:        notification.ChannelUser,
				EventType:      notification.EventEntityUpdated,
				DeliveryStatus: notification.DeliveryStatusPending,
			},
			wantError: true,
			errorMsg:  "Notification ID is required",
		},
		{
			name: "missing title",
			history: &notification.NotificationHistory{
				NotificationID: pulid.MustNew("notif_"),
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Message:        "This is a test notification",
				Priority:       notification.PriorityMedium,
				Channel:        notification.ChannelUser,
				EventType:      notification.EventEntityUpdated,
				DeliveryStatus: notification.DeliveryStatusPending,
			},
			wantError: true,
			errorMsg:  "Title is required",
		},
		{
			name: "invalid priority",
			history: &notification.NotificationHistory{
				NotificationID: pulid.MustNew("notif_"),
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Title:          "Test Notification",
				Message:        "This is a test notification",
				Priority:       notification.Priority("invalid"),
				Channel:        notification.ChannelUser,
				EventType:      notification.EventEntityUpdated,
				DeliveryStatus: notification.DeliveryStatusPending,
			},
			wantError: true,
			errorMsg:  "Invalid priority level",
		},
		{
			name: "invalid delivery status",
			history: &notification.NotificationHistory{
				NotificationID: pulid.MustNew("notif_"),
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Title:          "Test Notification",
				Message:        "This is a test notification",
				Priority:       notification.PriorityMedium,
				Channel:        notification.ChannelUser,
				EventType:      notification.EventEntityUpdated,
				DeliveryStatus: notification.DeliveryStatus("invalid"),
			},
			wantError: true,
			errorMsg:  "Invalid delivery status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := errors.NewMultiError()
			tt.history.Validate(context.Background(), multiErr)

			if tt.wantError {
				assert.True(t, multiErr.HasErrors())
				if tt.errorMsg != "" {
					assert.Contains(t, multiErr.Error(), tt.errorMsg)
				}
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestNotificationHistory_IsRead(t *testing.T) {
	tests := []struct {
		name    string
		history *notification.NotificationHistory
		want    bool
	}{
		{
			name:    "unread notification",
			history: &notification.NotificationHistory{},
			want:    false,
		},
		{
			name: "read notification",
			history: &notification.NotificationHistory{
				ReadAt: &[]int64{timeutils.NowUnix()}[0],
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.history.IsRead()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationHistory_IsDismissed(t *testing.T) {
	tests := []struct {
		name    string
		history *notification.NotificationHistory
		want    bool
	}{
		{
			name:    "not dismissed notification",
			history: &notification.NotificationHistory{},
			want:    false,
		},
		{
			name: "dismissed notification",
			history: &notification.NotificationHistory{
				DismissedAt: &[]int64{timeutils.NowUnix()}[0],
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.history.IsDismissed()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationHistory_IsExpired(t *testing.T) {
	now := timeutils.NowUnix()
	past := now - 3600   // 1 hour ago
	future := now + 3600 // 1 hour from now

	tests := []struct {
		name    string
		history *notification.NotificationHistory
		want    bool
	}{
		{
			name:    "no expiration set",
			history: &notification.NotificationHistory{},
			want:    false,
		},
		{
			name: "expired notification",
			history: &notification.NotificationHistory{
				ExpiresAt: &past,
			},
			want: true,
		},
		{
			name: "not expired notification",
			history: &notification.NotificationHistory{
				ExpiresAt: &future,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.history.IsExpired()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationHistory_MarkAsRead(t *testing.T) {
	history := &notification.NotificationHistory{}
	require.Nil(t, history.ReadAt)

	history.MarkAsRead()
	assert.NotNil(t, history.ReadAt)
	assert.True(t, history.IsRead())
}

func TestNotificationHistory_MarkAsDismissed(t *testing.T) {
	history := &notification.NotificationHistory{}
	require.Nil(t, history.DismissedAt)

	history.MarkAsDismissed()
	assert.NotNil(t, history.DismissedAt)
	assert.True(t, history.IsDismissed())
}

func TestNotificationHistory_MarkAsClicked(t *testing.T) {
	history := &notification.NotificationHistory{}
	require.Nil(t, history.ClickedAt)

	history.MarkAsClicked()
	assert.NotNil(t, history.ClickedAt)
}

func TestNotificationHistory_SetDelivered(t *testing.T) {
	history := &notification.NotificationHistory{
		DeliveryStatus: notification.DeliveryStatusPending,
	}
	require.Nil(t, history.DeliveredAt)

	history.SetDelivered()
	assert.Equal(t, notification.DeliveryStatusDelivered, history.DeliveryStatus)
	assert.NotNil(t, history.DeliveredAt)
}

func TestNotificationHistory_SetFailed(t *testing.T) {
	history := &notification.NotificationHistory{
		DeliveryStatus: notification.DeliveryStatusPending,
		RetryCount:     0,
	}

	reason := "Connection timeout"
	history.SetFailed(reason)

	assert.Equal(t, notification.DeliveryStatusFailed, history.DeliveryStatus)
	assert.Equal(t, reason, history.FailureReason)
	assert.Equal(t, 1, history.RetryCount)
}

func TestNotificationHistory_CompleteEntityReference(t *testing.T) {
	history := &notification.NotificationHistory{
		EntityType:  permission.ResourceShipment,
		EntityID:    pulid.MustNew("ship_"),
		UpdateType:  notification.UpdateTypeStatusChange,
		UpdatedByID: pulid.MustNew("user_"),
	}

	assert.Equal(t, permission.ResourceShipment, history.EntityType)
	assert.False(t, history.EntityID.IsNil())
	assert.Equal(t, notification.UpdateTypeStatusChange, history.UpdateType)
	assert.False(t, history.UpdatedByID.IsNil())
}
