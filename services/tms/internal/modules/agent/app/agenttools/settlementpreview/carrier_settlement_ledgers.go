package settlementpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/shared/pulid"
)

const NounCarrierSettlement = "carrier settlement"

const carrierSettlementRead = "settlement_list_carrier or settlement_get_carrier"

const carrierSettlementEntity = "carrier_settlement"

var CarrierPaymentMethods = agenttoolschema.Source(
	"carrierSettlement.paymentMethod",
	[]string{
		PaymentMethodCheck,
		"ACHManual",
		PaymentMethodOther,
	},
)

func CarrierSettlementLedger() SettlementLedger[carriersettlement.CarrierSettlement] {
	return SettlementLedger[carriersettlement.CarrierSettlement]{
		Resource:  permission.ResourceCarrierSettlement,
		Kind:      recordkind.Kind(permission.ResourceCarrierSettlement),
		Artifact:  carrierSettlementEntity,
		Noun:      NounCarrierSettlement,
		Sources:   carrierSettlementRead,
		Sensitive: []string{"grossCostMinor", "adjustmentsMinor", "netPayableMinor"},
		Facts:     CarrierSettlementFacts,
		Editable:  (*carriersettlement.CarrierSettlement).IsEditable,
		Line:      carrierSettlementLine,
		Effects:   carrierSettlementEffects,
	}
}

func carrierSettlementLine(
	entity *carriersettlement.CarrierSettlement,
	lineID pulid.ID,
) (SettlementLineFacts, bool) {
	for _, line := range entity.Lines {
		if line != nil && line.ID == lineID {
			return SettlementLineFacts{
				AmountMinor:  line.AmountMinor,
				Manual:       line.IsManualAdjustment(),
				AddedByAgent: line.AddedByAgent,
			}, true
		}
	}

	return SettlementLineFacts{}, false
}

func CarrierSettlementFacts(
	entity *carriersettlement.CarrierSettlement,
) *SettlementFacts {
	facts := &SettlementFacts{
		ID:       entity.ID,
		Number:   entity.SettlementNumber,
		Version:  entity.Version,
		Status:   string(entity.Status),
		Currency: entity.CurrencyCode,
		Totals: []SettlementTotal{
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
