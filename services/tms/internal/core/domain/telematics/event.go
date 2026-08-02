package telematics

import (
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

func (e *TelematicsEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		// The provider's own id is what makes a redelivered event recognizable
		// as one already recorded rather than a second occurrence.
		validation.Field(&e.EventID, validation.Required.Error("Event id is required")),
		validation.Field(&e.EventType,
			validation.Required.Error("Event type is required"),
			domainvalidation.ValidEnum[EventType]("Event type is invalid"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a valid timestamp"),
		),
	))
}

const maxProviderLength = 32
