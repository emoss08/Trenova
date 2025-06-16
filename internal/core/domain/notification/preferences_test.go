package notification_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
)

func TestNotificationPreference_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pref      *notification.NotificationPreference
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid preference with all updates",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15, // Default value
			},
			wantError: false,
		},
		{
			name: "valid preference with specific update types",
			pref: &notification.NotificationPreference{
				UserID:         pulid.MustNew("user_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Resource:       permission.ResourceShipment,
				UpdateTypes: []notification.UpdateType{
					notification.UpdateTypeStatusChange,
					notification.UpdateTypeAssignment,
				},
				NotifyOnAllUpdates:   false,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15, // Default value
			},
			wantError: false,
		},
		{
			name: "missing user ID",
			pref: &notification.NotificationPreference{
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "User ID is required",
		},
		{
			name: "missing organization ID",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "Organization ID is required",
		},
		{
			name: "missing resource",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "Resource is required",
		},
		{
			name: "invalid resource type",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.Resource("invalid_resource"),
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "Resource must be a valid resource type that supports notifications",
		},
		{
			name: "missing update types when not notifying on all",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   false,
				UpdateTypes:          []notification.UpdateType{},
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "At least one update type is required when not notifying on all updates",
		},
		{
			name: "missing preferred channels",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{},
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "At least one preferred channel is required",
		},
		{
			name: "quiet hours enabled without start time",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				QuietHoursEnabled:    true,
				QuietHoursEnd:        "08:00",
				BatchIntervalMinutes: 15,
			},
			wantError: true,
			errorMsg:  "Quiet hours start time is required when quiet hours are enabled",
		},
		{
			name: "invalid batch interval",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchNotifications:   true,
				BatchIntervalMinutes: 0,
			},
			wantError: true,
			errorMsg:  "Batch interval must be at least 1 minute",
		},
		{
			name: "batch interval too long",
			pref: &notification.NotificationPreference{
				UserID:               pulid.MustNew("user_"),
				OrganizationID:       pulid.MustNew("org_"),
				BusinessUnitID:       pulid.MustNew("bu_"),
				Resource:             permission.ResourceShipment,
				NotifyOnAllUpdates:   true,
				PreferredChannels:    []notification.Channel{notification.ChannelUser},
				BatchNotifications:   true,
				BatchIntervalMinutes: 1441, // > 24 hours
			},
			wantError: true,
			errorMsg:  "Batch interval cannot exceed 24 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := errors.NewMultiError()
			tt.pref.Validate(context.Background(), multiErr)

			if tt.wantError {
				assert.True(t, multiErr.HasErrors(), "Expected validation errors but got none")
				if tt.errorMsg != "" {
					assert.Contains(t, multiErr.Error(), tt.errorMsg)
				}
			} else {
				assert.False(t, multiErr.HasErrors(), "Expected no validation errors but got: %v", multiErr.Error())
			}
		})
	}
}

func TestNotificationPreference_IsUpdateTypeEnabled(t *testing.T) {
	tests := []struct {
		name       string
		pref       *notification.NotificationPreference
		updateType notification.UpdateType
		want       bool
	}{
		{
			name: "notify on all updates returns true",
			pref: &notification.NotificationPreference{
				NotifyOnAllUpdates: true,
			},
			updateType: notification.UpdateTypeStatusChange,
			want:       true,
		},
		{
			name: "specific update type enabled",
			pref: &notification.NotificationPreference{
				NotifyOnAllUpdates: false,
				UpdateTypes: []notification.UpdateType{
					notification.UpdateTypeStatusChange,
					notification.UpdateTypeAssignment,
				},
			},
			updateType: notification.UpdateTypeStatusChange,
			want:       true,
		},
		{
			name: "update type not enabled",
			pref: &notification.NotificationPreference{
				NotifyOnAllUpdates: false,
				UpdateTypes: []notification.UpdateType{
					notification.UpdateTypeStatusChange,
				},
			},
			updateType: notification.UpdateTypeAssignment,
			want:       false,
		},
		{
			name: "any update type always returns true",
			pref: &notification.NotificationPreference{
				NotifyOnAllUpdates: false,
				UpdateTypes: []notification.UpdateType{
					notification.UpdateTypeAny,
				},
			},
			updateType: notification.UpdateTypeStatusChange,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pref.IsUpdateTypeEnabled(tt.updateType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationPreference_ShouldNotifyUser(t *testing.T) {
	userID1 := pulid.MustNew("user_")
	userID2 := pulid.MustNew("user_")
	userID3 := pulid.MustNew("user_")

	tests := []struct {
		name            string
		pref            *notification.NotificationPreference
		updatedByUserID pulid.ID
		want            bool
	}{
		{
			name: "user not excluded",
			pref: &notification.NotificationPreference{
				ExcludedUserIDs: []pulid.ID{userID2, userID3},
			},
			updatedByUserID: userID1,
			want:            true,
		},
		{
			name: "user is excluded",
			pref: &notification.NotificationPreference{
				ExcludedUserIDs: []pulid.ID{userID1, userID2},
			},
			updatedByUserID: userID1,
			want:            false,
		},
		{
			name:            "no excluded users",
			pref:            &notification.NotificationPreference{},
			updatedByUserID: userID1,
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pref.ShouldNotifyUser(tt.updatedByUserID)
			assert.Equal(t, tt.want, got)
		})
	}
}
