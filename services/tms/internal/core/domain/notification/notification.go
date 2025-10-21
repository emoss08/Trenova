package notification

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Notification)(nil)

var _ domain.Validatable = (*Notification)(nil)

type Targeting struct {
	Channel        Channel   `json:"channel"`
	OrganizationID pulid.ID  `json:"organizationId"`
	BusinessUnitID *pulid.ID `json:"businessUnitId,omitempty"`
	TargetUserID   *pulid.ID `json:"targetUserId,omitempty"`
}

type Action struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Type     string         `json:"type"`
	Style    string         `json:"style"`
	Endpoint string         `json:"endpoint,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type RelatedEntity struct {
	Type string   `json:"type"`
	ID   pulid.ID `json:"id"`
	Name string   `json:"name,omitempty"`
	URL  string   `json:"url,omitempty"`
}

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:notif" json:"-"`

	ID              pulid.ID        `json:"id"                        bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID  pulid.ID        `json:"organizationId"            bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  *pulid.ID       `json:"businessUnitId,omitempty"  bun:"business_unit_id,type:VARCHAR(100)"`
	TargetUserID    *pulid.ID       `json:"targetUserId,omitempty"    bun:"target_user_id,type:VARCHAR(100)"`
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

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	TargetUser   *tenant.User         `json:"targetUser,omitempty"   bun:"rel:belongs-to,join:target_user_id=id"`
}

func (n *Notification) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
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
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (n *Notification) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

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

func (n *Notification) GetTargeting() Targeting {
	return Targeting{
		Channel:        n.Channel,
		OrganizationID: n.OrganizationID,
		BusinessUnitID: n.BusinessUnitID,
		TargetUserID:   n.TargetUserID,
	}
}

func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return utils.NowUnix() > *n.ExpiresAt
}

func (n *Notification) IsDelivered() bool {
	return n.DeliveryStatus == DeliveryStatusDelivered
}

func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}

func (n *Notification) IsDismissed() bool {
	return n.DismissedAt != nil
}

func (n *Notification) CanRetry() bool {
	return n.RetryCount < n.MaxRetries && n.DeliveryStatus == DeliveryStatusFailed
}

func (n *Notification) GenerateRoomName() string {
	targeting := n.GetTargeting()
	return GenerateRoomName(targeting)
}

func (n *Notification) GetID() pulid.ID {
	return n.ID
}

func (n *Notification) GetTableName() string {
	return "notifications"
}

func (n *Notification) GetEventType() EventType {
	return n.EventType
}
