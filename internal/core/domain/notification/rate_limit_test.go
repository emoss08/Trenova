package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
)

func TestNotificationRateLimit_Validate(t *testing.T) {
	tests := []struct {
		name      string
		limit     *notification.NotificationRateLimit
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid rate limit for all users",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				Name:             "Shipment Update Limit",
				Description:      "Limit shipment update notifications",
				Resource:         permission.ResourceShipment,
				MaxNotifications: 10,
				Period:           notification.RateLimitPeriodHour,
				ApplyToAllUsers:  true,
			},
			wantError: false,
		},
		{
			name: "valid rate limit for specific user",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				Name:             "User Limit",
				MaxNotifications: 5,
				Period:           notification.RateLimitPeriodMinute,
				ApplyToAllUsers:  false,
				UserID:           pulid.MustNew("user_"),
			},
			wantError: false,
		},
		{
			name: "missing organization ID",
			limit: &notification.NotificationRateLimit{
				Name:             "Test Limit",
				MaxNotifications: 10,
				Period:           notification.RateLimitPeriodHour,
				ApplyToAllUsers:  true,
			},
			wantError: true,
			errorMsg:  "Organization ID is required",
		},
		{
			name: "missing name",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				MaxNotifications: 10,
				Period:           notification.RateLimitPeriodHour,
				ApplyToAllUsers:  true,
			},
			wantError: true,
			errorMsg:  "Name is required",
		},
		{
			name: "invalid max notifications",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				Name:             "Test Limit",
				MaxNotifications: 0,
				Period:           notification.RateLimitPeriodHour,
				ApplyToAllUsers:  true,
			},
			wantError: true,
			errorMsg:  "Maximum notifications must be at least 1",
		},
		{
			name: "invalid period",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				Name:             "Test Limit",
				MaxNotifications: 10,
				Period:           notification.RateLimitPeriod("week"),
				ApplyToAllUsers:  true,
			},
			wantError: true,
			errorMsg:  "Invalid rate limit period",
		},
		{
			name: "not applying to all but no user or role",
			limit: &notification.NotificationRateLimit{
				OrganizationID:   pulid.MustNew("org_"),
				Name:             "Test Limit",
				MaxNotifications: 10,
				Period:           notification.RateLimitPeriodHour,
				ApplyToAllUsers:  false,
			},
			wantError: true,
			errorMsg:  "User ID is required when not applying to all users and no role is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := errors.NewMultiError()
			tt.limit.Validate(context.Background(), multiErr)

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

func TestNotificationRateLimit_GetPeriodDuration(t *testing.T) {
	tests := []struct {
		name   string
		period notification.RateLimitPeriod
		want   time.Duration
	}{
		{
			name:   "minute period",
			period: notification.RateLimitPeriodMinute,
			want:   time.Minute,
		},
		{
			name:   "hour period",
			period: notification.RateLimitPeriodHour,
			want:   time.Hour,
		},
		{
			name:   "day period",
			period: notification.RateLimitPeriodDay,
			want:   24 * time.Hour,
		},
		{
			name:   "unknown period defaults to hour",
			period: notification.RateLimitPeriod("unknown"),
			want:   time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := &notification.NotificationRateLimit{
				Period: tt.period,
			}
			got := limit.GetPeriodDuration()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationRateLimit_AppliesTo(t *testing.T) {
	userID := pulid.MustNew("user_")
	anotherUserID := pulid.MustNew("user_")
	roleID := pulid.MustNew("role_")
	anotherRoleID := pulid.MustNew("role_")

	tests := []struct {
		name        string
		limit       *notification.NotificationRateLimit
		userID      pulid.ID
		userRoleIDs []pulid.ID
		want        bool
	}{
		{
			name: "applies to all users",
			limit: &notification.NotificationRateLimit{
				ApplyToAllUsers: true,
			},
			userID:      userID,
			userRoleIDs: []pulid.ID{roleID},
			want:        true,
		},
		{
			name: "applies to specific user",
			limit: &notification.NotificationRateLimit{
				ApplyToAllUsers: false,
				UserID:          userID,
			},
			userID:      userID,
			userRoleIDs: []pulid.ID{roleID},
			want:        true,
		},
		{
			name: "does not apply to different user",
			limit: &notification.NotificationRateLimit{
				ApplyToAllUsers: false,
				UserID:          userID,
			},
			userID:      anotherUserID,
			userRoleIDs: []pulid.ID{roleID},
			want:        false,
		},
		{
			name: "applies to user with role",
			limit: &notification.NotificationRateLimit{
				ApplyToAllUsers: false,
				RoleID:          roleID,
			},
			userID:      userID,
			userRoleIDs: []pulid.ID{roleID, anotherRoleID},
			want:        true,
		},
		{
			name: "does not apply to user without role",
			limit: &notification.NotificationRateLimit{
				ApplyToAllUsers: false,
				RoleID:          roleID,
			},
			userID:      userID,
			userRoleIDs: []pulid.ID{anotherRoleID},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.limit.AppliesTo(tt.userID, tt.userRoleIDs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationRateLimitCounter_IsExceeded(t *testing.T) {
	tests := []struct {
		name             string
		counter          *notification.NotificationRateLimitCounter
		maxNotifications int
		want             bool
	}{
		{
			name: "not exceeded",
			counter: &notification.NotificationRateLimitCounter{
				Count: 5,
			},
			maxNotifications: 10,
			want:             false,
		},
		{
			name: "exactly at limit",
			counter: &notification.NotificationRateLimitCounter{
				Count: 10,
			},
			maxNotifications: 10,
			want:             true,
		},
		{
			name: "exceeded",
			counter: &notification.NotificationRateLimitCounter{
				Count: 15,
			},
			maxNotifications: 10,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counter.IsExceeded(tt.maxNotifications)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationRateLimitCounter_ShouldReset(t *testing.T) {
	now := timeutils.NowUnix()

	tests := []struct {
		name    string
		counter *notification.NotificationRateLimitCounter
		want    bool
	}{
		{
			name: "window not ended",
			counter: &notification.NotificationRateLimitCounter{
				WindowEnd: now + 3600, // 1 hour from now
			},
			want: false,
		},
		{
			name: "window exactly ended",
			counter: &notification.NotificationRateLimitCounter{
				WindowEnd: now,
			},
			want: true,
		},
		{
			name: "window passed",
			counter: &notification.NotificationRateLimitCounter{
				WindowEnd: now - 3600, // 1 hour ago
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counter.ShouldReset()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationRateLimitCounter_Reset(t *testing.T) {
	tests := []struct {
		name               string
		period             notification.RateLimitPeriod
		expectedWindowSize int64
	}{
		{
			name:               "minute period",
			period:             notification.RateLimitPeriodMinute,
			expectedWindowSize: 60,
		},
		{
			name:               "hour period",
			period:             notification.RateLimitPeriodHour,
			expectedWindowSize: 3600,
		},
		{
			name:               "day period",
			period:             notification.RateLimitPeriodDay,
			expectedWindowSize: 86400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &notification.NotificationRateLimitCounter{
				Count: 10,
			}

			beforeReset := timeutils.NowUnix()
			counter.Reset(tt.period)
			afterReset := timeutils.NowUnix()

			// Check count is reset
			assert.Equal(t, 0, counter.Count)

			// Check window start is set
			assert.GreaterOrEqual(t, counter.WindowStart, beforeReset)
			assert.LessOrEqual(t, counter.WindowStart, afterReset)

			// Check window end is set correctly
			expectedEnd := counter.WindowStart + tt.expectedWindowSize
			assert.Equal(t, expectedEnd, counter.WindowEnd)
		})
	}
}

func TestNotificationRateLimitCounter_Increment(t *testing.T) {
	counter := &notification.NotificationRateLimitCounter{
		Count: 5,
	}

	counter.Increment()
	assert.Equal(t, 6, counter.Count)

	counter.Increment()
	assert.Equal(t, 7, counter.Count)
}
