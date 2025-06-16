package notification

import (
	"context"

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
	_ bun.BeforeAppendModelHook = (*NotificationHistory)(nil)
	_ domain.Validatable        = (*NotificationHistory)(nil)
)

// NotificationHistory represents a record of a sent notification
type NotificationHistory struct {
	bun.BaseModel `bun:"table:notification_history,alias:nh" json:"-"`

	// Core identification
	ID             pulid.ID `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	NotificationID pulid.ID `json:"notificationId"     bun:"notification_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"             bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull"`

	// Entity reference
	EntityType  permission.Resource `json:"entityType"   bun:"entity_type,type:VARCHAR(50)"`
	EntityID    pulid.ID            `json:"entityId"     bun:"entity_id,type:VARCHAR(100)"`
	UpdateType  UpdateType          `json:"updateType"   bun:"update_type,type:VARCHAR(50)"`
	UpdatedByID pulid.ID            `json:"updatedById"  bun:"updated_by_id,type:VARCHAR(100)"`

	// Notification details
	Title     string         `json:"title"       bun:"title,type:VARCHAR(255),notnull"`
	Message   string         `json:"message"     bun:"message,type:TEXT,notnull"`
	Priority  Priority       `json:"priority"    bun:"priority,type:VARCHAR(20),notnull"`
	Channel   Channel        `json:"channel"     bun:"channel,type:VARCHAR(20),notnull"`
	EventType EventType      `json:"eventType"   bun:"event_type,type:VARCHAR(50),notnull"`
	Data      map[string]any `json:"data"        bun:"data,type:JSONB"`

	// Delivery information
	DeliveryStatus DeliveryStatus `json:"deliveryStatus" bun:"delivery_status,type:VARCHAR(20),notnull"`
	DeliveredAt    *int64         `json:"deliveredAt"    bun:"delivered_at"`
	FailureReason  string         `json:"failureReason"  bun:"failure_reason,type:TEXT"`
	RetryCount     int            `json:"retryCount"     bun:"retry_count,type:INT,notnull,default:0"`

	// User interaction
	ReadAt      *int64 `json:"readAt"        bun:"read_at"`
	DismissedAt *int64 `json:"dismissedAt"   bun:"dismissed_at"`
	ClickedAt   *int64 `json:"clickedAt"     bun:"clicked_at"`
	ActionTaken string `json:"actionTaken"   bun:"action_taken,type:VARCHAR(100)"`

	// Grouping
	GroupID       *pulid.ID `json:"groupId"       bun:"group_id,type:VARCHAR(100)"`
	GroupPosition int       `json:"groupPosition" bun:"group_position,type:INT"`

	// Timestamps
	CreatedAt int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	ExpiresAt *int64 `json:"expiresAt" bun:"expires_at"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (h *NotificationHistory) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if h.ID.IsNil() {
			h.ID = pulid.MustNew("nhist_")
		}
		h.CreatedAt = now

		// Set delivery status to pending if not set
		if h.DeliveryStatus == "" {
			h.DeliveryStatus = DeliveryStatusPending
		}
	}

	return nil
}

// Validate validates the notification history
func (h *NotificationHistory) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, h,
		// NotificationID is required
		validation.Field(&h.NotificationID,
			validation.Required.Error("Notification ID is required"),
		),

		// UserID is required
		validation.Field(&h.UserID,
			validation.Required.Error("User ID is required"),
		),

		// OrganizationID is required
		validation.Field(&h.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),

		// Title is required
		validation.Field(&h.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, 255).Error("Title must be between 1 and 255 characters"),
		),

		// Message is required
		validation.Field(&h.Message,
			validation.Required.Error("Message is required"),
		),

		// Priority is required
		validation.Field(&h.Priority,
			validation.Required.Error("Priority is required"),
			validation.In(
				PriorityCritical,
				PriorityHigh,
				PriorityMedium,
				PriorityLow,
			).Error("Invalid priority level"),
		),

		// Channel is required
		validation.Field(&h.Channel,
			validation.Required.Error("Channel is required"),
			validation.In(
				ChannelGlobal,
				ChannelUser,
				ChannelRole,
			).Error("Invalid channel type"),
		),

		// EventType is required
		validation.Field(&h.EventType,
			validation.Required.Error("Event type is required"),
		),

		// DeliveryStatus is required
		validation.Field(&h.DeliveryStatus,
			validation.Required.Error("Delivery status is required"),
			validation.In(
				DeliveryStatusPending,
				DeliveryStatusDelivered,
				DeliveryStatusFailed,
				DeliveryStatusExpired,
			).Error("Invalid delivery status"),
		),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID of the notification history
func (h *NotificationHistory) GetID() pulid.ID {
	return h.ID
}

// GetTableName returns the table name for the notification history
func (h *NotificationHistory) GetTableName() string {
	return "notification_history"
}

// IsRead returns true if the notification has been read
func (h *NotificationHistory) IsRead() bool {
	return h.ReadAt != nil
}

// IsDismissed returns true if the notification has been dismissed
func (h *NotificationHistory) IsDismissed() bool {
	return h.DismissedAt != nil
}

// IsExpired returns true if the notification has expired
func (h *NotificationHistory) IsExpired() bool {
	if h.ExpiresAt == nil {
		return false
	}
	return *h.ExpiresAt < timeutils.NowUnix()
}

// MarkAsRead marks the notification as read
func (h *NotificationHistory) MarkAsRead() {
	now := timeutils.NowUnix()
	h.ReadAt = &now
}

// MarkAsDismissed marks the notification as dismissed
func (h *NotificationHistory) MarkAsDismissed() {
	now := timeutils.NowUnix()
	h.DismissedAt = &now
}

// MarkAsClicked marks the notification as clicked
func (h *NotificationHistory) MarkAsClicked() {
	now := timeutils.NowUnix()
	h.ClickedAt = &now
}

// SetDelivered marks the notification as delivered
func (h *NotificationHistory) SetDelivered() {
	now := timeutils.NowUnix()
	h.DeliveryStatus = DeliveryStatusDelivered
	h.DeliveredAt = &now
}

// SetFailed marks the notification as failed
func (h *NotificationHistory) SetFailed(reason string) {
	h.DeliveryStatus = DeliveryStatusFailed
	h.FailureReason = reason
	h.RetryCount++
}
