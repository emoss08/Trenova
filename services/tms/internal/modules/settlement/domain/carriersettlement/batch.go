package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*CarrierSettlementBatch)(nil)
	_ validationframework.TenantedEntity = (*CarrierSettlementBatch)(nil)
)

type CarrierSettlementBatch struct {
	keel.Table `keel:"carrier_settlement_batches"`

	ID              pulid.ID    `json:"id"              keel:"id,pk,autoid:carstlb_"`
	BusinessUnitID  pulid.ID    `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID    `json:"organizationId"  keel:"organization_id,pk,tenant"`
	Status          BatchStatus `json:"status"          keel:"status"`
	Name            string      `json:"name"            keel:"name"`
	PeriodStart     int64       `json:"periodStart"     keel:"period_start"`
	PeriodEnd       int64       `json:"periodEnd"       keel:"period_end"`
	PayDate         int64       `json:"payDate"         keel:"pay_date"`
	SettlementCount int         `json:"settlementCount" keel:"settlement_count"`
	TotalGrossMinor int64       `json:"totalGrossMinor" keel:"total_gross_minor"`
	TotalNetMinor   int64       `json:"totalNetMinor"   keel:"total_net_minor"`
	CurrencyCode    string      `json:"currencyCode"    keel:"currency_code"`
	Notes           string      `json:"notes"           keel:"notes,nullzero"`
	GeneratedByID   pulid.ID    `json:"generatedById"   keel:"generated_by_id,nullzero"`
	GeneratedAt     *int64      `json:"generatedAt"     keel:"generated_at"`
	CompletedAt     *int64      `json:"completedAt"     keel:"completed_at"`
	CanceledByID    pulid.ID    `json:"canceledById"    keel:"canceled_by_id,nullzero"`
	CanceledAt      *int64      `json:"canceledAt"      keel:"canceled_at"`
	Version         int64       `json:"version"         keel:"version,version"`
	CreatedAt       int64       `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64       `json:"updatedAt"       keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	Settlements  []*CarrierSettlement `json:"settlements,omitempty"  keel:"many:batch_id"`
}

func (b *CarrierSettlementBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&b.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&b.PeriodEnd, validation.Required.Error("Period end is required")),
		validation.Field(&b.PayDate, validation.Required.Error("Pay date is required")),
	))

	if !b.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Batch status is invalid")
	}
	if b.PeriodEnd <= b.PeriodStart {
		multiErr.Add(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}
}

func (b *CarrierSettlementBatch) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "carstlb",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (b *CarrierSettlementBatch) GetID() pulid.ID { return b.ID }

func (b *CarrierSettlementBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *CarrierSettlementBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *CarrierSettlementBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *CarrierSettlementBatch) GetTableName() string { return "carrier_settlement_batches" }

func (b *CarrierSettlementBatch) BeforeCreate(ctx context.Context) error {
	if b.Status == "" {
		b.Status = BatchStatusOpen
	}
	if b.CurrencyCode == "" {
		b.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (CarrierSettlementBatch) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_carrier_settlement_batches_period",
			Message: "An open carrier settlement batch already exists for this period",
		},
	}
}
