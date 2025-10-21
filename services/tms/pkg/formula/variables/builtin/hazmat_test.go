package builtin_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formula/variables/builtin"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHazmatVariables(t *testing.T) {
	// Create resolver and register computers
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	t.Run("HasHazmatVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected any
			wantErr  bool
		}{
			{
				name:     "shipment with hazmat",
				shipment: createShipmentWithHazmat(),
				expected: true,
				wantErr:  false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: false,
				wantErr:  false,
			},
			{
				name:     "shipment with nil commodities",
				shipment: &shipment.Shipment{},
				expected: false,
				wantErr:  false,
			},
			{
				name:     "shipment with commodity but no hazmat",
				shipment: createShipmentWithNonHazmatCommodity(),
				expected: false,
				wantErr:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.HasHazmatVar.Resolve(ctx)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result)
				}
			})
		}
	})

	t.Run("HazmatClassVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected string
			wantErr  bool
		}{
			{
				name:     "shipment with Class 3 hazmat",
				shipment: createShipmentWithHazmat(),
				expected: "HazardClass3",
				wantErr:  false,
			},
			{
				name:     "shipment with Class 8 hazmat",
				shipment: createShipmentWithClass8Hazmat(),
				expected: "HazardClass8",
				wantErr:  false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: "",
				wantErr:  false,
			},
			{
				name:     "non-shipment entity",
				shipment: nil,
				expected: "",
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var ctx variables.VariableContext
				if tt.shipment != nil {
					ctx = variables.NewDefaultContext(tt.shipment, resolver)
				} else {
					ctx = variables.NewDefaultContext(struct{}{}, resolver)
				}

				result, err := builtin.HazmatClassVar.Resolve(ctx)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result)
				}
			})
		}
	})

	t.Run("HazmatClassesVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected []string
			wantErr  bool
		}{
			{
				name:     "shipment with single hazmat class",
				shipment: createShipmentWithHazmat(),
				expected: []string{"HazardClass3"},
				wantErr:  false,
			},
			{
				name:     "shipment with multiple hazmat classes",
				shipment: createShipmentWithMultipleHazmatClasses(),
				expected: []string{"HazardClass3", "HazardClass8"}, // Order may vary
				wantErr:  false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: []string{},
				wantErr:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.HazmatClassesVar.Resolve(ctx)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					classes, ok := result.([]string)
					require.True(t, ok)
					assert.Len(t, classes, len(tt.expected))
					for _, exp := range tt.expected {
						assert.Contains(t, classes, exp)
					}
				}
			})
		}
	})

	t.Run("HazmatUNNumberVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected string
		}{
			{
				name:     "shipment with UN number",
				shipment: createShipmentWithHazmat(),
				expected: "UN1203",
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.HazmatUNNumberVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("HazmatPackingGroupVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected string
		}{
			{
				name:     "shipment with packing group II",
				shipment: createShipmentWithHazmat(),
				expected: "II",
			},
			{
				name:     "shipment with packing group I",
				shipment: createShipmentWithPackingGroupI(),
				expected: "I",
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.HazmatPackingGroupVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("IsExplosiveVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected bool
		}{
			{
				name:     "shipment with Class 1.1 explosive",
				shipment: createShipmentWithExplosive(hazardousmaterial.HazardousClass1And1),
				expected: true,
			},
			{
				name:     "shipment with Class 1.4 explosive",
				shipment: createShipmentWithExplosive(hazardousmaterial.HazardousClass1And4),
				expected: true,
			},
			{
				name:     "shipment with non-explosive hazmat",
				shipment: createShipmentWithHazmat(),
				expected: false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.IsExplosiveVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("IsFlammableVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected bool
		}{
			{
				name:     "shipment with Class 3 flammable",
				shipment: createShipmentWithHazmat(),
				expected: true,
			},
			{
				name:     "shipment with non-flammable hazmat",
				shipment: createShipmentWithClass8Hazmat(),
				expected: false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.IsFlammableVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("IsCorrosiveVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected bool
		}{
			{
				name:     "shipment with Class 8 corrosive",
				shipment: createShipmentWithClass8Hazmat(),
				expected: true,
			},
			{
				name:     "shipment with non-corrosive hazmat",
				shipment: createShipmentWithHazmat(),
				expected: false,
			},
			{
				name:     "shipment without hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.IsCorrosiveVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("HazmatClassNameVar", func(t *testing.T) {
		tests := []struct {
			name     string
			shipment *shipment.Shipment
			expected string
		}{
			{
				name:     "Class 3 - Flammable Liquids",
				shipment: createShipmentWithHazmat(),
				expected: "Flammable Liquids",
			},
			{
				name:     "Class 8 - Corrosive Materials",
				shipment: createShipmentWithClass8Hazmat(),
				expected: "Corrosive Materials",
			},
			{
				name:     "Class 1.1 - Explosives",
				shipment: createShipmentWithExplosive(hazardousmaterial.HazardousClass1And1),
				expected: "Explosives (Division 1.1)",
			},
			{
				name:     "No hazmat",
				shipment: createShipmentWithoutHazmat(),
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := variables.NewDefaultContext(tt.shipment, resolver)
				result, err := builtin.HazmatClassNameVar.Resolve(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestRegisterHazmatVariables(t *testing.T) {
	registry := variables.NewRegistry()

	require.NotPanics(t, func() {
		builtin.RegisterHazmatVariables(registry)
	})

	expectedVars := []string{
		"has_hazmat",
		"hazmat_class",
		"hazmat_classes",
		"hazmat_class_name",
		"hazmat_un_number",
		"hazmat_packing_group",
		"is_explosive",
		"is_flammable",
		"is_corrosive",
	}

	for _, varName := range expectedVars {
		v, err := registry.Get(varName)
		assert.NoError(t, err, "Variable %s should be registered", varName)
		assert.NotNil(t, v, "Variable %s should not be nil", varName)
	}
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

func createShipmentWithClass8Hazmat() *shipment.Shipment {
	hazmatID := pulid.MustNew("hm_")
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Sulfuric Acid",
					HazardousMaterialID: &hazmatID,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID,
						Name:         "Sulfuric Acid",
						Class:        hazardousmaterial.HazardousClass8,
						PackingGroup: hazardousmaterial.PackingGroupI,
						UNNumber:     "UN1830",
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

func createShipmentWithNonHazmatCommodity() *shipment.Shipment {
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Electronics",
					HazardousMaterialID: nil,
					HazardousMaterial:   nil,
				},
			},
		},
	}
}

func createShipmentWithMultipleHazmatClasses() *shipment.Shipment {
	hazmatID1 := pulid.MustNew("hm_")
	hazmatID2 := pulid.MustNew("hm_")
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Gasoline",
					HazardousMaterialID: &hazmatID1,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID1,
						Name:         "Gasoline",
						Class:        hazardousmaterial.HazardousClass3,
						PackingGroup: hazardousmaterial.PackingGroupII,
						UNNumber:     "UN1203",
					},
				},
			},
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Battery Acid",
					HazardousMaterialID: &hazmatID2,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID2,
						Name:         "Battery Acid",
						Class:        hazardousmaterial.HazardousClass8,
						PackingGroup: hazardousmaterial.PackingGroupII,
						UNNumber:     "UN2796",
					},
				},
			},
		},
	}
}

func createShipmentWithPackingGroupI() *shipment.Shipment {
	hazmatID := pulid.MustNew("hm_")
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Dangerous Chemical",
					HazardousMaterialID: &hazmatID,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID,
						Name:         "Dangerous Chemical",
						Class:        hazardousmaterial.HazardousClass6And1,
						PackingGroup: hazardousmaterial.PackingGroupI,
						UNNumber:     "UN2810",
					},
				},
			},
		},
	}
}

func createShipmentWithExplosive(class hazardousmaterial.HazardousClass) *shipment.Shipment {
	hazmatID := pulid.MustNew("hm_")
	return &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID: pulid.MustNew("sc_"),
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Explosive Material",
					HazardousMaterialID: &hazmatID,
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           hazmatID,
						Name:         "Explosive Material",
						Class:        class,
						PackingGroup: hazardousmaterial.PackingGroupI,
						UNNumber:     "UN0004",
					},
				},
			},
		},
	}
}
