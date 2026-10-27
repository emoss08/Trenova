package notification

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable = (*Notification)(nil)
)

type Notification struct {
	keel.Table `keel:"notifications"`

	ID              pulid.ID       `json:"id"              keel:"id,pk,autoid:notif_"`
	OrganizationID  pulid.ID       `json:"organizationId"  keel:"organization_id,tenant"`
	BusinessUnitID  *pulid.ID      `json:"businessUnitId"  keel:"business_unit_id"`
	TargetUserID    *pulid.ID      `json:"targetUserId"    keel:"target_user_id"`
	EventType       string         `json:"eventType"       keel:"event_type"`
	Priority        Priority       `json:"priority"        keel:"priority"`
	Channel         Channel        `json:"channel"         keel:"channel"`
	Title           string         `json:"title"           keel:"title"`
	Message         string         `json:"message"         keel:"message"`
	Data            map[string]any `json:"data"            keel:"data,jsonb"`
	RelatedEntities map[string]any `json:"relatedEntities" keel:"related_entities,jsonb"`
	Actions         map[string]any `json:"actions"         keel:"actions,jsonb"`
	ExpiresAt       *int64         `json:"expiresAt"       keel:"expires_at"`
	DeliveredAt     *int64         `json:"deliveredAt"     keel:"delivered_at"`
	ReadAt          *int64         `json:"readAt"          keel:"read_at"`
	DismissedAt     *int64         `json:"dismissedAt"     keel:"dismissed_at"`
	CreatedAt       int64          `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64          `json:"updatedAt"       keel:"updated_at,autoupdate"`
	DeliveryStatus  DeliveryStatus `json:"deliveryStatus"  keel:"delivery_status"`
	RetryCount      int            `json:"retryCount"      keel:"retry_count"`
	MaxRetries      int            `json:"maxRetries"      keel:"max_retries"`
	Source          string         `json:"source"          keel:"source"`
	JobID           *string        `json:"jobId"           keel:"job_id"`
	CorrelationID   *string        `json:"correlationId"   keel:"correlation_id"`
	Tags            []string       `json:"tags"            keel:"tags"`
	Version         int64          `json:"version"         keel:"version,version"`
}

func (n *Notification) BeforeCreate(_ context.Context) error {
	if n.Priority == "" {
		n.Priority = PriorityMedium
	}
	if n.Channel == "" {
		n.Channel = ChannelGlobal
	}
	if n.DeliveryStatus == "" {
		n.DeliveryStatus = DeliveryStatusPending
	}
	if n.MaxRetries == 0 {
		n.MaxRetries = 3
	}

	return nil
}

func (n *Notification) GetID() pulid.ID {
	return n.ID
}

func (n *Notification) GetCreatedAt() int64 {
	return n.CreatedAt
}

func (n *Notification) GetTableName() string {
	return "notifications"
}

func (n *Notification) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "notif",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "message", Type: domaintypes.FieldTypeText},
		},
	}
}

func (n *Notification) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(n,
		validation.Field(&n.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&n.EventType,
			validation.Required.Error("Event type is required"),
			validation.Length(1, maxEventTypeLength).
				Error("Event type cannot be longer than 100 characters"),
		),
		validation.Field(&n.Priority,
			validation.Required.Error("Priority is required"),
			domainvalidation.ValidEnum[Priority]("Priority is invalid"),
		),
		validation.Field(&n.Channel,
			validation.Required.Error("Channel is required"),
			domainvalidation.ValidEnum[Channel]("Channel is invalid"),
		),
		// A user-channel notification with no recipient would be delivered to
		// nobody while counting as sent.
		validation.Field(&n.TargetUserID,
			validation.When(
				n.Channel == ChannelUser,
				validation.Required.Error("A user notification must name its recipient"),
			),
		),
		validation.Field(&n.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, maxNotificationTitleLength).
				Error("Title cannot be longer than 255 characters"),
		),
		validation.Field(&n.Message, validation.Required.Error("Message is required")),
		validation.Field(&n.DeliveryStatus,
			validation.Required.Error("Delivery status is required"),
			domainvalidation.ValidEnum[DeliveryStatus]("Delivery status is invalid"),
		),
		validation.Field(&n.Source,
			validation.Required.Error("Source is required"),
			validation.Length(1, maxNotificationSourceLength).
				Error("Source cannot be longer than 100 characters"),
		),
		validation.Field(&n.RetryCount,
			validation.Min(0).Error("Retry count cannot be negative"),
		),
		// Allowing more retries than the counter can reach would leave a failed
		// notification retrying forever.
		validation.Field(&n.MaxRetries,
			validation.Min(0).Error("Maximum retries cannot be negative"),
			validation.Max(maxNotificationRetries).
				Error("Maximum retries cannot exceed 10"),
		),
	))
}

const (
	maxEventTypeLength          = 100
	maxNotificationTitleLength  = 255
	maxNotificationSourceLength = 100
	maxNotificationRetries      = 10
)
