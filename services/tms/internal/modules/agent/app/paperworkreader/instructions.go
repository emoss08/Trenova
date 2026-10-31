package paperworkreader

import (
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/deliveryread"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
)

const maxCatalog = 60

const instructions = `Your task: read one piece of a shipment's billing paperwork and record the facts a biller checks before invoicing.

The file attached is a proof of delivery, a bill of lading, a delivery receipt, or a receipt for something paid out during the trip (a lumper, a scale ticket, a toll, a wash). Read every page.

Record:
- signedBy: the printed name of whoever signed for the freight at delivery. Empty when the signature line is blank.
- pieces: the count received, as written, with its unit ("22 pallets", "1,140 cartons").
- osd: "Over", "Short" or "Damaged" when the receiver noted one, "None" when the document says none was noted, empty when it does not say.
- osdNote: the receiver's own words about the exception.
- seal: the seal number at delivery, and "intact" or "broken" if written.
- amount: for a receipt, the total paid, digits only ("185.00"). Null for anything that is not a receipt.
- reference: the receipt's, ticket's or document's own number.
- chargeId: for a receipt, the id of the accessorial below that it bills as. Empty when none fits or it is not a receipt.
- documentKind: one of ProofOfDelivery, BillOfLading, DeliveryReceipt, LumperReceipt, ScaleTicket, OtherReceipt, Other.`

func schema() map[string]any {
	properties := deliveryread.Properties()
	maps.Copy(properties, map[string]any{
		"documentKind": map[string]any{
			"type": "string",
			"enum": []string{
				"ProofOfDelivery", "BillOfLading", "DeliveryReceipt", "LumperReceipt",
				"ScaleTicket", "OtherReceipt", "Other",
			},
		},
	})

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             append(deliveryread.Required(), "documentKind"),
		"additionalProperties": false,
	}
}

func catalogText(catalog []*accessorialcharge.AccessorialCharge) string {
	if len(catalog) == 0 {
		return "No accessorials are set up; leave chargeId empty."
	}
	if len(catalog) > maxCatalog {
		catalog = catalog[:maxCatalog]
	}
	var out strings.Builder
	for _, charge := range catalog {
		out.WriteString(charge.ID.String())
		out.WriteString(" — ")
		out.WriteString(charge.Code)
		if charge.Description != "" {
			out.WriteString(" · ")
			out.WriteString(charge.Description)
		}
		out.WriteString("\n")
	}

	return out.String()
}

func charges(catalog []*accessorialcharge.AccessorialCharge) map[string]deliveryread.Charge {
	out := make(map[string]deliveryread.Charge, len(catalog))
	for _, charge := range catalog {
		label := charge.Description
		if label == "" {
			label = charge.Code
		}
		out[charge.ID.String()] = deliveryread.Charge{ID: charge.ID, Label: label}
	}

	return out
}
