package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*CostEvent)(nil)
	_ validationframework.TenantedEntity = (*CostEvent)(nil)
)

// Purchased transportation cost accruals from carrier assignments
type CostEvent struct {
	keel.Table `keel:"carrier_cost_events,report=Billing,label=Carrier Cost Event"`

	ID                  pulid.ID        `json:"id"                  keel:"id,pk,autoid:cacc_"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID        `json:"organizationId"      keel:"organization_id,pk,tenant"`
	CarrierID           pulid.ID        `json:"carrierId"           keel:"carrier_id"`
	CarrierAssignmentID *pulid.ID       `json:"carrierAssignmentId" keel:"carrier_assignment_id"`
	ShipmentID          *pulid.ID       `json:"shipmentId"          keel:"shipment_id"`
	MoveID              *pulid.ID       `json:"moveId"              keel:"move_id"`
	SettlementID        *pulid.ID       `json:"settlementId"        keel:"settlement_id"`
	EventType           CostEventType   `json:"eventType"           keel:"event_type"`
	Status              CostEventStatus `json:"status"              keel:"status"`
	IdempotencyKey      string          `json:"idempotencyKey"      keel:"idempotency_key"`
	EventDate           int64           `json:"eventDate"           keel:"event_date"`
	AmountMinor         int64           `json:"amountMinor"         keel:"amount_minor,label=Amount (Minor)"`
	CurrencyCode        string          `json:"currencyCode"        keel:"currency_code"`
	Description         string          `json:"description"         keel:"description,nullzero"`
	ProNumber           string          `json:"proNumber"           keel:"pro_number,nullzero"`
	AssignmentVersion   int64           `json:"assignmentVersion"   keel:"assignment_version,dimension"`
	VoidedAt            *int64          `json:"voidedAt"            keel:"voided_at"`
	VoidReason          string          `json:"voidReason"          keel:"void_reason,nullzero"`
	Version             int64           `json:"version"             keel:"version,version"`
	CreatedAt           int64           `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64           `json:"updatedAt"           keel:"updated_at,autoupdate"`

	Carrier *carrier.Carrier `json:"carrier,omitempty" keel:"belongs:carrier_id"`
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

func (c *CostEvent) BeforeCreate(ctx context.Context) error {
	if c.Status == "" {
		c.Status = CostEventStatusPending
	}
	if c.CurrencyCode == "" {
		c.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}
