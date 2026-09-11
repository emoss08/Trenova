package invoicerun

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*InvoiceRun)(nil)
	_ domaintypes.PostgresSearchable     = (*InvoiceRun)(nil)
	_ pagination.CursorEntity            = (*InvoiceRun)(nil)
	_ validationframework.TenantedEntity = (*InvoiceRun)(nil)
)

// InvoiceRun is one pass at turning a period's approved billing-queue items into
// invoices.
//
// It is persisted rather than derived because the flow is preview, adjust,
// commit across separate requests. A transient preview would force the server to
// re-derive eligibility at commit time and could silently bill a different set
// than the operator approved; it would give an operator's exclusion no resource
// to audit against; and it would leave a scheduled run that failed halfway with
// nothing to resume from.
type InvoiceRun struct {
	bun.BaseModel             `bun:"table:invoice_runs,alias:invrun" json:"-"`
	pagination.CursorValueSet `bun:",embed"                          json:"-"`

	ID             pulid.ID              `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID              `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID              `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Number         string                `json:"number"         bun:"number,type:VARCHAR(100),notnull"`
	Status         Status                `json:"status"         bun:"status,type:invoice_run_status_enum,notnull,default:'Building'"`
	Source         Source                `json:"source"         bun:"source,type:invoice_run_source_enum,notnull"`
	Cycle          customer.BillingCycle `json:"cycle"          bun:"cycle,type:customer_billing_cycle_enum,nullzero"`
	PeriodStart    int64                 `json:"periodStart"    bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd      int64                 `json:"periodEnd"      bun:"period_end,type:BIGINT,notnull"`
	InvoiceDate    int64                 `json:"invoiceDate"    bun:"invoice_date,type:BIGINT,notnull"`
	CustomerIDs    []string              `json:"customerIds"    bun:"customer_ids,array,type:text[],nullzero"`
	CurrencyCode   string                `json:"currencyCode"   bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`

	GroupCount       int             `json:"groupCount"       bun:"group_count,type:INTEGER,notnull"`
	ItemCount        int             `json:"itemCount"        bun:"item_count,type:INTEGER,notnull"`
	ExcludedCount    int             `json:"excludedCount"    bun:"excluded_count,type:INTEGER,notnull"`
	InvoiceCount     int             `json:"invoiceCount"     bun:"invoice_count,type:INTEGER,notnull"`
	TotalAmount      decimal.Decimal `json:"totalAmount"      bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalAmountMinor int64           `json:"totalAmountMinor" bun:"total_amount_minor,type:BIGINT,notnull"`

	FailureReason string `json:"failureReason" bun:"failure_reason,type:TEXT,nullzero"`

	// OffCycleReason is why this run billed a period before its boundary. Empty
	// on the ordinary path. It is kept on the run rather than only in the audit
	// log because the next biller looking at the customer's history needs to see
	// why a month has two invoices without going hunting for it.
	OffCycleReason string `json:"offCycleReason" bun:"off_cycle_reason,type:TEXT,nullzero"`

	BuiltByID pulid.ID `json:"builtById" bun:"built_by_id,type:VARCHAR(100),nullzero"`
	BuiltAt       *int64   `json:"builtAt"       bun:"built_at,type:BIGINT,nullzero"`
	CommittedByID pulid.ID `json:"committedById" bun:"committed_by_id,type:VARCHAR(100),nullzero"`
	CommittedAt   *int64   `json:"committedAt"   bun:"committed_at,type:BIGINT,nullzero"`
	CanceledByID  pulid.ID `json:"canceledById"  bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CanceledAt    *int64   `json:"canceledAt"    bun:"canceled_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Groups       []*InvoiceRunGroup   `json:"groups,omitempty"       bun:"rel:has-many,join:id=run_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (r *InvoiceRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.Number,
			validation.Required.Error("Run number is required"),
			validation.Length(1, 100).Error("Run number must be between 1 and 100 characters"),
		),
		validation.Field(&r.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&r.PeriodEnd, validation.Required.Error("Period end is required")),
		validation.Field(&r.InvoiceDate, validation.Required.Error("Invoice date is required")),
		validation.Field(&r.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be a three letter code"),
		),
	))

	if !r.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Run status is invalid")
	}
	if !r.Source.IsValid() {
		multiErr.Add("source", errortypes.ErrInvalid, "Run source is invalid")
	}
	if r.PeriodEnd <= r.PeriodStart {
		multiErr.Add(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}
	// A scheduled run is keyed on its cycle, so two workers racing the same cron
	// tick collide on the unique index rather than both building the period.
	if r.Source == SourceScheduled && r.Cycle == "" {
		multiErr.Add(
			"cycle",
			errortypes.ErrRequired,
			"A scheduled run must record the billing cycle that produced it",
		)
	}

	for i, group := range r.Groups {
		if group == nil {
			continue
		}
		group.Validate(multiErr.WithIndex("groups", i))
	}
}

// SyncTotals recomputes the run's rollups from its groups, counting only the
// groups that would actually bill.
func (r *InvoiceRun) SyncTotals() {
	total := decimal.Zero
	groups := 0
	items := 0
	excluded := 0

	for _, group := range r.Groups {
		if group == nil {
			continue
		}

		group.SyncTotals()
		excluded += group.ExcludedCount()

		if group.Status == GroupStatusSkipped {
			continue
		}

		groups++
		items += group.ItemCount
		total = total.Add(group.TotalAmount)
	}

	r.GroupCount = groups
	r.ItemCount = items
	r.ExcludedCount = excluded
	r.TotalAmount = total
	r.TotalAmountMinor = money.MinorUnits(total)
}

func (r *InvoiceRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("invrun_")
		}
		if r.Status == "" {
			r.Status = StatusBuilding
		}
		if r.CurrencyCode == "" {
			r.CurrencyCode = "USD"
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *InvoiceRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "invrun",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "number", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "source", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (r *InvoiceRun) GetID() pulid.ID             { return r.ID }
func (r *InvoiceRun) GetCreatedAt() int64         { return r.CreatedAt }
func (r *InvoiceRun) GetTableName() string        { return "invoice_runs" }
func (r *InvoiceRun) GetOrganizationID() pulid.ID { return r.OrganizationID }
func (r *InvoiceRun) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }
