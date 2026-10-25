package factoring

import (
	"context"
	"slices"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

// The terms each factor buys invoices on, with their advance, reserve and effective dates
type Agreement struct {
	keel.Table `keel:"factoring_agreements,report=Billing,label=Factoring Agreement"`

	ID                     pulid.ID            `json:"id"                     keel:"id,pk,autoid:facagr_"`
	OrganizationID         pulid.ID            `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID     pulid.ID            `json:"factoringCompanyId"     keel:"factoring_company_id"`
	Status                 AgreementStatus     `json:"status"                 keel:"status"`
	EffectiveFrom          int64               `json:"effectiveFrom"          keel:"effective_from,type=epoch"`
	EffectiveTo            *int64              `json:"effectiveTo"            keel:"effective_to,type=epoch"`
	AdvanceRateBP          int                 `json:"advanceRateBp"          keel:"advance_rate_bp,label=Advance Rate (bp),dimension"`
	ReserveRateBP          int                 `json:"reserveRateBp"          keel:"reserve_rate_bp,label=Reserve Rate (bp),dimension"`
	MinimumFeeMinor        int64               `json:"minimumFeeMinor"        keel:"minimum_fee_minor,label=Minimum Fee (Minor)"`
	ChargebackWindowDays   int                 `json:"chargebackWindowDays"   keel:"chargeback_window_days,label=Chargeback Window (Days),dimension"`
	Recourse               Recourse            `json:"recourse"               keel:"recourse"`
	AccountingTreatment    AccountingTreatment `json:"accountingTreatment"    keel:"accounting_treatment"`
	FundingDays            int                 `json:"fundingDays"            keel:"funding_days,dimension"`
	SubmissionTime         int                 `json:"submissionTime"         keel:"submission_time,noreport"`
	Notes                  string              `json:"notes"                  keel:"notes,nullzero"`
	DueFromFactorAccountID pulid.ID            `json:"dueFromFactorAccountId" keel:"due_from_factor_account_id,nullzero"`
	FeeExpenseAccountID    pulid.ID            `json:"feeExpenseAccountId"    keel:"fee_expense_account_id,nullzero,label=Factoring Fee Account ID"`
	LiabilityAccountID     pulid.ID            `json:"liabilityAccountId"     keel:"factoring_liability_account_id,nullzero,label=Factoring Liability Account ID"`
	CashAccountID          pulid.ID            `json:"cashAccountId"          keel:"cash_account_id,nullzero"`
	ActivatedAt            *int64              `json:"activatedAt"            keel:"activated_at"`
	EndedAt                *int64              `json:"endedAt"                keel:"ended_at"`
	Version                int64               `json:"version"                keel:"version,version"`
	CreatedAt              int64               `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64               `json:"updatedAt"              keel:"updated_at,autoupdate"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
	FeeTiers         []*FeeTier                         `json:"feeTiers"                   keel:"many:agreement_id,noreport"`
	DocumentTypes    []*documenttype.DocumentType       `json:"documentTypes"              keel:"m2m:factoring_agreement_document_types.agreement_id.document_type_id,noreport"`
}

type FeeTier struct {
	keel.Table `keel:"factoring_fee_tiers"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:facfee_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AgreementID    pulid.ID `json:"agreementId"    keel:"agreement_id"`
	DaysFrom       int      `json:"daysFrom"       keel:"days_from"`
	DaysTo         *int     `json:"daysTo"         keel:"days_to"`
	RateBP         int      `json:"rateBp"         keel:"rate_bp"`
	FlatMinor      int64    `json:"flatMinor"      keel:"flat_minor"`
}

type AgreementDocumentType struct {
	keel.Table `keel:"factoring_agreement_document_types"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AgreementID    pulid.ID `json:"agreementId"    keel:"agreement_id,pk"`
	DocumentTypeID pulid.ID `json:"documentTypeId" keel:"document_type_id,pk"`

	Agreement    *Agreement                 `json:"-" keel:"belongs:agreement_id"`
	DocumentType *documenttype.DocumentType `json:"-" keel:"belongs:document_type_id"`
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

func (a *Agreement) BeforeCreate(ctx context.Context) error {
	if a.Status == "" {
		a.Status = AgreementStatusDraft
	}
	if a.Recourse == "" {
		a.Recourse = RecourseFull
	}
	if a.AccountingTreatment == "" {
		a.AccountingTreatment = AccountingTreatmentSale
	}

	return nil
}

func (a *Agreement) GetID() pulid.ID             { return a.ID }
func (a *Agreement) GetOrganizationID() pulid.ID { return a.OrganizationID }
func (a *Agreement) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }
func (a *Agreement) GetTableName() string        { return "factoring_agreements" }
