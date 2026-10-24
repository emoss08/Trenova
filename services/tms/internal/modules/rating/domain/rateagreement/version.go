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
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// What a contract said at each revision, and why it changed
type RateAgreementVersion struct {
	keel.Table `                                                        keel:"rate_agreement_versions,report=Billing,label=Agreement Version"`

	ID              pulid.ID `json:"id" keel:"id,pk,autoid:ragv_"`
	BusinessUnitID  pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RateAgreementID pulid.ID `json:"rateAgreementId" keel:"rate_agreement_id"`

	VersionNumber int64  `json:"versionNumber" keel:"version_number,label=Version,dimension"`
	EffectiveFrom int64  `json:"effectiveFrom" keel:"effective_from,type=epoch"`
	EffectiveTo   *int64 `json:"effectiveTo" keel:"effective_to,type=epoch"`

	PartyType  PartyType `json:"partyType" keel:"party_type,nullzero"`
	CustomerID *pulid.ID `json:"customerId" keel:"customer_id"`
	CarrierID  *pulid.ID `json:"carrierId" keel:"carrier_id"`

	Code          string        `json:"code" keel:"code"`
	Name          string        `json:"name" keel:"name"`
	Description   string        `json:"description" keel:"description,nullzero"`
	AgreementType AgreementType `json:"agreementType" keel:"agreement_type"`
	Status        Status        `json:"status" keel:"status"`
	ContractRef   string        `json:"contractRef" keel:"contract_ref,nullzero"`
	DocumentID    *pulid.ID     `json:"documentId" keel:"document_id"`

	Priority               int16     `json:"priority" keel:"priority,dimension"`
	AgreementEffectiveFrom int64     `json:"agreementEffectiveFrom" keel:"agreement_effective_from,type=epoch"`
	AgreementEffectiveTo   *int64    `json:"agreementEffectiveTo" keel:"agreement_effective_to,type=epoch"`
	AutoRenew              bool      `json:"autoRenew" keel:"auto_renew"`
	RenewalNoticeDays      int16     `json:"renewalNoticeDays" keel:"renewal_notice_days,dimension"`
	BillToCustomerID       *pulid.ID `json:"billToCustomerId" keel:"bill_to_customer_id"`

	Currency string             `json:"currency" keel:"currency"`
	Inputs   tarifftypes.Inputs `json:"inputs" keel:"inputs,jsonb"`

	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent" keel:"margin_floor_percent,label=Margin Floor (%)"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" keel:"max_pay_percent_of_sell,label=Max Pay % of Sell"`

	AccessorialTerms map[string]AccessorialTermSnapshot `json:"accessorialTerms" keel:"accessorial_terms,jsonb,nullzero"`
	FuelTerms        *FuelTermSnapshot                  `json:"fuelTerms" keel:"fuel_terms,jsonb"`

	AccessorialNames map[string]string `json:"accessorialNames,omitempty"`

	ChangeMessage string                           `json:"changeMessage" keel:"change_message,nullzero"`
	ChangeSummary map[string]jsonutils.FieldChange `json:"changeSummary" keel:"change_summary,jsonb,nullzero"`
	CreatedByID   pulid.ID                         `json:"createdById" keel:"created_by_id"`
	CreatedAt     int64                            `json:"createdAt" keel:"created_at,autocreate"`

	Agreement *RateAgreement `json:"-" keel:"belongs:rate_agreement_id"`
	CreatedBy *tenant.User   `json:"createdBy,omitempty" keel:"belongs:created_by_id,noreport"`
}

func (ragv *RateAgreementVersion) BeforeUpdate(_ context.Context) error {
	ragv.Inputs = ragv.Inputs.OrEmpty()

	return nil
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

func (ragv *RateAgreementVersion) BeforeCreate(_ context.Context) error {
	ragv.Inputs = ragv.Inputs.OrEmpty()
	if ragv.RenewalNoticeDays == 0 {
		ragv.RenewalNoticeDays = 30
	}
	ragv.Inputs = ragv.Inputs.OrEmpty()

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
