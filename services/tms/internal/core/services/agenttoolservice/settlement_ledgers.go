package agenttoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
)

const (
	nounDriverSettlement  = "driver settlement"
	nounCarrierSettlement = "carrier settlement"
	driverSettlementReads = "list_driver_settlements or get_driver_settlement"
	carrierSettlementRead = "list_carrier_settlements or get_carrier_settlement"
)

const (
	paymentMethodCheck = "Check"
	paymentMethodOther = "Other"
)

var (
	driverPaymentMethods  = []string{"ACH", paymentMethodCheck, "InstantPay", paymentMethodOther}
	carrierPaymentMethods = []string{paymentMethodCheck, "ACHManual", paymentMethodOther}
)

func driverSettlementLedger() settlementLedger[driversettlement.Settlement] {
	return settlementLedger[driversettlement.Settlement]{
		resource: permission.ResourceDriverSettlement,
		noun:     nounDriverSettlement,
		sources:  driverSettlementReads,
		sensitive: []string{
			"grossEarningsMinor",
			"reimbursementsMinor",
			"deductionsMinor",
			"netPayMinor",
		},
		facts:   driverSettlementFacts,
		effects: driverSettlementEffects,
	}
}

func driverSettlementFacts(entity *driversettlement.Settlement) *settlementFacts {
	facts := &settlementFacts{
		id:       entity.ID,
		number:   entity.SettlementNumber,
		version:  entity.Version,
		status:   string(entity.Status),
		currency: entity.CurrencyCode,
		totals: []settlementTotal{
			{label: "Earnings", minor: entity.GrossEarningsMinor},
			{label: "Reimbursements", minor: entity.ReimbursementsMinor},
			{label: "Deductions", minor: -entity.DeductionsMinor},
			{label: "Carried in", minor: entity.CarryForwardInMinor},
		},
	}
	if entity.Worker != nil {
		facts.payee = entity.Worker.FullName()
	}

	return facts
}

func driverSettlementEffects(
	action settlementshared.Action,
	plan *settlementshared.ActionPlan[*driversettlement.Settlement],
) []string {
	switch action { //nolint:exhaustive // only these actions reach beyond the settlement itself
	case settlementshared.ActionApprove:
		return []string{approvalEffects(plan.Before)}
	case settlementshared.ActionPost:
		return []string{
			"It is queued for the accounting system as a bill, and the driver is told in " +
				"the driver portal that it was posted.",
		}
	case settlementshared.ActionMarkPaid:
		return []string{
			"It is queued for the accounting system as a bill payment, and the driver is " +
				"told in the driver portal that it was paid.",
		}
	case settlementshared.ActionVoid:
		return []string{voidEffects(plan.Before)}
	case settlementshared.ActionRecalculate:
		return []string{
			"It is rebuilt from the pay events accrued for its period and the pay profile " +
				"in force; manual adjustments are kept.",
		}
	default:
		return nil
	}
}

func approvalEffects(entity *driversettlement.Settlement) string {
	var deductions, escrow, advances, earnings int
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		switch line.Category { //nolint:exhaustive // the categories approval applies to a balance
		case driversettlement.LineCategoryDeduction:
			if line.RecurringDeductionID != nil {
				deductions++
			}
		case driversettlement.LineCategoryEscrowContribution:
			escrow++
		case driversettlement.LineCategoryAdvanceRecovery:
			advances++
		case driversettlement.LineCategoryEarning, driversettlement.LineCategoryReimbursement:
			if line.RecurringEarningID != nil {
				earnings++
			}
		}
	}
	if deductions+escrow+advances+earnings == 0 {
		return "Approving applies no recurring deduction, escrow contribution or advance recovery."
	}

	return fmt.Sprintf(
		"Approving applies %s, %s, %s and %s to their running balances.",
		countOf(deductions, "recurring deduction"),
		countOf(earnings, "recurring earning"),
		countOf(escrow, "escrow contribution"),
		countOf(advances, "advance recovery"),
	)
}

func voidEffects(entity *driversettlement.Settlement) string {
	switch entity.Status { //nolint:exhaustive // the statuses a void reverses more than the status of
	case driversettlement.StatusPosted:
		return "Its journal entry is reversed, the deductions, escrow contributions and " +
			"advance recoveries it applied are undone, and its pay events return to the " +
			"unsettled pool."
	case driversettlement.StatusApproved:
		return "The deductions, escrow contributions and advance recoveries it applied are " +
			"undone, and its pay events return to the unsettled pool."
	default:
		return "Its pay events return to the unsettled pool."
	}
}

func carrierSettlementLedger() settlementLedger[carriersettlement.CarrierSettlement] {
	return settlementLedger[carriersettlement.CarrierSettlement]{
		resource:  permission.ResourceCarrierSettlement,
		noun:      nounCarrierSettlement,
		sources:   carrierSettlementRead,
		sensitive: []string{"grossCostMinor", "adjustmentsMinor", "netPayableMinor"},
		facts:     carrierSettlementFacts,
		effects:   carrierSettlementEffects,
	}
}

func carrierSettlementFacts(entity *carriersettlement.CarrierSettlement) *settlementFacts {
	facts := &settlementFacts{
		id:       entity.ID,
		number:   entity.SettlementNumber,
		version:  entity.Version,
		status:   string(entity.Status),
		currency: entity.CurrencyCode,
		totals: []settlementTotal{
			{label: "Freight cost", minor: entity.GrossCostMinor},
			{label: "Adjustments", minor: entity.AdjustmentsMinor},
		},
	}
	if entity.Carrier != nil {
		facts.payee = entity.Carrier.Name
	}

	return facts
}

func carrierSettlementEffects(
	action settlementshared.Action,
	plan *settlementshared.ActionPlan[*carriersettlement.CarrierSettlement],
) []string {
	switch action { //nolint:exhaustive // only these actions reach beyond the settlement itself
	case settlementshared.ActionPost:
		return []string{
			"Its cost events are marked settled, the carrier's ledger records the bill and " +
				"it is queued for the accounting system.",
		}
	case settlementshared.ActionMarkPaid:
		return []string{
			"The carrier's ledger records the payment and it is queued for the accounting " +
				"system as a bill payment.",
		}
	case settlementshared.ActionVoid:
		if plan.Before.Status == carriersettlement.StatusPosted {
			return []string{
				"Its journal entry is reversed, the carrier's ledger records the reversal and " +
					"its cost events return to the pending pool.",
			}
		}

		return []string{"Its cost events return to the pending pool."}
	case settlementshared.ActionRecalculate:
		return []string{
			"It is rebuilt from the carrier's pending cost events for its period; manual " +
				"adjustments are kept.",
		}
	default:
		return nil
	}
}
