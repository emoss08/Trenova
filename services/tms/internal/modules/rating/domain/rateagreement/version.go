package rateagreement

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
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
	keel.Table    `keel:"rate_agreement_versions"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:ragv_"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull" keel:"rate_agreement_id"`

	VersionNumber int64  `json:"versionNumber" bun:"version_number,type:BIGINT,notnull" keel:"version_number"`
	EffectiveFrom int64  `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull" keel:"effective_from"`
	EffectiveTo   *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero" keel:"effective_to"`

	PartyType  PartyType `json:"partyType"  bun:"party_type,type:rate_agreement_party_type_enum,nullzero" keel:"party_type,nullzero"`
	CustomerID *pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero" keel:"customer_id"`
	CarrierID  *pulid.ID `json:"carrierId"  bun:"carrier_id,type:VARCHAR(100),nullzero" keel:"carrier_id"`

	Code          string        `json:"code"          bun:"code,type:VARCHAR(50),notnull,default:''" keel:"code"`
	Name          string        `json:"name"          bun:"name,type:VARCHAR(150),notnull" keel:"name"`
	Description   string        `json:"description"   bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	AgreementType AgreementType `json:"agreementType" bun:"agreement_type,type:rate_agreement_type_enum,notnull" keel:"agreement_type"`
	Status        Status        `json:"status"        bun:"status,type:rate_agreement_status_enum,notnull" keel:"status"`
	ContractRef   string        `json:"contractRef"   bun:"contract_ref,type:VARCHAR(100),nullzero" keel:"contract_ref,nullzero"`
	DocumentID    *pulid.ID     `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero" keel:"document_id"`

	Priority int16 `json:"priority" bun:"priority,type:SMALLINT,notnull,default:0" keel:"priority"`
	// The agreement's own window, distinct from EffectiveFrom/EffectiveTo above,
	// which say when this *version* of the terms governed. Moving the contract's
	// dates is a renegotiation like any other and diffs under these names.
	AgreementEffectiveFrom int64     `json:"agreementEffectiveFrom" bun:"agreement_effective_from,type:BIGINT,notnull,default:0" keel:"agreement_effective_from"`
	AgreementEffectiveTo   *int64    `json:"agreementEffectiveTo"   bun:"agreement_effective_to,type:BIGINT,nullzero" keel:"agreement_effective_to"`
	AutoRenew              bool      `json:"autoRenew"              bun:"auto_renew,type:BOOLEAN,notnull,default:false" keel:"auto_renew"`
	RenewalNoticeDays      int16     `json:"renewalNoticeDays"      bun:"renewal_notice_days,type:SMALLINT,notnull,default:30" keel:"renewal_notice_days"`
	BillToCustomerID       *pulid.ID `json:"billToCustomerId"       bun:"bill_to_customer_id,type:VARCHAR(100),nullzero" keel:"bill_to_customer_id"`

	Currency string             `json:"currency" bun:"currency,type:VARCHAR(3),notnull" keel:"currency"`
	Inputs   tarifftypes.Inputs `json:"inputs"   bun:"inputs,type:JSONB,notnull,default:'{}'" keel:"inputs,jsonb"`

	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent"  bun:"margin_floor_percent,type:NUMERIC(9,4),nullzero" keel:"margin_floor_percent"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" bun:"max_pay_percent_of_sell,type:NUMERIC(9,4),nullzero" keel:"max_pay_percent_of_sell"`

	// The negotiated accessorial schedule keyed by accessorial charge id.
	// Stored as ids only — names are patched on at read time into
	// AccessorialNames, so a later rename of the charge cannot rewrite what
	// the contract said.
	AccessorialTerms map[string]AccessorialTermSnapshot `json:"accessorialTerms" bun:"accessorial_terms,type:JSONB,nullzero" keel:"accessorial_terms,jsonb,nullzero"`
	// The fuel binding's negotiated terms as they stood.
	FuelTerms *FuelTermSnapshot `json:"fuelTerms" bun:"fuel_terms,type:JSONB,nullzero" keel:"fuel_terms,jsonb"`

	// Read-time only: accessorial charge id → code, resolved by the repository
	// for every id the snapshot and change summary mention. Never persisted.
	AccessorialNames map[string]string `json:"accessorialNames,omitempty" bun:"-"`

	ChangeMessage string                           `json:"changeMessage" bun:"change_message,type:TEXT,nullzero" keel:"change_message,nullzero"`
	ChangeSummary map[string]jsonutils.FieldChange `json:"changeSummary" bun:"change_summary,type:JSONB,nullzero" keel:"change_summary,jsonb,nullzero"`
	CreatedByID   pulid.ID                         `json:"createdById"   bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	CreatedAt     int64                            `json:"createdAt"     bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	Agreement *RateAgreement `json:"-"                   bun:"rel:belongs-to,join:rate_agreement_id=id" keel:"belongs:rate_agreement_id"`
	CreatedBy *tenant.User   `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id" keel:"belongs:created_by_id"`
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
	Method          accessorialcharge.Method   `json:"method"`
	RateUnit        accessorialcharge.RateUnit `json:"rateUnit,omitempty"`
	Amount          decimal.Decimal            `json:"amount"`
	Waived          bool                       `json:"waived"`
	AutoApply       bool                       `json:"autoApply"`
	Condition       string                     `json:"condition,omitempty"`
	FreeUnits       *int16                     `json:"freeUnits,omitempty"`
	MaxAmount       decimal.NullDecimal        `json:"maxAmount"`
	ServiceTypeIDs  []pulid.ID                 `json:"serviceTypeIds,omitempty"`
	ShipmentTypeIDs []pulid.ID                 `json:"shipmentTypeIds,omitempty"`
	AppliesFrom     *int64                     `json:"appliesFrom,omitempty"`
	AppliesTo       *int64                     `json:"appliesTo,omitempty"`
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
		Method:          acc.Method,
		RateUnit:        acc.RateUnit,
		Amount:          acc.Amount,
		Waived:          acc.Waived,
		AutoApply:       acc.AutoApply,
		Condition:       acc.Condition,
		FreeUnits:       acc.FreeUnits,
		MaxAmount:       acc.MaxAmount,
		ServiceTypeIDs:  sortedIDs(acc.ServiceTypeIDs),
		ShipmentTypeIDs: sortedIDs(acc.ShipmentTypeIDs),
		AppliesFrom:     acc.EffectiveFrom,
		AppliesTo:       acc.EffectiveTo,
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
		Inputs:                 agreement.Inputs.Clone(),
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
	ragv.Inputs = ragv.Inputs.OrEmpty()
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
