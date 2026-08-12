package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CostEvent)(nil)
	_ pagination.CursorEntity            = (*CostEvent)(nil)
	_ validationframework.TenantedEntity = (*CostEvent)(nil)
)

type CostEvent struct {
	bun.BaseModel             `bun:"table:carrier_cost_events,alias:cacc" json:"-"`
	pagination.CursorValueSet `bun:",embed"                               json:"-"`

	ID                  pulid.ID        `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID        `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	CarrierID           pulid.ID        `json:"carrierId"           bun:"carrier_id,type:VARCHAR(100),notnull"`
	CarrierAssignmentID *pulid.ID       `json:"carrierAssignmentId" bun:"carrier_assignment_id,type:VARCHAR(100),nullzero"`
	ShipmentID          *pulid.ID       `json:"shipmentId"          bun:"shipment_id,type:VARCHAR(100),nullzero"`
	MoveID              *pulid.ID       `json:"moveId"              bun:"move_id,type:VARCHAR(100),nullzero"`
	SettlementID        *pulid.ID       `json:"settlementId"        bun:"settlement_id,type:VARCHAR(100),nullzero"`
	EventType           CostEventType   `json:"eventType"           bun:"event_type,type:VARCHAR(50),notnull"`
	Status              CostEventStatus `json:"status"              bun:"status,type:VARCHAR(50),notnull,default:'Pending'"`
	IdempotencyKey      string          `json:"idempotencyKey"      bun:"idempotency_key,type:VARCHAR(255),notnull"`
	EventDate           int64           `json:"eventDate"           bun:"event_date,type:BIGINT,notnull"`
	AmountMinor         int64           `json:"amountMinor"         bun:"amount_minor,type:BIGINT,notnull,default:0"`
	CurrencyCode        string          `json:"currencyCode"        bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Description         string          `json:"description"         bun:"description,type:VARCHAR(255),nullzero"`
	ProNumber           string          `json:"proNumber"           bun:"pro_number,type:VARCHAR(100),nullzero"`
	AssignmentVersion   int64           `json:"assignmentVersion"   bun:"assignment_version,type:BIGINT,notnull,default:0"`
	VoidedAt            *int64          `json:"voidedAt"            bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason          string          `json:"voidReason"          bun:"void_reason,type:TEXT,nullzero"`
	Version             int64           `json:"version"             bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64           `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Carrier *carrier.Carrier `json:"carrier,omitempty" bun:"rel:belongs-to,join:carrier_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (c *CostEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(
			&c.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
		),
		validation.Field(&c.EventDate, validation.Required.Error("Event date is required")),
	))

	if !c.EventType.IsValid() {
		multiErr.Add("eventType", errortypes.ErrInvalid, "Cost event type is invalid")
	}
	if !c.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Cost event status is invalid")
	}
	if c.EventType != CostEventTypeAdjustment && c.AmountMinor < 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Only adjustment cost events may carry a negative amount",
		)
	}
}

func (c *CostEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "cacc",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (c *CostEvent) GetID() pulid.ID { return c.ID }

func (c *CostEvent) GetCreatedAt() int64 { return c.CreatedAt }

func (c *CostEvent) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *CostEvent) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *CostEvent) GetTableName() string { return "carrier_cost_events" }

func (c *CostEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cacc_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}
	return nil
}
