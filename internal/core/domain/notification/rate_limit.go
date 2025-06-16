package notification

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*NotificationRateLimit)(nil)
	_ domain.Validatable        = (*NotificationRateLimit)(nil)
)

// RateLimitPeriod represents the time period for rate limiting
type RateLimitPeriod string

const (
	RateLimitPeriodMinute = RateLimitPeriod("minute")
	RateLimitPeriodHour   = RateLimitPeriod("hour")
	RateLimitPeriodDay    = RateLimitPeriod("day")
)

// NotificationRateLimit represents rate limiting rules for notifications
type NotificationRateLimit struct {
	bun.BaseModel `bun:"table:notification_rate_limits,alias:nrl" json:"-"`

	// Core identification
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`

	// Rule configuration
	Name        string              `json:"name"         bun:"name,type:VARCHAR(100),notnull"`
	Description string              `json:"description"  bun:"description,type:TEXT"`
	Resource    permission.Resource `json:"resource"     bun:"resource,type:VARCHAR(50)"`
	EventType   EventType           `json:"eventType"    bun:"event_type,type:VARCHAR(50)"`
	Priority    Priority            `json:"priority"     bun:"priority,type:VARCHAR(20)"`

	// Rate limit settings
	MaxNotifications int             `json:"maxNotifications" bun:"max_notifications,type:INT,notnull"`
	Period           RateLimitPeriod `json:"period"           bun:"period,type:VARCHAR(20),notnull"`

	// Scope
	ApplyToAllUsers bool     `json:"applyToAllUsers" bun:"apply_to_all_users,type:BOOLEAN,notnull,default:true"`
	UserID          pulid.ID `json:"userId"          bun:"user_id,type:VARCHAR(100)"`
	RoleID          pulid.ID `json:"roleId"          bun:"role_id,type:VARCHAR(100)"`

	// Status
	IsActive  bool  `json:"isActive"  bun:"is_active,type:BOOLEAN,notnull,default:true"`
	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (r *NotificationRateLimit) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("nrl_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
		r.Version++
	}

	return nil
}

// Validate validates the notification rate limit
func (r *NotificationRateLimit) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, r,
		// OrganizationID is required
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),

		// Name is required
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// MaxNotifications must be positive
		validation.Field(&r.MaxNotifications,
			validation.By(func(value interface{}) error {
				v := value.(int)
				if v < 1 {
					return validation.NewError("validation_min", "Maximum notifications must be at least 1")
				}
				return nil
			}),
		),

		// Period is required
		validation.Field(&r.Period,
			validation.Required.Error("Period is required"),
			validation.In(
				RateLimitPeriodMinute,
				RateLimitPeriodHour,
				RateLimitPeriodDay,
			).Error("Invalid rate limit period"),
		),

		// If not applying to all users, must specify user or role
		validation.Field(&r.UserID,
			validation.When(
				!r.ApplyToAllUsers && r.RoleID.IsNil(),
				validation.Required.Error("User ID is required when not applying to all users and no role is specified"),
			),
		),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID of the notification rate limit
func (r *NotificationRateLimit) GetID() pulid.ID {
	return r.ID
}

// GetTableName returns the table name for the notification rate limit
func (r *NotificationRateLimit) GetTableName() string {
	return "notification_rate_limits"
}

// GetPeriodDuration returns the duration of the rate limit period
func (r *NotificationRateLimit) GetPeriodDuration() time.Duration {
	switch r.Period {
	case RateLimitPeriodMinute:
		return time.Minute
	case RateLimitPeriodHour:
		return time.Hour
	case RateLimitPeriodDay:
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

// AppliesTo checks if the rate limit applies to a specific user
func (r *NotificationRateLimit) AppliesTo(userID pulid.ID, userRoleIDs []pulid.ID) bool {
	if r.ApplyToAllUsers {
		return true
	}

	if !r.UserID.IsNil() && r.UserID == userID {
		return true
	}

	if !r.RoleID.IsNil() {
		for _, roleID := range userRoleIDs {
			if r.RoleID == roleID {
				return true
			}
		}
	}

	return false
}

// NotificationRateLimitCounter tracks the current count for a rate limit
type NotificationRateLimitCounter struct {
	UserID      pulid.ID            `json:"userId"`
	RateLimitID pulid.ID            `json:"rateLimitId"`
	Resource    permission.Resource `json:"resource"`
	EventType   EventType           `json:"eventType"`
	Period      RateLimitPeriod     `json:"period"`
	Count       int                 `json:"count"`
	WindowStart int64               `json:"windowStart"`
	WindowEnd   int64               `json:"windowEnd"`
}

// IsExceeded checks if the rate limit has been exceeded
func (c *NotificationRateLimitCounter) IsExceeded(maxNotifications int) bool {
	return c.Count >= maxNotifications
}

// ShouldReset checks if the counter window should be reset
func (c *NotificationRateLimitCounter) ShouldReset() bool {
	now := timeutils.NowUnix()
	return now >= c.WindowEnd
}

// Reset resets the counter for a new window
func (c *NotificationRateLimitCounter) Reset(period RateLimitPeriod) {
	now := timeutils.NowUnix()
	c.Count = 0
	c.WindowStart = now

	switch period {
	case RateLimitPeriodMinute:
		c.WindowEnd = now + 60
	case RateLimitPeriodHour:
		c.WindowEnd = now + 3600
	case RateLimitPeriodDay:
		c.WindowEnd = now + 86400
	}
}

// Increment increments the counter
func (c *NotificationRateLimitCounter) Increment() {
	c.Count++
}
