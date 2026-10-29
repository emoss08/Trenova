package settlementpreview

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
)

const NounDriverSettlement = "driver settlement"

const DriverSettlementReads = "settlement_list_driver or settlement_get_driver"

const PaymentMethodCheck = "Check"

const PaymentMethodOther = "Other"

var DriverPaymentMethods = agenttoolschema.Source(
	"driverSettlement.paymentMethod",
	[]string{"ACH", PaymentMethodCheck, "InstantPay", PaymentMethodOther},
)

func DriverSettlementLedger() SettlementLedger[driversettlement.Settlement] {
	return SettlementLedger[driversettlement.Settlement]{
		Resource: permission.ResourceDriverSettlement,
		Kind:     recordkind.Kind(permission.ResourceDriverSettlement),
		Artifact: DriverSettlementEntity,
		Noun:     NounDriverSettlement,
		Sources:  DriverSettlementReads,
		ByName: " settlement_list_driver finds it by the driver's name as query, with no " +
			"lookup before it.",
		Sensitive: []string{
			"grossEarningsMinor",
			"reimbursementsMinor",
			"deductionsMinor",
			"netPayMinor",
		},
		Facts:    DriverSettlementFacts,
		Editable: (*driversettlement.Settlement).IsEditable,
		Line:     driverSettlementLine,
		Effects:  driverSettlementEffects,
	}
}

func driverSettlementLine(
	entity *driversettlement.Settlement,
	lineID pulid.ID,
) (SettlementLineFacts, bool) {
	for _, line := range entity.Lines {
		if line != nil && line.ID == lineID {
			return SettlementLineFacts{
				AmountMinor:  line.AmountMinor,
				Manual:       line.Category == driversettlement.LineCategoryAdjustment,
				AddedByAgent: line.AddedByAgent,
			}, true
		}
	}

	return SettlementLineFacts{}, false
}

func DriverSettlementFacts(entity *driversettlement.Settlement) *SettlementFacts {
	facts := &SettlementFacts{
		ID:       entity.ID,
		Number:   entity.SettlementNumber,
		Version:  entity.Version,
		Status:   string(entity.Status),
		Currency: entity.CurrencyCode,
		Totals: []SettlementTotal{
			{Label: "Earnings", Minor: entity.GrossEarningsMinor},
			{Label: "Reimbursements", Minor: entity.ReimbursementsMinor},
			{Label: "Deductions", Minor: -entity.DeductionsMinor},
			{Label: "Carried in", Minor: entity.CarryForwardInMinor},
		},
	}
	if entity.Worker != nil {
		facts.Payee = entity.Worker.FullName()
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
		changepreview.CountOf(deductions, "recurring deduction"),
		changepreview.CountOf(earnings, "recurring earning"),
		changepreview.CountOf(escrow, "escrow contribution"),
		changepreview.CountOf(advances, "advance recovery"),
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
