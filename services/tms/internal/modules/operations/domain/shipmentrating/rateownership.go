package shipmentrating

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

// RestoreSystemOwnedCharges re-seats the fields of an inbound charge set that
// the rating engines own rather than the caller: whether a charge was machine
// generated, and which engine owns it.
//
// No caller may claim any of them. A payload that drops the detention marker —
// an older client, an integration, any editor that rebuilds the list from the
// fields it cares about — would otherwise turn a detention charge into a manual
// one and leave the next recalculation free to bill the same dwell twice. The
// same is true of the agreement link: a contract accessorial that lost its
// owner would stop being reconciled and would be added again on every save.
//
// Exactly one owner column is set on any system charge, which the database
// enforces. That is what keeps three engines writing charges from ever billing
// the same thing twice.
func RestoreSystemOwnedCharges(original, updated []*shipment.AdditionalCharge) {
	originals := make(map[pulid.ID]*shipment.AdditionalCharge, len(original))
	for _, charge := range original {
		if charge == nil || charge.ID.IsNil() {
			continue
		}
		originals[charge.ID] = charge
	}

	for _, charge := range updated {
		if charge == nil {
			continue
		}

		source := originals[charge.ID]
		if charge.ID.IsNil() || source == nil {
			charge.IsSystemGenerated = false
			charge.IsDetention = false
			charge.IsTariff = false
			charge.RateAgreementAccessorialID = nil
			charge.FuelSurchargeProgramID = nil
			continue
		}

		charge.IsSystemGenerated = source.IsSystemGenerated
		charge.IsDetention = source.IsDetention
		charge.IsTariff = source.IsTariff
		charge.RateAgreementAccessorialID = source.RateAgreementAccessorialID
		charge.FuelSurchargeProgramID = source.FuelSurchargeProgramID
	}
}

func RestoreRateOwnedFields(original, updated *shipment.Shipment) {
	if original == nil || updated == nil {
		return
	}

	updated.RateOverrideAmount = original.RateOverrideAmount
	updated.RateOverrideByID = original.RateOverrideByID
	updated.RateOverrideAt = original.RateOverrideAt
	updated.RateLocked = original.RateLocked
	updated.RateQuoteID = original.RateQuoteID
	updated.RateAgreementID = original.RateAgreementID
	updated.RateAgreementRuleID = original.RateAgreementRuleID
	updated.AutoRated = original.AutoRated
	updated.AutoRatedAt = original.AutoRatedAt

	if updated.RateOverrideReason == "" {
		updated.RateOverrideReason = original.RateOverrideReason
	}

	updated.RatingDetail = original.RatingDetail
	updated.FreightLines = original.FreightLines
}

func RatedFieldsEdited(original, updated *shipment.Shipment) bool {
	if original == nil || updated == nil {
		return false
	}

	if original.FormulaTemplateID != updated.FormulaTemplateID {
		return true
	}

	if !original.TariffInputs.Equal(updated.TariffInputs) {
		return true
	}

	return agreementChargesEdited(original.AdditionalCharges, updated.AdditionalCharges)
}

func agreementChargesEdited(original, updated []*shipment.AdditionalCharge) bool {
	remaining := make(map[pulid.ID]*shipment.AdditionalCharge, len(updated))
	for _, charge := range updated {
		if charge == nil || charge.ID.IsNil() {
			continue
		}
		remaining[charge.ID] = charge
	}

	for _, charge := range original {
		if charge == nil || charge.Owner() != shipmentvocab.SystemOwnerAgreement {
			continue
		}

		current, kept := remaining[charge.ID]
		if !kept {
			return true
		}

		if current.Method != charge.Method || current.Unit != charge.Unit ||
			!current.Amount.Equal(charge.Amount) {
			return true
		}
	}

	return false
}
