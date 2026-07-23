package driversettlement

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*PayEvent)(nil)
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

type PayEvent struct {
	bun.BaseModel             `bun:"table:driver_pay_events,alias:dpe" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID               pulid.ID            `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID            `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID            `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID         pulid.ID            `json:"workerId"         bun:"worker_id,type:VARCHAR(100),notnull"`
	ShipmentID       pulid.ID            `json:"shipmentId"       bun:"shipment_id,type:VARCHAR(100),notnull"`
	MoveID           *pulid.ID           `json:"moveId"           bun:"move_id,type:VARCHAR(100),nullzero"`
	AssignmentID     *pulid.ID           `json:"assignmentId"     bun:"assignment_id,type:VARCHAR(100),nullzero"`
	PayProfileID     *pulid.ID           `json:"payProfileId"     bun:"pay_profile_id,type:VARCHAR(100),nullzero"`
	SettlementID     *pulid.ID           `json:"settlementId"     bun:"settlement_id,type:VARCHAR(100),nullzero"`
	SettlementLineID *pulid.ID           `json:"settlementLineId" bun:"settlement_line_id,type:VARCHAR(100),nullzero"`
	IdempotencyKey   string              `json:"idempotencyKey"   bun:"idempotency_key,type:VARCHAR(255),notnull"`
	Status           PayEventStatus      `json:"status"           bun:"status,type:VARCHAR(50),notnull,default:'Accrued'"`
	EventDate        int64               `json:"eventDate"        bun:"event_date,type:BIGINT,notnull"`
	GrossAmountMinor int64               `json:"grossAmountMinor" bun:"gross_amount_minor,type:BIGINT,notnull,default:0"`
	TotalMiles       decimal.Decimal     `json:"totalMiles"       bun:"total_miles,type:NUMERIC(19,4),notnull,default:0"`
	CurrencyCode     string              `json:"currencyCode"     bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Components       []PayEventComponent `json:"components"       bun:"components,type:JSONB,nullzero"`
	ProNumber        string              `json:"proNumber"        bun:"pro_number,type:VARCHAR(100),nullzero"`
	OnHold           bool                `json:"onHold"           bun:"on_hold,type:BOOLEAN,notnull,default:false"`
	HoldReason       string              `json:"holdReason"       bun:"hold_reason,type:TEXT,nullzero"`
	VoidedAt         *int64              `json:"voidedAt"         bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason       string              `json:"voidReason"       bun:"void_reason,type:TEXT,nullzero"`
	Version          int64               `json:"version"          bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt        int64               `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64               `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *PayEvent) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&p.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(
			&p.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
		),
		validation.Field(&p.EventDate, validation.Required.Error("Event date is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

func (p *PayEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("dpe_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}
