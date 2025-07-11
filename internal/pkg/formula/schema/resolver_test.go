package schema_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDataResolver_ResolveField(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()

	tests := []struct {
		name        string
		entity      any
		fieldSource *schema.FieldSource
		want        any
		wantErr     bool
	}{
		{
			name: "resolve simple string field",
			entity: &shipment.Shipment{
				ProNumber: "PRO123456",
			},
			fieldSource: &schema.FieldSource{
				Path: "ProNumber",
			},
			want:    "PRO123456",
			wantErr: false,
		},
		{
			name: "resolve nullable int64 field with value",
			entity: &shipment.Shipment{
				Weight: func() *int64 { v := int64(50000); return &v }(),
			},
			fieldSource: &schema.FieldSource{
				Path:      "Weight",
				Transform: "int64ToFloat64",
			},
			want:    50000.0,
			wantErr: false,
		},
		{
			name: "resolve nullable int64 field without value",
			entity: &shipment.Shipment{
				Weight: nil,
			},
			fieldSource: &schema.FieldSource{
				Path:      "Weight",
				Transform: "int64ToFloat64",
			},
			want:    0.0,
			wantErr: false,
		},
		{
			name: "resolve decimal field",
			entity: &shipment.Shipment{
				FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromFloat(1500.50)),
			},
			fieldSource: &schema.FieldSource{
				Path:      "FreightChargeAmount",
				Transform: "decimalToFloat64",
			},
			want:    1500.50,
			wantErr: false,
		},
		{
			name: "resolve nested field",
			entity: &shipment.Shipment{
				Customer: &customer.Customer{
					Name: "Test Customer Inc",
				},
			},
			fieldSource: &schema.FieldSource{
				Path: "Customer.Name",
			},
			want:    "Test Customer Inc",
			wantErr: false,
		},
		{
			name: "resolve nested field with nil parent",
			entity: &shipment.Shipment{
				Customer: nil,
			},
			fieldSource: &schema.FieldSource{
				Path: "Customer.Name",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:   "field not found",
			entity: &shipment.Shipment{},
			fieldSource: &schema.FieldSource{
				Path: "NonExistentField",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveField(tt.entity, tt.fieldSource)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDefaultDataResolver_Transforms(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()

	t.Run("int16ToFloat64", func(t *testing.T) {
		// Test with value
		tempMin := int16(32)
		entity := &shipment.Shipment{
			TemperatureMin: &tempMin,
		}
		fieldSource := &schema.FieldSource{
			Path:      "TemperatureMin",
			Transform: "int16ToFloat64",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 32.0, result)

		// Test with nil
		entity.TemperatureMin = nil
		result, err = resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})

	t.Run("decimalToFloat64 with NullDecimal", func(t *testing.T) {
		entity := &shipment.Shipment{
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(999.99),
				Valid:   true,
			},
		}
		fieldSource := &schema.FieldSource{
			Path:      "FreightChargeAmount",
			Transform: "decimalToFloat64",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 999.99, result)

		// Test with invalid NullDecimal
		entity.FreightChargeAmount = decimal.NullDecimal{Valid: false}
		result, err = resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 0.0, result)
	})
}

func TestDefaultDataResolver_ComputedFields(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	t.Run("computeTemperatureDifferential", func(t *testing.T) {
		tempMin := int16(32)
		tempMax := int16(78)
		entity := &shipment.Shipment{
			TemperatureMin: &tempMin,
			TemperatureMax: &tempMax,
		}
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeTemperatureDifferential",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 46.0, result) // 78 - 32 = 46
	})

	t.Run("computeHasHazmat with hazmat", func(t *testing.T) {
		hazmatID := pulid.MustNew("hm_")
		entity := &shipment.Shipment{
			Commodities: []*shipment.ShipmentCommodity{
				{
					Commodity: &commodity.Commodity{
						Name:                "Chemical A",
						HazardousMaterialID: &hazmatID,
					},
				},
				{
					Commodity: &commodity.Commodity{
						Name:                "Regular Cargo",
						HazardousMaterialID: nil,
					},
				},
			},
		}
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeHasHazmat",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("computeHasHazmat without hazmat", func(t *testing.T) {
		entity := &shipment.Shipment{
			Commodities: []*shipment.ShipmentCommodity{
				{
					Commodity: &commodity.Commodity{
						Name:                "Regular Cargo 1",
						HazardousMaterialID: nil,
					},
				},
				{
					Commodity: &commodity.Commodity{
						Name:                "Regular Cargo 2",
						HazardousMaterialID: nil,
					},
				},
			},
		}
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeHasHazmat",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("computeTotalStops", func(t *testing.T) {
		entity := &shipment.Shipment{
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{{}, {}, {}}, // 3 stops
				},
				{
					Stops: []*shipment.Stop{{}, {}}, // 2 stops
				},
			},
		}
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeTotalStops",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 5, result)
	})

	t.Run("computeRequiresTemperatureControl", func(t *testing.T) {
		// With temperature requirements
		tempMin := int16(32)
		entity := &shipment.Shipment{
			TemperatureMin: &tempMin,
		}
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeRequiresTemperatureControl",
		}

		result, err := resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, true, result)

		// Without temperature requirements
		entity.TemperatureMin = nil
		entity.TemperatureMax = nil
		result, err = resolver.ResolveField(entity, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

func TestDefaultDataResolver_ComplexNestedFields(t *testing.T) {
	resolver := schema.NewDefaultDataResolver()

	// * Create a complex shipment with nested relationships
	entity := &shipment.Shipment{
		ProNumber: "PRO123",
		Customer: &customer.Customer{
			Name: "ABC Logistics",
			Code: "ABC001",
		},
		TractorType: &equipmenttype.EquipmentType{
			Code:        "FLC",
			Description: "Freightliner Cascadia - Heavy Duty Tractor",
			Class:       equipmenttype.ClassTractor,
		},
		TrailerType: &equipmenttype.EquipmentType{
			Code:        "DV53",
			Description: "53ft Dry Van Trailer",
			Class:       equipmenttype.ClassTrailer,
		},
		Commodities: []*shipment.ShipmentCommodity{
			{
				Weight: 5000,
				Pieces: 100,
				Commodity: &commodity.Commodity{
					Name: "Hazardous Chemical",
					HazardousMaterialID: func() *pulid.ID {
						id := pulid.MustNew("hm_")
						return &id
					}(),
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						Name:     "Flammable Liquid",
						Class:    hazardousmaterial.HazardousClass3,
						UNNumber: "1203",
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		fieldSource *schema.FieldSource
		want        any
	}{
		{
			name: "equipment type code",
			fieldSource: &schema.FieldSource{
				Path: "TractorType.Code",
			},
			want: "FLC",
		},
		{
			name: "equipment type description",
			fieldSource: &schema.FieldSource{
				Path: "TractorType.Description",
			},
			want: "Freightliner Cascadia - Heavy Duty Tractor",
		},
		{
			name: "customer code",
			fieldSource: &schema.FieldSource{
				Path: "Customer.Code",
			},
			want: "ABC001",
		},
		{
			name: "trailer type code",
			fieldSource: &schema.FieldSource{
				Path: "TrailerType.Code",
			},
			want: "DV53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveField(entity, tt.fieldSource)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
