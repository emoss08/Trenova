package factoring

import (
	"context"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	BasisPoints              = 10000
	MaxChargebackWindowDays  = 730
	MaxFundingDays           = 60
	MinutesPerDay            = 1440
	MaxAgreementNotesLength  = 2000
	MaxFeeTiers              = 20
	DefaultChargebackWindow  = 90
	ChargebackWarningDays    = 7
	DefaultFundingDays       = 2
	DefaultSubmissionMinutes = 600
)

var (
	_ bun.BeforeAppendModelHook = (*Agreement)(nil)
	_ bun.BeforeAppendModelHook = (*FeeTier)(nil)
)

type Agreement struct {
	bun.BaseModel             `bun:"table:factoring_agreements,alias:facagr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                  json:"-"`

	ID                     pulid.ID            `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID            `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	FactoringCompanyID     pulid.ID            `json:"factoringCompanyId"     bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	Status                 AgreementStatus     `json:"status"                 bun:"status,type:VARCHAR(20),notnull,default:'Draft'"`
	EffectiveFrom          int64               `json:"effectiveFrom"          bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo            *int64              `json:"effectiveTo"            bun:"effective_to,type:BIGINT,nullzero"`
	AdvanceRateBP          int                 `json:"advanceRateBp"          bun:"advance_rate_bp,type:INTEGER,notnull"`
	ReserveRateBP          int                 `json:"reserveRateBp"          bun:"reserve_rate_bp,type:INTEGER,notnull"`
	MinimumFeeMinor        int64               `json:"minimumFeeMinor"        bun:"minimum_fee_minor,type:BIGINT,notnull"`
	ChargebackWindowDays   int                 `json:"chargebackWindowDays"   bun:"chargeback_window_days,type:INTEGER,notnull"`
	Recourse               Recourse            `json:"recourse"               bun:"recourse,type:VARCHAR(20),notnull,default:'Recourse'"`
	AccountingTreatment    AccountingTreatment `json:"accountingTreatment"    bun:"accounting_treatment,type:VARCHAR(30),notnull,default:'Sale'"`
	FundingDays            int                 `json:"fundingDays"            bun:"funding_days,type:INTEGER,notnull"`
	SubmissionTime         int                 `json:"submissionTime"         bun:"submission_time,type:INTEGER,notnull"`
	Notes                  string              `json:"notes"                  bun:"notes,type:TEXT,nullzero"`
	DueFromFactorAccountID pulid.ID            `json:"dueFromFactorAccountId" bun:"due_from_factor_account_id,type:VARCHAR(100),nullzero"`
	FeeExpenseAccountID    pulid.ID            `json:"feeExpenseAccountId"    bun:"fee_expense_account_id,type:VARCHAR(100),nullzero"`
	LiabilityAccountID     pulid.ID            `json:"liabilityAccountId"     bun:"factoring_liability_account_id,type:VARCHAR(100),nullzero"`
	CashAccountID          pulid.ID            `json:"cashAccountId"          bun:"cash_account_id,type:VARCHAR(100),nullzero"`
	ActivatedAt            *int64              `json:"activatedAt"            bun:"activated_at,type:BIGINT,nullzero"`
	EndedAt                *int64              `json:"endedAt"                bun:"ended_at,type:BIGINT,nullzero"`
	Version                int64               `json:"version"                bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64               `json:"createdAt"              bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt              int64               `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FeeTiers         []*FeeTier                         `json:"feeTiers"                   bun:"rel:has-many,join:id=agreement_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DocumentTypes    []*documenttype.DocumentType       `json:"documentTypes"              bun:"m2m:factoring_agreement_document_types,join:Agreement=DocumentType"`
}

type FeeTier struct {
	bun.BaseModel `bun:"table:factoring_fee_tiers,alias:facfee" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AgreementID    pulid.ID `json:"agreementId"    bun:"agreement_id,type:VARCHAR(100),notnull"`
	DaysFrom       int      `json:"daysFrom"       bun:"days_from,type:INTEGER,notnull"`
	DaysTo         *int     `json:"daysTo"         bun:"days_to,type:INTEGER,nullzero"`
	RateBP         int      `json:"rateBp"         bun:"rate_bp,type:INTEGER,notnull"`
	FlatMinor      int64    `json:"flatMinor"      bun:"flat_minor,type:BIGINT,notnull"`
}

type AgreementDocumentType struct {
	bun.BaseModel `bun:"table:factoring_agreement_document_types,alias:facadt" json:"-"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AgreementID    pulid.ID `json:"agreementId"    bun:"agreement_id,pk,type:VARCHAR(100),notnull"`
	DocumentTypeID pulid.ID `json:"documentTypeId" bun:"document_type_id,pk,type:VARCHAR(100),notnull"`

	Agreement    *Agreement                 `json:"-" bun:"rel:belongs-to,join:agreement_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DocumentType *documenttype.DocumentType `json:"-" bun:"rel:belongs-to,join:document_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *Agreement) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OrganizationID, validation.Required),
		validation.Field(&a.BusinessUnitID, validation.Required),
		validation.Field(
			&a.FactoringCompanyID,
			validation.Required.Error("Factoring company is required"),
		),
		validation.Field(
			&a.Status,
			validation.Required,
			domainvalidation.ValidEnum[AgreementStatus]("Invalid agreement status"),
		),
		validation.Field(&a.EffectiveFrom, validation.Required.Error("Start date is required")),
		validation.Field(
			&a.AdvanceRateBP,
			validation.Required.Error("Advance rate is required"),
			validation.Min(1).Error("Advance rate must be greater than zero"),
			validation.Max(BasisPoints).Error("Advance rate cannot be more than 100%"),
		),
		validation.Field(
			&a.ReserveRateBP,
			validation.Min(0).Error("Reserve rate cannot be negative"),
			validation.Max(BasisPoints).Error("Reserve rate cannot be more than 100%"),
		),
		validation.Field(
			&a.MinimumFeeMinor,
			validation.Min(int64(0)).Error("Minimum fee cannot be negative"),
		),
		validation.Field(
			&a.ChargebackWindowDays,
			validation.Min(0).Error("Chargeback window cannot be negative"),
			validation.Max(MaxChargebackWindowDays).
				Error("Chargeback window cannot be longer than 730 days"),
		),
		validation.Field(
			&a.Recourse,
			validation.Required.Error("Recourse is required"),
			domainvalidation.ValidEnum[Recourse]("Invalid recourse"),
		),
		validation.Field(
			&a.AccountingTreatment,
			validation.Required.Error("Accounting treatment is required"),
			domainvalidation.ValidEnum[AccountingTreatment]("Invalid accounting treatment"),
		),
		validation.Field(
			&a.FundingDays,
			validation.Min(0).Error("Funding days cannot be negative"),
			validation.Max(MaxFundingDays).Error("Funding days cannot be more than 60"),
		),
		validation.Field(
			&a.SubmissionTime,
			validation.Min(0).Error("Submission time must be within the day"),
			validation.Max(MinutesPerDay-1).Error("Submission time must be within the day"),
		),
		validation.Field(
			&a.Notes,
			validation.Length(0, MaxAgreementNotesLength).
				Error("Notes cannot be longer than 2000 characters"),
		),
	))

	if a.AdvanceRateBP+a.ReserveRateBP > BasisPoints {
		multiErr.Add(
			"reserveRateBp",
			errortypes.ErrInvalid,
			"Advance and reserve together cannot be more than 100%",
		)
	}
	if a.EffectiveTo != nil && *a.EffectiveTo <= a.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"End date must be after the start date",
		)
	}
	ValidateFeeTiers(a.FeeTiers, multiErr)
}

func ValidateFeeTiers(tiers []*FeeTier, multiErr *errortypes.MultiError) {
	if len(tiers) == 0 {
		multiErr.Add("feeTiers", errortypes.ErrRequired, "Add at least one fee tier")
		return
	}
	if len(tiers) > MaxFeeTiers {
		multiErr.Add(
			"feeTiers",
			errortypes.ErrInvalid,
			"An agreement can have at most 20 fee tiers",
		)
		return
	}

	ordered := SortedFeeTiers(tiers)
	next := 0
	for idx, tier := range ordered {
		field := "feeTiers[" + strconv.Itoa(slices.Index(tiers, tier)) + "]"
		if tier.RateBP < 0 || tier.RateBP > BasisPoints {
			multiErr.Add(
				field+".rateBp",
				errortypes.ErrInvalid,
				"Fee rate must be between 0% and 100%",
			)
		}
		if tier.FlatMinor < 0 {
			multiErr.Add(field+".flatMinor", errortypes.ErrInvalid, "Flat fee cannot be negative")
		}
		if tier.DaysFrom != next {
			multiErr.Add(
				field+".daysFrom",
				errortypes.ErrInvalid,
				"Fee tiers must start at day 0 and follow each other without gaps",
			)
		}
		if tier.DaysTo == nil {
			if idx != len(ordered)-1 {
				multiErr.Add(
					field+".daysTo",
					errortypes.ErrInvalid,
					"Only the last fee tier can be open-ended",
				)
			}
			continue
		}
		if *tier.DaysTo <= tier.DaysFrom {
			multiErr.Add(
				field+".daysTo",
				errortypes.ErrInvalid,
				"A fee tier must end after it starts",
			)
		}
		next = *tier.DaysTo
	}
}

func SortedFeeTiers(tiers []*FeeTier) []*FeeTier {
	ordered := slices.Clone(tiers)
	slices.SortStableFunc(ordered, func(a, b *FeeTier) int {
		return a.DaysFrom - b.DaysFrom
	})
	return ordered
}

func (a *Agreement) InForce(at int64) bool {
	if a == nil || a.Status != AgreementStatusActive || at < a.EffectiveFrom {
		return false
	}
	return a.EffectiveTo == nil || at < *a.EffectiveTo
}

func (a *Agreement) IsNonRecourse() bool {
	return a != nil && a.Recourse == RecourseNone
}

type PostingAccounts struct {
	AR            pulid.ID
	Cash          pulid.ID
	DueFromFactor pulid.ID
	FeeExpense    pulid.ID
	Liability     pulid.ID
}

type MissingAccount struct {
	Field   string
	Message string
}

func (a *Agreement) PostingAccounts(defaultAR, defaultCash pulid.ID) PostingAccounts {
	cash := a.CashAccountID
	if cash.IsNil() {
		cash = defaultCash
	}
	return PostingAccounts{
		AR:            defaultAR,
		Cash:          cash,
		DueFromFactor: a.DueFromFactorAccountID,
		FeeExpense:    a.FeeExpenseAccountID,
		Liability:     a.LiabilityAccountID,
	}
}

func (p *PostingAccounts) For(role AccountRole) pulid.ID {
	switch role {
	case AccountRoleAR:
		return p.AR
	case AccountRoleCash:
		return p.Cash
	case AccountRoleDueFromFactor:
		return p.DueFromFactor
	case AccountRoleFeeExpense:
		return p.FeeExpense
	case AccountRoleLiability:
		return p.Liability
	default:
		return pulid.Nil
	}
}

func (p *PostingAccounts) Missing(treatment AccountingTreatment) []MissingAccount {
	missing := make([]MissingAccount, 0, 5)
	if p.AR.IsNil() {
		missing = append(missing, MissingAccount{
			Field:   "defaultArAccountId",
			Message: "Set the organization's accounts receivable account in accounting settings before booking factoring",
		})
	}
	if p.Cash.IsNil() {
		missing = append(missing, MissingAccount{
			Field:   "cashAccountId",
			Message: "Set a cash account on the agreement or in accounting settings before booking factoring",
		})
	}
	if p.FeeExpense.IsNil() {
		missing = append(missing, MissingAccount{
			Field:   "feeExpenseAccountId",
			Message: "Set the agreement's factoring fee account before booking factoring",
		})
	}
	switch treatment {
	case AccountingTreatmentSecuredBorrowing:
		if p.Liability.IsNil() {
			missing = append(missing, MissingAccount{
				Field:   "liabilityAccountId",
				Message: "Set the agreement's factoring liability account before booking factoring",
			})
		}
	case AccountingTreatmentSale:
		if p.DueFromFactor.IsNil() {
			missing = append(missing, MissingAccount{
				Field:   "dueFromFactorAccountId",
				Message: "Set the agreement's due-from-factor account before booking factoring",
			})
		}
	}
	return missing
}

func (a *Agreement) ChargebackDeadline(fundedOn int64) int64 {
	return fundedOn + int64(a.ChargebackWindowDays)*timeutils.SecondsPerDay
}

func (a *Agreement) DocumentTypeIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(a.DocumentTypes))
	for _, dt := range a.DocumentTypes {
		if dt != nil {
			ids = append(ids, dt.ID)
		}
	}
	return ids
}

func (a *Agreement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("facagr_")
		}
		if a.Status == "" {
			a.Status = AgreementStatusDraft
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}

func (a *Agreement) GetID() pulid.ID             { return a.ID }
func (a *Agreement) GetOrganizationID() pulid.ID { return a.OrganizationID }
func (a *Agreement) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }
func (a *Agreement) GetTableName() string        { return "factoring_agreements" }

func (t *FeeTier) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok && t.ID.IsNil() {
		t.ID = pulid.MustNew("facfee_")
	}
	return nil
}
