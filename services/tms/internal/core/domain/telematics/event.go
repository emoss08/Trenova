package telematics

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type TelematicsEvent struct {
	bun.BaseModel `bun:"table:telematics_events,alias:tlev" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider       string         `json:"provider"       bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	EventID        string         `json:"eventId"        bun:"event_id,type:TEXT,notnull"`
	EventType      EventType      `json:"eventType"      bun:"event_type,type:VARCHAR(64),notnull"`
	OccurredAt     int64          `json:"occurredAt"     bun:"occurred_at,type:BIGINT,notnull"`
	TractorID      pulid.ID       `json:"tractorId"      bun:"tractor_id,type:VARCHAR(100),nullzero"`
	WorkerID       pulid.ID       `json:"workerId"       bun:"worker_id,type:VARCHAR(100),nullzero"`
	LocationID     pulid.ID       `json:"locationId"     bun:"location_id,type:VARCHAR(100),nullzero"`
	AddressName    string         `json:"addressName"    bun:"address_name,type:TEXT,nullzero"`
	Payload        map[string]any `json:"payload"        bun:"payload,type:JSONB,nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
}

func NewEventID() pulid.ID {
	return pulid.MustNew("tlev_")
}
