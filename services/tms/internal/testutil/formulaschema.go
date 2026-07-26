package testutil

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/stretchr/testify/require"
)

// RegisterShipmentFormulaSchema registers the shipment formula schema every
// integration suite that evaluates shipment formulas depends on. Keeping one
// copy means the schema cannot drift between packages.
func RegisterShipmentFormulaSchema(t *testing.T, registry *schema.Registry) {
	t.Helper()

	const shipmentSchema = `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://trenova.com/schemas/formula/shipment.schema.json",
		"type": "object",
		"x-formula-context": {
			"category": "shipment",
			"entities": ["Shipment"]
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Customer", "Moves.Stops", "Commodities.Commodity", "Commodities.Commodity.HazardousMaterial"]
		},
		"properties": {
			"customer": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"x-source": { "path": "Customer.Name" }
					},
					"code": {
						"type": "string",
						"x-source": { "path": "Customer.Code" }
					}
				}
			},
			"weight": {
				"type": ["number", "null"],
				"x-source": { "path": "Weight", "nullable": true, "transform": "int64ToFloat64" }
			},
			"pieces": {
				"type": ["integer", "null"],
				"x-source": { "path": "Pieces", "nullable": true }
			},
			"freightChargeAmount": {
				"type": "number",
				"x-source": { "path": "FreightChargeAmount", "transform": "decimalToFloat64" }
			},
			"otherChargeAmount": {
				"type": "number",
				"x-source": { "path": "OtherChargeAmount", "transform": "decimalToFloat64" }
			},
			"currentTotalCharge": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeCurrentTotalCharge" }
			},
			"ratingUnit": {
				"type": "integer",
				"x-source": { "path": "RatingUnit" }
			},
			"totalDistance": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalDistance" }
			},
			"totalStops": {
				"type": "integer",
				"x-source": { "computed": true, "function": "computeTotalStops" }
			},
			"totalWeight": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalWeight" }
			},
			"totalPieces": {
				"type": "integer",
				"x-source": { "computed": true, "function": "computeTotalPieces" }
			},
			"totalLinearFeet": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalLinearFeet" }
			},
			"hasHazmat": {
				"type": "boolean",
				"x-source": { "computed": true, "function": "computeHasHazmat" }
			},
			"requiresTemperatureControl": {
				"type": "boolean",
				"x-source": { "computed": true, "function": "computeRequiresTemperatureControl" }
			},
			"temperatureDifferential": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTemperatureDifferential" }
			}
		}
	}`

	require.NoError(t, registry.Register("shipment", []byte(shipmentSchema)))
}
