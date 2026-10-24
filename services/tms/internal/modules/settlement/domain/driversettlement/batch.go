package driversettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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
	_ bun.BeforeAppendModelHook          = (*SettlementBatch)(nil)
	_ pagination.CursorEntity            = (*SettlementBatch)(nil)
	_ validationframework.TenantedEntity = (*SettlementBatch)(nil)
)

// A pay run and its totals
type SettlementBatch struct {
	bun.BaseModel             `bun:"table:driver_settlement_batches,alias:dstlb" json:"-"`
	keel.Table                `                                                           keel:"driver_settlement_batches,report=Payroll"`
	pagination.CursorValueSet `bun:",embed"                                      json:"-"`

	ID              pulid.ID    `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:dstlb_"`
	BusinessUnitID  pulid.ID    `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID    `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	Status          BatchStatus `json:"status"          bun:"status,type:VARCHAR(50),notnull,default:'Open'"                                       keel:"status"`
	Name            string      `json:"name"            bun:"name,type:VARCHAR(100),notnull"                                                       keel:"name,label=Batch Name"`
	PeriodStart     int64       `json:"periodStart"     bun:"period_start,type:BIGINT,notnull"                                                     keel:"period_start,type=epoch"`
	PeriodEnd       int64       `json:"periodEnd"       bun:"period_end,type:BIGINT,notnull"                                                       keel:"period_end,type=epoch"`
	PayDate         int64       `json:"payDate"         bun:"pay_date,type:BIGINT,notnull"                                                         keel:"pay_date"`
	SettlementCount int         `json:"settlementCount" bun:"settlement_count,type:INTEGER,notnull"                                                keel:"settlement_count,label=Settlements,format=count"`
	ExceptionCount  int         `json:"exceptionCount"  bun:"exception_count,type:INTEGER,notnull"                                                 keel:"exception_count,label=Exceptions,format=count"`
	TotalGrossMinor int64       `json:"totalGrossMinor" bun:"total_gross_minor,type:BIGINT,notnull"                                                keel:"total_gross_minor,label=Total Gross (Minor)"`
	TotalNetMinor   int64       `json:"totalNetMinor"   bun:"total_net_minor,type:BIGINT,notnull"                                                  keel:"total_net_minor,label=Total Net (Minor)"`
	CurrencyCode    string      `json:"currencyCode"    bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"                                  keel:"currency_code,label=Currency"`
	Notes           string      `json:"notes"           bun:"notes,type:TEXT,nullzero"                                                             keel:"notes,nullzero"`
	GeneratedByID   pulid.ID    `json:"generatedById"   bun:"generated_by_id,type:VARCHAR(100),nullzero"                                           keel:"generated_by_id,nullzero"`
	GeneratedAt     *int64      `json:"generatedAt"     bun:"generated_at,type:BIGINT,nullzero"                                                    keel:"generated_at"`
	CompletedAt     *int64      `json:"completedAt"     bun:"completed_at,type:BIGINT,nullzero"                                                    keel:"completed_at"`
	CanceledByID    pulid.ID    `json:"canceledById"    bun:"canceled_by_id,type:VARCHAR(100),nullzero"                                            keel:"canceled_by_id,nullzero"`
	CanceledAt      *int64      `json:"canceledAt"      bun:"canceled_at,type:BIGINT,nullzero"                                                     keel:"canceled_at"`
	Version         int64       `json:"version"         bun:"version,type:BIGINT,notnull"                                                          keel:"version,version"`
	CreatedAt       int64       `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt       int64       `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Settlements  []*Settlement        `json:"settlements,omitempty"  bun:"rel:has-many,join:id=batch_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"many:batch_id"`
}

func (b *SettlementBatch) Validate(multiErr *errortypes.MultiError) {
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

func (b *SettlementBatch) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dstlb",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (b *SettlementBatch) GetID() pulid.ID { return b.ID }

func (b *SettlementBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *SettlementBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *SettlementBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *SettlementBatch) GetTableName() string { return "driver_settlement_batches" }

func (b *SettlementBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("dstlb_")
		}
		b.CreatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}
	return nil
}

func (b *SettlementBatch) BeforeCreate(ctx context.Context) error {
	return b.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
