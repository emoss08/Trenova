package driversettlement

import (
	"context"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ pagination.CursorEntity            = (*PayEvent)(nil)
	_ validationframework.TenantedEntity = (*PayEvent)(nil)
)

type PayEventComponent struct {
	Kind        driverpay.ComponentKind `json:"kind"`
	Method      driverpay.CalcMethod    `json:"method"`
	Description string                  `json:"description"`
	Quantity    decimal.Decimal         `json:"quantity"`
	Rate        decimal.Decimal         `json:"rate"`
	AmountMinor int64                   `json:"amountMinor"`
}

// Pay accrued against a shipment, before and after it settles
type PayEvent struct {
	keel.Table `keel:"driver_pay_events,report=Payroll,label=Driver Pay Event"`

	ID               pulid.ID            `json:"id"               keel:"id,pk,autoid:dpe_"`
	BusinessUnitID   pulid.ID            `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID            `json:"organizationId"   keel:"organization_id,pk,tenant"`
	WorkerID         pulid.ID            `json:"workerId"         keel:"worker_id"`
	ShipmentID       pulid.ID            `json:"shipmentId"       keel:"shipment_id"`
	MoveID           *pulid.ID           `json:"moveId"           keel:"move_id"`
	AssignmentID     *pulid.ID           `json:"assignmentId"     keel:"assignment_id"`
	PayProfileID     *pulid.ID           `json:"payProfileId"     keel:"pay_profile_id"`
	SettlementID     *pulid.ID           `json:"settlementId"     keel:"settlement_id"`
	SettlementLineID *pulid.ID           `json:"settlementLineId" keel:"settlement_line_id"`
	IdempotencyKey   string              `json:"idempotencyKey"   keel:"idempotency_key,noreport"`
	Status           PayEventStatus      `json:"status"           keel:"status"`
	EventDate        int64               `json:"eventDate"        keel:"event_date"`
	GrossAmountMinor int64               `json:"grossAmountMinor" keel:"gross_amount_minor,label=Gross Amount (Minor)"`
	TotalMiles       decimal.Decimal     `json:"totalMiles"       keel:"total_miles,format=distance"`
	CurrencyCode     string              `json:"currencyCode"     keel:"currency_code,label=Currency"`
	Components       []PayEventComponent `json:"components"       keel:"components,jsonb,nullzero,noreport"`
	ProNumber        string              `json:"proNumber"        keel:"pro_number,nullzero"`
	OnHold           bool                `json:"onHold"           keel:"on_hold"`
	HoldReason       string              `json:"holdReason"       keel:"hold_reason,nullzero"`
	VoidedAt         *int64              `json:"voidedAt"         keel:"voided_at"`
	VoidReason       string              `json:"voidReason"       keel:"void_reason,nullzero"`
	Version          int64               `json:"version"          keel:"version,version"`
	CreatedAt        int64               `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64               `json:"updatedAt"        keel:"updated_at,autoupdate"`

	Worker     *worker.Worker     `json:"worker,omitempty"     keel:"belongs:worker_id,label=Driver"`
	Shipment   *shipment.Shipment `json:"shipment,omitempty"   keel:"belongs:shipment_id"`
	Settlement *Settlement        `json:"settlement,omitempty" keel:"belongs:settlement_id"`
}

func (p *PayEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&p.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(
			&p.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
		),
		validation.Field(&p.EventDate, validation.Required.Error("Event date is required")),
	))

	if !p.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Pay event status is invalid")
	}
	if p.GrossAmountMinor < 0 {
		multiErr.Add(
			"grossAmountMinor",
			errortypes.ErrInvalid,
			"Gross amount cannot be negative",
		)
	}
}

func (p *PayEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dpe",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (p *PayEvent) GetID() pulid.ID { return p.ID }

func (p *PayEvent) GetCreatedAt() int64 { return p.CreatedAt }

func (p *PayEvent) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *PayEvent) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *PayEvent) GetTableName() string { return "driver_pay_events" }

func (p *PayEvent) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = PayEventStatusAccrued
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (PayEvent) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_driver_pay_events_idempotency",
			Message: "Pay event with this idempotency key already exists",
		},
	}
}
