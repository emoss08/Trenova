package ifta

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
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

const (
	maxTimezoneLength        = 100
	maxFilingReferenceLength = 100
)

var (
	_ bun.BeforeAppendModelHook          = (*Return)(nil)
	_ pagination.CursorEntity            = (*Return)(nil)
	_ validationframework.TenantedEntity = (*Return)(nil)
)

type Return struct {
	bun.BaseModel             `bun:"table:ifta_returns,alias:ifr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                       json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Year            int          `json:"year"            bun:"year,type:SMALLINT,notnull"`
	Quarter         int          `json:"quarter"         bun:"quarter,type:SMALLINT,notnull"`
	AmendmentNumber int          `json:"amendmentNumber" bun:"amendment_number,type:SMALLINT,notnull,default:0"`
	AmendsReturnID  *pulid.ID    `json:"amendsReturnId"  bun:"amends_return_id,type:VARCHAR(100),nullzero"`
	Status          ReturnStatus `json:"status"          bun:"status,type:ifta_return_status_enum,notnull,default:'Draft'"`
	Timezone        string       `json:"timezone"        bun:"timezone,type:VARCHAR(100),notnull"`
	PeriodStart     int64        `json:"periodStart"     bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd       int64        `json:"periodEnd"       bun:"period_end,type:BIGINT,notnull"`

	TotalMiles          decimal.Decimal `json:"totalMiles"          bun:"total_miles,type:NUMERIC(14,2),notnull"`
	TotalTaxableMiles   decimal.Decimal `json:"totalTaxableMiles"   bun:"total_taxable_miles,type:NUMERIC(14,2),notnull"`
	TotalGallons        decimal.Decimal `json:"totalGallons"        bun:"total_gallons,type:NUMERIC(14,3),notnull"`
	TotalTaxPaidGallons decimal.Decimal `json:"totalTaxPaidGallons" bun:"total_tax_paid_gallons,type:NUMERIC(14,3),notnull"`
	NetTaxableGallons   decimal.Decimal `json:"netTaxableGallons"   bun:"net_taxable_gallons,type:NUMERIC(14,3),notnull"`

	TaxDueMinor       int64  `json:"taxDueMinor"       bun:"tax_due_minor,type:BIGINT,notnull"`
	SurchargeDueMinor int64  `json:"surchargeDueMinor" bun:"surcharge_due_minor,type:BIGINT,notnull"`
	NetDueMinor       int64  `json:"netDueMinor"       bun:"net_due_minor,type:BIGINT,notnull"`
	CurrencyCode      string `json:"currencyCode"      bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`

	FleetMPGByFuelType []FleetMPG `json:"fleetMpgByFuelType" bun:"fleet_mpg_by_fuel_type,type:JSONB,nullzero"`

	UnattributedMiles     decimal.Decimal `json:"unattributedMiles"     bun:"unattributed_miles,type:NUMERIC(14,2),notnull"`
	UnattributedMoveCount int             `json:"unattributedMoveCount" bun:"unattributed_move_count,type:INTEGER,notnull,default:0"`
	NoTractorMiles        decimal.Decimal `json:"noTractorMiles"        bun:"no_tractor_miles,type:NUMERIC(14,2),notnull"`
	NoTractorMoveCount    int             `json:"noTractorMoveCount"    bun:"no_tractor_move_count,type:INTEGER,notnull,default:0"`

	Problems   []Problem `json:"problems"   bun:"problems,type:JSONB,nullzero"`
	ComputedAt *int64    `json:"computedAt" bun:"computed_at,type:BIGINT,nullzero"`

	FinalizedAt   *int64   `json:"finalizedAt"   bun:"finalized_at,type:BIGINT,nullzero"`
	FinalizedByID pulid.ID `json:"finalizedById" bun:"finalized_by_id,type:VARCHAR(100),nullzero"`

	FiledAt         *int64   `json:"filedAt"         bun:"filed_at,type:BIGINT,nullzero"`
	FiledByID       pulid.ID `json:"filedById"       bun:"filed_by_id,type:VARCHAR(100),nullzero"`
	FilingReference string   `json:"filingReference" bun:"filing_reference,type:VARCHAR(100),nullzero"`

	ReopenedAt   *int64   `json:"reopenedAt"   bun:"reopened_at,type:BIGINT,nullzero"`
	ReopenedByID pulid.ID `json:"reopenedById" bun:"reopened_by_id,type:VARCHAR(100),nullzero"`
	ReopenReason string   `json:"reopenReason" bun:"reopen_reason,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	AmendsReturn *Return       `json:"amendsReturn,omitempty" bun:"rel:belongs-to,join:amends_return_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FinalizedBy  *tenant.User  `json:"finalizedBy,omitempty"  bun:"rel:belongs-to,join:finalized_by_id=id"`
	FiledBy      *tenant.User  `json:"filedBy,omitempty"      bun:"rel:belongs-to,join:filed_by_id=id"`
	ReopenedBy   *tenant.User  `json:"reopenedBy,omitempty"   bun:"rel:belongs-to,join:reopened_by_id=id"`
	Lines        []*ReturnLine `json:"lines,omitempty"        bun:"rel:has-many,join:id=return_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (r *Return) Normalize() {
	r.Timezone = strings.TrimSpace(r.Timezone)
	r.FilingReference = strings.TrimSpace(r.FilingReference)
	r.ReopenReason = strings.TrimSpace(r.ReopenReason)
	r.CurrencyCode = strings.ToUpper(strings.TrimSpace(r.CurrencyCode))
	if r.CurrencyCode == "" {
		r.CurrencyCode = money.DefaultCurrencyCode
	}
	if r.Status == "" {
		r.Status = ReturnStatusDraft
	}
}

func (r *Return) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ReturnStatus]("Status is not valid"),
		),
		validation.Field(&r.Timezone,
			validation.Required.Error("Timezone is required"),
			validation.Length(1, maxTimezoneLength).
				Error("Timezone cannot exceed 100 characters"),
		),
		validation.Field(&r.CurrencyCode,
			validation.Required.Error("Currency is required"),
			validation.Match(domaintypes.CurrencyCodeRegex).
				Error("Currency must be a three-letter ISO code"),
		),
		validation.Field(&r.FilingReference,
			validation.Length(0, maxFilingReferenceLength).
				Error("Filing reference cannot exceed 100 characters"),
		),
	))

	if err := r.Period().Validate(); err != nil {
		multiErr.Add("quarter", errortypes.ErrInvalid, err.Error())
	}
	if r.PeriodEnd <= r.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "Period end must be after period start")
	}

	r.validateAmendment(multiErr)
	r.validateStamps(multiErr)
}

func (r *Return) validateAmendment(multiErr *errortypes.MultiError) {
	if r.AmendmentNumber < 0 {
		multiErr.Add(
			"amendmentNumber",
			errortypes.ErrInvalid,
			"Amendment number cannot be negative",
		)
		return
	}

	hasParent := r.AmendsReturnID != nil && !r.AmendsReturnID.IsNil()
	switch {
	case r.AmendmentNumber > 0 && !hasParent:
		multiErr.Add(
			"amendsReturnId",
			errortypes.ErrRequired,
			"An amendment must reference the return it amends",
		)
	case r.AmendmentNumber == 0 && hasParent:
		multiErr.Add(
			"amendsReturnId",
			errortypes.ErrInvalid,
			"An original return cannot amend another return",
		)
	}
}

func (r *Return) validateStamps(multiErr *errortypes.MultiError) {
	hasFinalized := r.FinalizedAt != nil && *r.FinalizedAt > 0
	hasFiled := r.FiledAt != nil && *r.FiledAt > 0

	if r.Status.IsLocked() != hasFinalized {
		if hasFinalized {
			multiErr.Add(
				"finalizedAt",
				errortypes.ErrInvalid,
				"Only a finalized or filed return carries a finalized stamp",
			)
		} else {
			multiErr.Add(
				"finalizedAt",
				errortypes.ErrRequired,
				"A finalized return must record when it was finalized",
			)
		}
	}

	if (r.Status == ReturnStatusFiled) != hasFiled {
		if hasFiled {
			multiErr.Add(
				"filedAt",
				errortypes.ErrInvalid,
				"Only a filed return carries a filed stamp",
			)
		} else {
			multiErr.Add(
				"filedAt",
				errortypes.ErrRequired,
				"A filed return must record when it was filed",
			)
		}
	}

	if hasFinalized && hasFiled && *r.FiledAt < *r.FinalizedAt {
		multiErr.Add(
			"filedAt",
			errortypes.ErrInvalid,
			"A return cannot be filed before it was finalized",
		)
	}

	if (r.ReopenedAt != nil || r.ReopenReason != "") && len(r.ReopenReason) < MinReasonLength {
		multiErr.Add(
			"reopenReason",
			errortypes.ErrRequired,
			"A reopen reason of at least 10 characters is required",
		)
	}
}

func (r *Return) Period() Period { return Period{Year: r.Year, Quarter: r.Quarter} }

func (r *Return) IsLocked() bool { return r.Status.IsLocked() }

func (r *Return) IsAmendment() bool { return r.AmendmentNumber > 0 }

func (r *Return) CanRecompute() bool { return r.Status == ReturnStatusDraft }

func (r *Return) CanFinalize() bool {
	if r.Status != ReturnStatusDraft {
		return false
	}
	for i := range r.Problems {
		if r.Problems[i].Blocks() {
			return false
		}
	}
	return true
}

func (r *Return) CanReopen() bool { return r.Status == ReturnStatusFinalized }

func (r *Return) CanMarkFiled() bool { return r.Status == ReturnStatusFinalized }

func (r *Return) CanAmend() bool { return r.Status == ReturnStatusFiled }

func (r *Return) CanDelete() bool { return r.Status == ReturnStatusDraft }

func (r *Return) HasProblem(code ProblemCode) bool {
	for i := range r.Problems {
		if r.Problems[i].Code == code {
			return true
		}
	}
	return false
}

func (r *Return) BlockingProblems() []Problem {
	out := make([]Problem, 0, len(r.Problems))
	for i := range r.Problems {
		if r.Problems[i].Blocks() {
			out = append(out, r.Problems[i])
		}
	}
	return out
}

func (r *Return) TaxDue() decimal.Decimal { return money.DecimalFromMinor(r.TaxDueMinor) }

func (r *Return) SurchargeDue() decimal.Decimal {
	return money.DecimalFromMinor(r.SurchargeDueMinor)
}

func (r *Return) NetDue() decimal.Decimal { return money.DecimalFromMinor(r.NetDueMinor) }

func (r *Return) IsCredit() bool { return r.NetDueMinor < 0 }

func (r *Return) GetID() pulid.ID { return r.ID }

func (r *Return) GetCreatedAt() int64 { return r.CreatedAt }

func (r *Return) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *Return) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *Return) GetTableName() string { return "ifta_returns" }

func (r *Return) GetResourceType() string { return "ifta_return" }

func (r *Return) GetResourceID() string { return r.ID.String() }

func (r *Return) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ifr_")
		}
		if r.Status == "" {
			r.Status = ReturnStatusDraft
		}
		if r.CurrencyCode == "" {
			r.CurrencyCode = money.DefaultCurrencyCode
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *Return) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ifr",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "filing_reference",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}
