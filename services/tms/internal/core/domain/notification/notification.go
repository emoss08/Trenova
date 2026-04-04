package notification

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Notification)(nil)
	_ domaintypes.PostgresSearchable = (*Notification)(nil)
)

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:notif" json:"-"`

	ID              pulid.ID       `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID  pulid.ID       `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  *pulid.ID      `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),nullzero"`
	TargetUserID    *pulid.ID      `json:"targetUserId"    bun:"target_user_id,type:VARCHAR(100),nullzero"`
	EventType       string         `json:"eventType"       bun:"event_type,type:VARCHAR(100),notnull"`
	Priority        Priority       `json:"priority"        bun:"priority,type:VARCHAR(20),notnull,default:'medium'"`
	Channel         Channel        `json:"channel"         bun:"channel,type:VARCHAR(20),notnull,default:'global'"`
	Title           string         `json:"title"           bun:"title,type:VARCHAR(255),notnull"`
	Message         string         `json:"message"         bun:"message,type:TEXT,notnull"`
	Data            map[string]any `json:"data"            bun:"data,type:JSONB"`
	RelatedEntities map[string]any `json:"relatedEntities" bun:"related_entities,type:JSONB"`
	Actions         map[string]any `json:"actions"         bun:"actions,type:JSONB"`
	ExpiresAt       *int64         `json:"expiresAt"       bun:"expires_at,type:BIGINT"`
	DeliveredAt     *int64         `json:"deliveredAt"     bun:"delivered_at,type:BIGINT"`
	ReadAt          *int64         `json:"readAt"          bun:"read_at,type:BIGINT"`
	DismissedAt     *int64         `json:"dismissedAt"     bun:"dismissed_at,type:BIGINT"`
	CreatedAt       int64          `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64          `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	DeliveryStatus  DeliveryStatus `json:"deliveryStatus"  bun:"delivery_status,type:VARCHAR(20),notnull,default:'pending'"`
	RetryCount      int            `json:"retryCount"      bun:"retry_count,type:INT,notnull,default:0"`
	MaxRetries      int            `json:"maxRetries"      bun:"max_retries,type:INT,notnull,default:3"`
	Source          string         `json:"source"          bun:"source,type:VARCHAR(100),notnull"`
	JobID           *string        `json:"jobId"           bun:"job_id,type:VARCHAR(255)"`
	CorrelationID   *string        `json:"correlationId"   bun:"correlation_id,type:VARCHAR(255)"`
	Tags            []string       `json:"tags"            bun:"tags,type:text[],array"`
	Version         int64          `json:"version"         bun:"version,type:BIGINT,notnull,default:0"`
}

func (n *Notification) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if n.ID.IsNil() {
			n.ID = pulid.MustNew("notif_")
		}
		n.CreatedAt = now
		n.UpdatedAt = now
	case *bun.UpdateQuery:
		n.UpdatedAt = now
	}

	return nil
}

func (n *Notification) GetID() pulid.ID {
	return n.ID
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
