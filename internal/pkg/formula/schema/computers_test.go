package schema_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterShipmentComputers(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()

	// Should not panic
	require.NotPanics(t, func() {
		schema.RegisterShipmentComputers(resolver)
	})

	// Test that computers are registered
	expectedComputers := []string{
		"computeTemperatureDifferential",
		"computeHasHazmat",
		"computeRequiresTemperatureControl",
		"computeTotalStops",
	}

	for _, computerName := range expectedComputers {
		// Test with a dummy shipment to verify registration
		_, err := resolver.ResolveComputed(&shipment.Shipment{}, &schema.FieldSource{
			Computed: true,
			Function: computerName,
		})
		// Should not error about missing computer
		if err != nil {
			assert.NotContains(t, err.Error(), "no computer registered")
		}
	}
}

func TestComputeTemperatureDifferential(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	tests := []struct {
		name     string
		shipment *shipment.Shipment
		expected float64
	}{
		{
			name: "valid temperature range",
			shipment: &shipment.Shipment{
				TemperatureMin: int16Ptr(-10),
				TemperatureMax: int16Ptr(32),
			},
			expected: 42.0, // 32 - (-10)
		},
		{
			name: "same min and max",
			shipment: &shipment.Shipment{
				TemperatureMin: int16Ptr(20),
				TemperatureMax: int16Ptr(20),
			},
			expected: 0.0,
		},
		{
			name: "nil temperature min",
			shipment: &shipment.Shipment{
				TemperatureMax: int16Ptr(32),
			},
			expected: 0.0,
		},
		{
			name: "nil temperature max",
			shipment: &shipment.Shipment{
				TemperatureMin: int16Ptr(-10),
			},
			expected: 0.0,
		},
		{
			name:     "both nil",
			shipment: &shipment.Shipment{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveComputed(tt.shipment, &schema.FieldSource{
				Computed: true,
				Function: "computeTemperatureDifferential",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test with non-shipment entity
	t.Run("non-shipment entity", func(t *testing.T) {
		result, err := resolver.ResolveComputed(struct{}{}, &schema.FieldSource{
			Computed: true,
			Function: "computeTemperatureDifferential",
		})
		require.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})
}

func TestComputeHasHazmat(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	tests := []struct {
		name     string
		shipment *shipment.Shipment
		expected bool
	}{
		{
			name:     "shipment with hazmat",
			shipment: createShipmentWithHazmat(),
			expected: true,
		},
		{
			name:     "shipment without hazmat",
			shipment: createShipmentWithoutHazmat(),
			expected: false,
		},
		{
			name:     "empty commodities",
			shipment: &shipment.Shipment{},
			expected: false,
		},
		{
			name: "commodity without hazmat",
			shipment: &shipment.Shipment{
				Commodities: []*shipment.ShipmentCommodity{
					{
						Commodity: &commodity.Commodity{
							Name: "Regular Freight",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "nil commodity",
			shipment: &shipment.Shipment{
				Commodities: []*shipment.ShipmentCommodity{
					{
						Commodity: nil,
					},
				},
			},
			expected: false,
		},
		{
			name: "multiple commodities with one hazmat",
			shipment: &shipment.Shipment{
				Commodities: []*shipment.ShipmentCommodity{
					{
						Commodity: &commodity.Commodity{
							Name: "Regular Freight",
						},
					},
					{
						Commodity: &commodity.Commodity{
							Name:                "Hazmat",
							HazardousMaterialID: &[]pulid.ID{pulid.MustNew("hm_")}[0],
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveComputed(tt.shipment, &schema.FieldSource{
				Computed: true,
				Function: "computeHasHazmat",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test with non-shipment entity
	t.Run("non-shipment entity", func(t *testing.T) {
		result, err := resolver.ResolveComputed(struct{}{}, &schema.FieldSource{
			Computed: true,
			Function: "computeHasHazmat",
		})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

func TestComputeRequiresTemperatureControl(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	tests := []struct {
		name     string
		shipment *shipment.Shipment
		expected bool
	}{
		{
			name: "has temperature min",
			shipment: &shipment.Shipment{
				TemperatureMin: int16Ptr(0),
			},
			expected: true,
		},
		{
			name: "has temperature max",
			shipment: &shipment.Shipment{
				TemperatureMax: int16Ptr(32),
			},
			expected: true,
		},
		{
			name: "has both temperatures",
			shipment: &shipment.Shipment{
				TemperatureMin: int16Ptr(-10),
				TemperatureMax: int16Ptr(32),
			},
			expected: true,
		},
		{
			name:     "no temperature control",
			shipment: &shipment.Shipment{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveComputed(tt.shipment, &schema.FieldSource{
				Computed: true,
				Function: "computeRequiresTemperatureControl",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeTotalStops(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	tests := []struct {
		name     string
		shipment *shipment.Shipment
		expected int
	}{
		{
			name: "single move with two stops",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{
							{Type: shipment.StopTypePickup},
							{Type: shipment.StopTypeDelivery},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "multiple moves",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{
							{Type: shipment.StopTypePickup},
							{Type: shipment.StopTypeDelivery},
						},
					},
					{
						Stops: []*shipment.Stop{
							{Type: shipment.StopTypePickup},
							{Type: shipment.StopTypeDelivery},
							{Type: shipment.StopTypeSplitPickup},
						},
					},
				},
			},
			expected: 5,
		},
		{
			name:     "no moves",
			shipment: &shipment.Shipment{},
			expected: 0,
		},
		{
			name: "moves with no stops",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{},
					},
				},
			},
			expected: 0,
		},
		{
			name: "nil stops",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: nil,
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveComputed(tt.shipment, &schema.FieldSource{
				Computed: true,
				Function: "computeTotalStops",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions
func int16Ptr(v int16) *int16 {
	return &v
}

func createShipmentWithHazmat() *shipment.Shipment {
	hazmatID := pulid.MustNew("hm_")
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Gasoline",
					HazardousMaterialID: &hazmatID,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID,
						Name:         "Gasoline",
						Class:        hazardousmaterial.HazardousClass3,
						PackingGroup: hazardousmaterial.PackingGroupII,
						UNNumber:     "UN1203",
					},
				},
			},
		},
	}
}

func createShipmentWithoutHazmat() *shipment.Shipment {
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:   pulid.MustNew("com_"),
					Name: "General Freight",
				},
			},
		},
	}
}
