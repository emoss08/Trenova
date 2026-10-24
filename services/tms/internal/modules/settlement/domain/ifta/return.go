package ifta

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	maxTimezoneLength        = 100
	maxFilingReferenceLength = 100
)

var (
	_ pagination.CursorEntity            = (*Return)(nil)
	_ validationframework.TenantedEntity = (*Return)(nil)
)

// Quarterly fuel tax returns
type Return struct {
	keel.Table `                                            keel:"ifta_returns,report=Equipment,label=IFTA Return"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ifr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Year            int          `json:"year" keel:"year,dimension"`
	Quarter         int          `json:"quarter" keel:"quarter,dimension"`
	AmendmentNumber int          `json:"amendmentNumber" keel:"amendment_number,dimension"`
	AmendsReturnID  *pulid.ID    `json:"amendsReturnId" keel:"amends_return_id"`
	Status          ReturnStatus `json:"status" keel:"status"`
	Timezone        string       `json:"timezone" keel:"timezone"`
	PeriodStart     int64        `json:"periodStart" keel:"period_start,type=epoch"`
	PeriodEnd       int64        `json:"periodEnd" keel:"period_end,type=epoch"`

	TotalMiles          decimal.Decimal `json:"totalMiles" keel:"total_miles,format=distance"`
	TotalTaxableMiles   decimal.Decimal `json:"totalTaxableMiles" keel:"total_taxable_miles,label=Taxable Miles,format=distance"`
	TotalGallons        decimal.Decimal `json:"totalGallons" keel:"total_gallons"`
	TotalTaxPaidGallons decimal.Decimal `json:"totalTaxPaidGallons" keel:"total_tax_paid_gallons,label=Tax-Paid Gallons"`
	NetTaxableGallons   decimal.Decimal `json:"netTaxableGallons" keel:"net_taxable_gallons"`

	TaxDueMinor       int64  `json:"taxDueMinor" keel:"tax_due_minor,label=Tax Due (Minor)"`
	SurchargeDueMinor int64  `json:"surchargeDueMinor" keel:"surcharge_due_minor,label=Surcharge Due (Minor)"`
	NetDueMinor       int64  `json:"netDueMinor" keel:"net_due_minor,label=Net Due (Minor)"`
	CurrencyCode      string `json:"currencyCode" keel:"currency_code"`

	FleetMPGByFuelType []FleetMPG `json:"fleetMpgByFuelType" keel:"fleet_mpg_by_fuel_type,jsonb,nullzero"`

	UnattributedMiles     decimal.Decimal `json:"unattributedMiles" keel:"unattributed_miles"`
	UnattributedMoveCount int             `json:"unattributedMoveCount" keel:"unattributed_move_count"`
	NoTractorMiles        decimal.Decimal `json:"noTractorMiles" keel:"no_tractor_miles"`
	NoTractorMoveCount    int             `json:"noTractorMoveCount" keel:"no_tractor_move_count"`

	Problems   []Problem `json:"problems" keel:"problems,jsonb,nullzero"`
	ComputedAt *int64    `json:"computedAt" keel:"computed_at"`

	FinalizedAt   *int64   `json:"finalizedAt" keel:"finalized_at"`
	FinalizedByID pulid.ID `json:"finalizedById" keel:"finalized_by_id,nullzero"`

	FiledAt         *int64   `json:"filedAt" keel:"filed_at"`
	FiledByID       pulid.ID `json:"filedById" keel:"filed_by_id,nullzero"`
	FilingReference string   `json:"filingReference" keel:"filing_reference,nullzero"`

	ReopenedAt   *int64   `json:"reopenedAt" keel:"reopened_at"`
	ReopenedByID pulid.ID `json:"reopenedById" keel:"reopened_by_id,nullzero"`
	ReopenReason string   `json:"reopenReason" keel:"reopen_reason,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	AmendsReturn *Return       `json:"amendsReturn,omitempty" keel:"belongs:amends_return_id"`
	FinalizedBy  *tenant.User  `json:"finalizedBy,omitempty" keel:"belongs:finalized_by_id,noreport"`
	FiledBy      *tenant.User  `json:"filedBy,omitempty" keel:"belongs:filed_by_id,noreport"`
	ReopenedBy   *tenant.User  `json:"reopenedBy,omitempty" keel:"belongs:reopened_by_id,noreport"`
	Lines        []*ReturnLine `json:"lines,omitempty" keel:"many:return_id"`
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

func (r *Return) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = ReturnStatusDraft
	}
	if r.CurrencyCode == "" {
		r.CurrencyCode = "USD"
	}
	if r.CurrencyCode == "" {
		r.CurrencyCode = money.DefaultCurrencyCode
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
