package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_EvaluateExpression_ReadsRelatedRecordsOnARealShipment(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	entity := &shipment.Shipment{
		ServiceType:  &servicetype.ServiceType{Code: "EXP2D", Description: "Expedited"},
		ShipmentType: &shipmenttype.ShipmentType{Code: "TL", Description: "Truckload"},
		Customer:     &customer.Customer{Code: "ACME", Name: "Acme Logistics"},
		TractorType:  &equipmenttype.EquipmentType{Code: "DC", Description: "Day Cab"},
		TrailerType:  &equipmenttype.EquipmentType{Code: "DV", Description: "Dry Van"},
	}

	tests := []struct {
		name       string
		expression string
		variables  map[string]any
		want       string
	}{
		{
			name: "service type code drives a multiplier",
			expression: `let expedite =
  hasPrefix(serviceType.code, expediteCode) ? expediteMultiplier : 1;
round(100 * expedite, 2)`,
			variables: map[string]any{"expediteCode": "EXP", "expediteMultiplier": 1.5},
			want:      "150",
		},
		{
			name:       "service type description",
			expression: `serviceType.description == "Expedited" ? 1 : 0`,
			want:       "1",
		},
		{
			name:       "shipment type code",
			expression: `shipmentType.code == "TL" ? 1 : 0`,
			want:       "1",
		},
		{
			name:       "customer code and name",
			expression: `customer.code == "ACME" && customer.name == "Acme Logistics" ? 1 : 0`,
			want:       "1",
		},
		{
			name:       "equipment type codes and descriptions",
			expression: `tractorType.code == "DC" && trailerType.description == "Dry Van" ? 1 : 0`,
			want:       "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
				Expression: tt.expression,
				Entity:     entity,
				SchemaID:   "shipment",
				Variables:  tt.variables,
			})
			require.NoError(t, err)
			assert.True(
				t,
				decimal.RequireFromString(tt.want).Equal(resp.Amount),
				"expected %s, got %s",
				tt.want,
				resp.Amount,
			)
		})
	}
}

func TestService_EvaluateExpression_AbsentOptionalRelationReadsAsEmpty(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	entity := &shipment.Shipment{
		ServiceType: &servicetype.ServiceType{Code: "STD"},
	}

	resp, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
		Expression: `(trailerType?.code ?? "none") == "none" && shipmentType == nil ? 1 : 0`,
		Entity:     entity,
		SchemaID:   "shipment",
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1).Equal(resp.Amount), "got %s", resp.Amount)
}
