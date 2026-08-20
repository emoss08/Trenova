package rateagreement

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementVersion)(nil)

// RateAgreementVersion records the negotiated header terms as they stood.
//
// It deliberately does not copy the rules. Rules carry their own effective
// windows, so "the contract as it stood on the third of March" is the header
// version live on that date plus whichever rules were live on that date — one
// query, and an amendment to forty lanes writes forty rows instead of copying
// every rule in the book.
type RateAgreementVersion struct {
	bun.BaseModel `bun:"table:rate_agreement_versions,alias:ragv" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull"`

	VersionNumber int64  `json:"versionNumber" bun:"version_number,type:BIGINT,notnull"`
	EffectiveFrom int64  `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo   *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero"`

	PartyType  PartyType `json:"partyType"  bun:"party_type,type:rate_agreement_party_type_enum,nullzero"`
	CustomerID *pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero"`
	CarrierID  *pulid.ID `json:"carrierId"  bun:"carrier_id,type:VARCHAR(100),nullzero"`

	Code          string        `json:"code"          bun:"code,type:VARCHAR(50),notnull,default:''"`
	Name          string        `json:"name"          bun:"name,type:VARCHAR(150),notnull"`
	Description   string        `json:"description"   bun:"description,type:TEXT,nullzero"`
	AgreementType AgreementType `json:"agreementType" bun:"agreement_type,type:rate_agreement_type_enum,notnull"`
	Status        Status        `json:"status"        bun:"status,type:rate_agreement_status_enum,notnull"`
	ContractRef   string        `json:"contractRef"   bun:"contract_ref,type:VARCHAR(100),nullzero"`
	DocumentID    *pulid.ID     `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`

	// The agreement's own window, distinct from EffectiveFrom/EffectiveTo above,
	// which say when this *version* of the terms governed. Moving the contract's
	// dates is a renegotiation like any other and diffs under these names.
	Priority               int16     `json:"priority"               bun:"priority,type:SMALLINT,notnull,default:0"`
	AgreementEffectiveFrom int64     `json:"agreementEffectiveFrom" bun:"agreement_effective_from,type:BIGINT,notnull,default:0"`
	AgreementEffectiveTo   *int64    `json:"agreementEffectiveTo"   bun:"agreement_effective_to,type:BIGINT,nullzero"`
	AutoRenew              bool      `json:"autoRenew"              bun:"auto_renew,type:BOOLEAN,notnull,default:false"`
	RenewalNoticeDays      int16     `json:"renewalNoticeDays"      bun:"renewal_notice_days,type:SMALLINT,notnull,default:30"`
	BillToCustomerID       *pulid.ID `json:"billToCustomerId"       bun:"bill_to_customer_id,type:VARCHAR(100),nullzero"`

	Currency          string                 `json:"currency"          bun:"currency,type:VARCHAR(3),notnull"`
	DefaultMinCharge  decimal.NullDecimal    `json:"defaultMinCharge"  bun:"default_min_charge,type:NUMERIC(19,4),nullzero"`
	DefaultMaxCharge  decimal.NullDecimal    `json:"defaultMaxCharge"  bun:"default_max_charge,type:NUMERIC(19,4),nullzero"`
	RoundingMode      ratetypes.RoundingMode `json:"roundingMode"      bun:"rounding_mode,type:rate_rounding_mode_enum,notnull"`
	RoundingPrecision int16                  `json:"roundingPrecision" bun:"rounding_precision,type:SMALLINT,notnull"`

	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent"  bun:"margin_floor_percent,type:NUMERIC(9,4),nullzero"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" bun:"max_pay_percent_of_sell,type:NUMERIC(9,4),nullzero"`

	// The negotiated schedule keyed by accessorial charge id, and the fuel
	// binding's terms. Stored as ids only — names are patched on at read time
	// into AccessorialNames, so a later rename of the charge cannot rewrite
	// what the contract said.
	AccessorialTerms map[string]AccessorialTermSnapshot `json:"accessorialTerms" bun:"accessorial_terms,type:JSONB,nullzero"`
	FuelTerms        *FuelTermSnapshot                  `json:"fuelTerms"        bun:"fuel_terms,type:JSONB,nullzero"`

	// Read-time only: accessorial charge id → code, resolved by the repository
	// for every id the snapshot and change summary mention. Never persisted.
	AccessorialNames map[string]string `json:"accessorialNames,omitempty" bun:"-"`

	ChangeMessage string                           `json:"changeMessage" bun:"change_message,type:TEXT,nullzero"`
	ChangeSummary map[string]jsonutils.FieldChange `json:"changeSummary" bun:"change_summary,type:JSONB,nullzero"`
	CreatedByID   pulid.ID                         `json:"createdById"   bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt     int64                            `json:"createdAt"     bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Agreement *RateAgreement `json:"-"                   bun:"rel:belongs-to,join:rate_agreement_id=id"`
	CreatedBy *tenant.User   `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id"`
}

// IsEffectiveAt reports whether this version's terms governed at a moment.
func (ragv *RateAgreementVersion) IsEffectiveAt(timestamp int64) bool {
	if timestamp < ragv.EffectiveFrom {
		return false
	}

	if ragv.EffectiveTo != nil && timestamp >= *ragv.EffectiveTo {
		return false
	}

	return true
}

// AccessorialTermSnapshot is one accessorial's negotiated terms as a version
// records them. The window fields are named appliesFrom/appliesTo rather than
// effectiveFrom/effectiveTo deliberately: the version diff ignores the latter
// pair because the version row uses those names for its own lifetime, and an
// accessorial's window is a term that must survive the diff.
type AccessorialTermSnapshot struct {
	Method            accessorialcharge.Method   `json:"method"`
	RateUnit          accessorialcharge.RateUnit `json:"rateUnit,omitempty"`
	Amount            decimal.Decimal            `json:"amount"`
	Waived            bool                       `json:"waived"`
	AutoApply         bool                       `json:"autoApply"`
	ApplyCondition    string                     `json:"applyCondition,omitempty"`
	FreeUnits         *int16                     `json:"freeUnits,omitempty"`
	MaxAmount         decimal.NullDecimal        `json:"maxAmount"`
	FormulaTemplateID *pulid.ID                  `json:"formulaTemplateId,omitempty"`
	ServiceTypeIDs    []pulid.ID                 `json:"serviceTypeIds,omitempty"`
	ShipmentTypeIDs   []pulid.ID                 `json:"shipmentTypeIds,omitempty"`
	AppliesFrom       *int64                     `json:"appliesFrom,omitempty"`
	AppliesTo         *int64                     `json:"appliesTo,omitempty"`
}

// FuelTermSnapshot is the fuel binding's negotiated terms as a version records
// them.
type FuelTermSnapshot struct {
	FuelSurchargeProgramID pulid.ID            `json:"fuelSurchargeProgramId"`
	Waived                 bool                `json:"waived"`
	PegPriceOverride       decimal.NullDecimal `json:"pegPriceOverride"`
	IncrementRateOverride  decimal.NullDecimal `json:"incrementRateOverride"`
	CapAmount              decimal.NullDecimal `json:"capAmount"`
}

// sortedIDs orders an applicability set so the snapshot compares by content:
// the same set sent in a different order is the same term, not a renegotiation.
func sortedIDs(ids []pulid.ID) []pulid.ID {
	if len(ids) == 0 {
		return nil
	}

	sorted := make([]pulid.ID, len(ids))
	copy(sorted, ids)
	slices.Sort(sorted)

	return sorted
}

func newAccessorialTermSnapshot(acc *RateAgreementAccessorial) AccessorialTermSnapshot {
	return AccessorialTermSnapshot{
		Method:            acc.Method,
		RateUnit:          acc.RateUnit,
		Amount:            acc.Amount,
		Waived:            acc.Waived,
		AutoApply:         acc.AutoApply,
		ApplyCondition:    acc.ApplyCondition,
		FreeUnits:         acc.FreeUnits,
		MaxAmount:         acc.MaxAmount,
		FormulaTemplateID: acc.FormulaTemplateID,
		ServiceTypeIDs:    sortedIDs(acc.ServiceTypeIDs),
		ShipmentTypeIDs:   sortedIDs(acc.ShipmentTypeIDs),
		AppliesFrom:       acc.EffectiveFrom,
		AppliesTo:         acc.EffectiveTo,
	}
}

func accessorialTermsFromAgreement(
	agreement *RateAgreement,
) map[string]AccessorialTermSnapshot {
	if len(agreement.Accessorials) == 0 {
		return nil
	}

	terms := make(map[string]AccessorialTermSnapshot, len(agreement.Accessorials))
	for _, acc := range agreement.Accessorials {
		terms[acc.AccessorialChargeID.String()] = newAccessorialTermSnapshot(acc)
	}

	return terms
}

func fuelTermsFromAgreement(agreement *RateAgreement) *FuelTermSnapshot {
	if agreement.FuelBinding == nil {
		return nil
	}

	return &FuelTermSnapshot{
		FuelSurchargeProgramID: agreement.FuelBinding.FuelSurchargeProgramID,
		Waived:                 agreement.FuelBinding.Waived,
		PegPriceOverride:       agreement.FuelBinding.PegPriceOverride,
		IncrementRateOverride:  agreement.FuelBinding.IncrementRateOverride,
		CapAmount:              agreement.FuelBinding.CapAmount,
	}
}

// NewVersionFromAgreement snapshots an agreement's current header terms.
func NewVersionFromAgreement(
	agreement *RateAgreement,
	versionNumber int64,
	effectiveFrom int64,
	createdByID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) *RateAgreementVersion {
	return &RateAgreementVersion{
		OrganizationID:         agreement.OrganizationID,
		BusinessUnitID:         agreement.BusinessUnitID,
		RateAgreementID:        agreement.ID,
		VersionNumber:          versionNumber,
		EffectiveFrom:          effectiveFrom,
		PartyType:              agreement.PartyType,
		CustomerID:             agreement.CustomerID,
		CarrierID:              agreement.CarrierID,
		Code:                   agreement.Code,
		Name:                   agreement.Name,
		Description:            agreement.Description,
		AgreementType:          agreement.AgreementType,
		Status:                 agreement.Status,
		ContractRef:            agreement.ContractRef,
		DocumentID:             agreement.DocumentID,
		Priority:               agreement.Priority,
		AgreementEffectiveFrom: agreement.EffectiveFrom,
		AgreementEffectiveTo:   agreement.EffectiveTo,
		AutoRenew:              agreement.AutoRenew,
		RenewalNoticeDays:      agreement.RenewalNoticeDays,
		BillToCustomerID:       agreement.BillToCustomerID,
		Currency:               agreement.Currency,
		DefaultMinCharge:       agreement.DefaultMinCharge,
		DefaultMaxCharge:       agreement.DefaultMaxCharge,
		RoundingMode:           agreement.RoundingMode,
		RoundingPrecision:      agreement.RoundingPrecision,
		MarginFloorPercent:     agreement.MarginFloorPercent,
		MaxPayPercentOfSell:    agreement.MaxPayPercentOfSell,
		AccessorialTerms:       accessorialTermsFromAgreement(agreement),
		FuelTerms:              fuelTermsFromAgreement(agreement),
		ChangeMessage:          changeMessage,
		ChangeSummary:          changeSummary,
		CreatedByID:            createdByID,
	}
}

func (ragv *RateAgreementVersion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(ragv,
		validation.Field(&ragv.RateAgreementID,
			validation.Required.Error("Agreement is required"),
		),
		validation.Field(&ragv.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(int64(1)).Error("Version number must be at least one"),
		),
		validation.Field(&ragv.EffectiveFrom,
			validation.Required.Error("Effective from is required"),
		),
		validation.Field(&ragv.Code, validation.Required.Error("Code is required")),
		validation.Field(&ragv.Name, validation.Required.Error("Name is required")),
		validation.Field(&ragv.AgreementEffectiveFrom,
			validation.Required.Error("Agreement effective from is required"),
		),
		validation.Field(&ragv.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&ragv.CreatedByID,
			validation.Required.Error("Created by is required"),
		),
	))

	if ragv.EffectiveTo != nil && *ragv.EffectiveTo <= ragv.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to must be after effective from",
		)
	}
}

func (ragv *RateAgreementVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if ragv.ID.IsNil() {
			ragv.ID = pulid.MustNew("ragv_")
		}
		ragv.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (ragv *RateAgreementVersion) GetID() pulid.ID {
	return ragv.ID
}

func (ragv *RateAgreementVersion) GetCreatedAt() int64 {
	return ragv.CreatedAt
}

func (ragv *RateAgreementVersion) GetOrganizationID() pulid.ID {
	return ragv.OrganizationID
}

func (ragv *RateAgreementVersion) GetBusinessUnitID() pulid.ID {
	return ragv.BusinessUnitID
}

func (ragv *RateAgreementVersion) GetTableName() string {
	return "rate_agreement_versions"
}
