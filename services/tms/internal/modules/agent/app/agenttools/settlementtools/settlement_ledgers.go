package settlementtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/shared/pulid"
)

const nounCarrierSettlement = "carrier settlement"

const carrierSettlementRead = "list_carrier_settlements or get_carrier_settlement"

const carrierSettlementEntity = "carrier_settlement"

var carrierPaymentMethods = agenttoolschema.Source(
	"carrierSettlement.paymentMethod",
	[]string{common.PaymentMethodCheck, "ACHManual", common.PaymentMethodOther},
)

func carrierSettlementLedger() common.SettlementLedger[carriersettlement.CarrierSettlement] {
	return common.SettlementLedger[carriersettlement.CarrierSettlement]{
		Resource:  permission.ResourceCarrierSettlement,
		Kind:      permission.RecordKind(permission.ResourceCarrierSettlement),
		Artifact:  carrierSettlementEntity,
		Noun:      nounCarrierSettlement,
		Sources:   carrierSettlementRead,
		Sensitive: []string{"grossCostMinor", "adjustmentsMinor", "netPayableMinor"},
		Facts:     carrierSettlementFacts,
		Editable:  (*carriersettlement.CarrierSettlement).IsEditable,
		Line:      carrierSettlementLine,
		Effects:   carrierSettlementEffects,
	}
}

func carrierSettlementLine(
	entity *carriersettlement.CarrierSettlement,
	lineID pulid.ID,
) (common.SettlementLineFacts, bool) {
	for _, line := range entity.Lines {
		if line != nil && line.ID == lineID {
			return common.SettlementLineFacts{
				AmountMinor:  line.AmountMinor,
				Manual:       line.IsManualAdjustment(),
				AddedByAgent: line.AddedByAgent,
			}, true
		}
	}

	return common.SettlementLineFacts{}, false
}

func carrierSettlementFacts(entity *carriersettlement.CarrierSettlement) *common.SettlementFacts {
	facts := &common.SettlementFacts{
		ID:       entity.ID,
		Number:   entity.SettlementNumber,
		Version:  entity.Version,
		Status:   string(entity.Status),
		Currency: entity.CurrencyCode,
		Totals: []common.SettlementTotal{
			{Label: "Freight cost", Minor: entity.GrossCostMinor},
			{Label: "Adjustments", Minor: entity.AdjustmentsMinor},
		},
	}
	if entity.Carrier != nil {
		facts.Payee = entity.Carrier.Name
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
