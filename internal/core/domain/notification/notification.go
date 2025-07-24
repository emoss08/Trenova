/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notification

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"

	"github.com/emoss08/trenova/internal/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Notification)(nil)

type Targeting struct {
	Channel        Channel   `json:"channel"`
	OrganizationID pulid.ID  `json:"organizationId"`
	BusinessUnitID *pulid.ID `json:"businessUnitId,omitempty"`
	TargetUserID   *pulid.ID `json:"targetUserId,omitempty"`
	TargetRoleID   *pulid.ID `json:"targetRoleId,omitempty"`
}

type Action struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Type     string         `json:"type"`  // "button", "link", "form"
	Style    string         `json:"style"` // "primary", "secondary", "danger"
	Endpoint string         `json:"endpoint,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type RelatedEntity struct {
	Type string   `json:"type"` // "shipment", "worker", "customer"
	ID   pulid.ID `json:"id"`
	Name string   `json:"name,omitempty"`
	URL  string   `json:"url,omitempty"`
}

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:notif" json:"-"`

	ID              pulid.ID        `json:"id"                        bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID  pulid.ID        `json:"organizationId"            bun:"organization_id,type:VARCHAR(100),notnull"`
	EventType       EventType       `json:"eventType"                 bun:"event_type,type:VARCHAR(100),notnull"`
	Priority        Priority        `json:"priority"                  bun:"priority,type:VARCHAR(20),notnull"`
	Channel         Channel         `json:"channel"                   bun:"channel,type:VARCHAR(20),notnull"`
	DeliveryStatus  DeliveryStatus  `json:"deliveryStatus"            bun:"delivery_status,type:VARCHAR(20),notnull,default:'pending'"`
	Title           string          `json:"title"                     bun:"title,type:VARCHAR(255),notnull"`
	Message         string          `json:"message"                   bun:"message,type:TEXT,notnull"`
	Source          string          `json:"source"                    bun:"source,type:VARCHAR(100),notnull"`
	Data            map[string]any  `json:"data,omitempty"            bun:"data,type:JSONB"`
	Tags            []string        `json:"tags,omitempty"            bun:"tags,type:TEXT[]"`
	RelatedEntities []RelatedEntity `json:"relatedEntities,omitempty" bun:"related_entities,type:JSONB"`
	Actions         []Action        `json:"actions,omitempty"         bun:"actions,type:JSONB"`
	CreatedAt       int64           `json:"createdAt"                 bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64           `json:"updatedAt"                 bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	RetryCount      int             `json:"retryCount"                bun:"retry_count,type:INT,notnull,default:0"`
	MaxRetries      int             `json:"maxRetries"                bun:"max_retries,type:INT,notnull,default:3"`
	Version         int64           `json:"version"                   bun:"version,type:BIGINT,notnull,default:0"`
	ExpiresAt       *int64          `json:"expiresAt,omitempty"       bun:"expires_at"`
	DeliveredAt     *int64          `json:"deliveredAt,omitempty"     bun:"delivered_at"`
	ReadAt          *int64          `json:"readAt,omitempty"          bun:"read_at"`
	DismissedAt     *int64          `json:"dismissedAt,omitempty"     bun:"dismissed_at"`
	JobID           *string         `json:"jobId,omitempty"           bun:"job_id,type:VARCHAR(255)"`
	CorrelationID   *string         `json:"correlationId,omitempty"   bun:"correlation_id,type:VARCHAR(255)"`
	BusinessUnitID  *pulid.ID       `json:"businessUnitId,omitempty"  bun:"business_unit_id,type:VARCHAR(100)"`
	TargetUserID    *pulid.ID       `json:"targetUserId,omitempty"    bun:"target_user_id,type:VARCHAR(100)"`
	TargetRoleID    *pulid.ID       `json:"targetRoleId,omitempty"    bun:"target_role_id,type:VARCHAR(100)"`
}

func (n *Notification) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(
		ctx,
		n,
		validation.Field(&n.ID, validation.Required.Error("ID is required")),
		validation.Field(
			&n.OrganizationID,
			validation.Required.Error("OrganizationID is required"),
		),
		validation.Field(&n.EventType, validation.Required.Error("EventType is required")),
		validation.Field(&n.Priority, validation.Required.Error("Priority is required")),
		validation.Field(&n.Channel, validation.Required.Error("Channel is required")),
		validation.Field(&n.Title, validation.Required.Error("Title is required")),
		validation.Field(&n.Message, validation.Required.Error("Message is required")),
		validation.Field(&n.Source, validation.Required.Error("Source is required")),
		validation.Field(&n.Tags, validation.By(domain.ValidateStringSlice)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (n *Notification) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if n.ID.IsNil() {
			n.ID = pulid.MustNew("notif_")
		}
		n.CreatedAt = now
	case *bun.UpdateQuery:
		n.UpdatedAt = now
	}

	return nil
}

// GetTargeting returns the targeting information for the notification
func (n *Notification) GetTargeting() Targeting {
	return Targeting{
		Channel:        n.Channel,
		OrganizationID: n.OrganizationID,
		BusinessUnitID: n.BusinessUnitID,
		TargetUserID:   n.TargetUserID,
		TargetRoleID:   n.TargetRoleID,
	}
}

// IsExpired checks if the notification has expired
func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return timeutils.NowUnix() > *n.ExpiresAt
}

// IsDelivered checks if the notification has been delivered
func (n *Notification) IsDelivered() bool {
	return n.DeliveryStatus == DeliveryStatusDelivered
}

// IsRead checks if the notification has been read
func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}

// IsDismissed checks if the notification has been dismissed
func (n *Notification) IsDismissed() bool {
	return n.DismissedAt != nil
}

// CanRetry checks if the notification can be retried
func (n *Notification) CanRetry() bool {
	return n.RetryCount < n.MaxRetries && n.DeliveryStatus == DeliveryStatusFailed
}

// GenerateRoomName generates the WebSocket room name based on targeting
func (n *Notification) GenerateRoomName() string {
	targeting := n.GetTargeting()
	return GenerateRoomName(targeting)
}

func (n *Notification) GetID() pulid.ID {
	return n.ID
}

func (n *Notification) GetEventType() EventType {
	return n.EventType
}
