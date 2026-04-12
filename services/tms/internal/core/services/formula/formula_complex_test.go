//go:build integration

package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestShipmentDomain struct {
	ID                  string
	ProNumber           string
	BOL                 string
	Status              string
	RatingMethod        string
	Weight              *int64
	Pieces              *int64
	TemperatureMin      *int16
	TemperatureMax      *int16
	RatingUnit          int64
	BaseRate            *decimal.Decimal
	FreightChargeAmount *decimal.Decimal
	OtherChargeAmount   *decimal.Decimal
	TotalChargeAmount   *decimal.Decimal
	Customer            *TestCustomer
	ServiceType         *TestServiceType
	ShipmentType        *TestShipmentType
	TractorType         *TestEquipmentType
	TrailerType         *TestEquipmentType
	FormulaTemplate     *TestFormulaTemplate
	Moves               []*TestShipmentMove
	Commodities         []*TestShipmentCommodity
	AdditionalCharges   []*TestAdditionalCharge
}

type TestCustomer struct {
	ID           string
	Name         string
	Code         string
	CreditLimit  *decimal.Decimal
	DiscountRate *float64
}

type TestServiceType struct {
	ID          string
	Code        string
	Description string
}

type TestShipmentType struct {
	ID          string
	Code        string
	Description string
}

type TestEquipmentType struct {
	ID          string
	Code        string
	Description string
	CostPerMile *float64
}

type TestFormulaTemplate struct {
	ID         string
	Name       string
	Expression string
}

type TestShipmentMove struct {
	ID       string
	Sequence int
	Status   string
	Loaded   bool
	Distance *float64
	Stops    []*TestStop
}

type TestStop struct {
	ID                   string
	Sequence             int
	Type                 string
	Status               string
	AddressLine          string
	Pieces               *int
	Weight               *int
	ScheduledWindowStart int64
	ScheduledWindowEnd   int64
	Location             *TestLocation
}

type TestLocation struct {
	ID      string
	Name    string
	City    string
	State   string
	ZipCode string
}

type TestShipmentCommodity struct {
	ID        string
	Weight    int64
	Pieces    int64
	Commodity *TestCommodity
}

type TestCommodity struct {
	ID                string
	Name              string
	Description       string
	HazardousMaterial *TestHazardousMaterial
}

type TestHazardousMaterial struct {
	ID    string
	Class string
	Name  string
}

type TestAdditionalCharge struct {
	ID                string
	Method            string
	Amount            decimal.Decimal
	Unit              int16
	AccessorialCharge *TestAccessorialCharge
}

type TestAccessorialCharge struct {
	ID          string
	Code        string
	Description string
	DefaultRate *decimal.Decimal
}

func setupComplexEngine(t *testing.T) *engine.Engine {
	t.Helper()

	reg := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: reg,
		Resolver: res,
	})

	return engine.NewEngine(engine.Params{
		Registry:   reg,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
}

func createRealisticShipment() *TestShipmentDomain {
	weight := int64(45000)
	pieces := int64(24)
	tempMin := int16(34)
	tempMax := int16(38)
	distance1 := 450.5
	distance2 := 125.3
	costPerMile := 1.85
	discountRate := 0.05

	stopWeight1 := 15000
	stopPieces1 := 8
	stopWeight2 := 18000
	stopPieces2 := 10
	stopWeight3 := 12000
	stopPieces3 := 6

	return &TestShipmentDomain{
		ID:             "SP_123456",
		ProNumber:      "PRO-2024-00001",
		BOL:            "BOL-98765",
		Status:         "InProgress",
		RatingMethod:   "PerMile",
		Weight:         &weight,
		Pieces:         &pieces,
		TemperatureMin: &tempMin,
		TemperatureMax: &tempMax,
		RatingUnit:     1,
		Customer: &TestCustomer{
			ID:           "CU_001",
			Name:         "Acme Distribution Inc",
			Code:         "ACME",
			DiscountRate: &discountRate,
		},
		ServiceType: &TestServiceType{
			ID:          "ST_001",
			Code:        "LTL",
			Description: "Less Than Truckload",
		},
		ShipmentType: &TestShipmentType{
			ID:          "SHT_001",
			Code:        "STANDARD",
			Description: "Standard Freight",
		},
		TractorType: &TestEquipmentType{
			ID:          "ET_001",
			Code:        "REEFER",
			Description: "Refrigerated Tractor",
			CostPerMile: &costPerMile,
		},
		Moves: []*TestShipmentMove{
			{
				ID:       "SM_001",
				Sequence: 1,
				Status:   "InProgress",
				Loaded:   true,
				Distance: &distance1,
				Stops: []*TestStop{
					{
						ID:       "STP_001",
						Sequence: 1,
						Type:     "Pickup",
						Status:   "Completed",
						Weight:   &stopWeight1,
						Pieces:   &stopPieces1,
						Location: &TestLocation{
							ID:      "LOC_001",
							Name:    "Chicago Distribution Center",
							City:    "Chicago",
							State:   "IL",
							ZipCode: "60601",
						},
					},
					{
						ID:       "STP_002",
						Sequence: 2,
						Type:     "Delivery",
						Status:   "Pending",
						Weight:   &stopWeight2,
						Pieces:   &stopPieces2,
						Location: &TestLocation{
							ID:      "LOC_002",
							Name:    "Detroit Warehouse",
							City:    "Detroit",
							State:   "MI",
							ZipCode: "48201",
						},
					},
				},
			},
			{
				ID:       "SM_002",
				Sequence: 2,
				Status:   "New",
				Loaded:   true,
				Distance: &distance2,
				Stops: []*TestStop{
					{
						ID:       "STP_003",
						Sequence: 1,
						Type:     "Delivery",
						Status:   "Pending",
						Weight:   &stopWeight3,
						Pieces:   &stopPieces3,
						Location: &TestLocation{
							ID:      "LOC_003",
							Name:    "Cleveland Fulfillment",
							City:    "Cleveland",
							State:   "OH",
							ZipCode: "44101",
						},
					},
				},
			},
		},
		Commodities: []*TestShipmentCommodity{
			{
				ID:     "SC_001",
				Weight: 25000,
				Pieces: 14,
				Commodity: &TestCommodity{
					ID:          "COM_001",
					Name:        "Frozen Produce",
					Description: "Temperature controlled produce",
				},
			},
			{
				ID:     "SC_002",
				Weight: 20000,
				Pieces: 10,
				Commodity: &TestCommodity{
					ID:          "COM_002",
					Name:        "Hazardous Chemicals",
					Description: "Class 3 Flammable Liquids",
					HazardousMaterial: &TestHazardousMaterial{
						ID:    "HM_001",
						Class: "3",
						Name:  "Flammable Liquids",
					},
				},
			},
		},
		AdditionalCharges: []*TestAdditionalCharge{
			{
				ID:     "AC_001",
				Method: "Flat",
				Amount: decimal.NewFromFloat(75.00),
				Unit:   1,
				AccessorialCharge: &TestAccessorialCharge{
					ID:          "ACC_001",
					Code:        "LIFTGATE",
					Description: "Liftgate Service",
				},
			},
			{
				ID:     "AC_002",
				Method: "PerStop",
				Amount: decimal.NewFromFloat(25.00),
				Unit:   3,
				AccessorialCharge: &TestAccessorialCharge{
					ID:          "ACC_002",
					Code:        "STOPOFF",
					Description: "Stop-off Charge",
				},
			},
		},
	}
}

func createMinimalShipment() *TestShipmentDomain {
	distance := 50.0
	return &TestShipmentDomain{
		ID:        "SP_MINIMAL",
		ProNumber: "PRO-MIN-001",
		Status:    "New",
		Moves: []*TestShipmentMove{
			{
				ID:       "SM_MIN",
				Distance: &distance,
				Stops:    []*TestStop{},
			},
		},
		Commodities: []*TestShipmentCommodity{},
	}
}

func TestComplex_PerMileWithMinimumAndFuelSurcharge(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Per Mile with Minimum and Fuel Surcharge",
		SchemaID: "shipment",
		Expression: `
			max(
				minimumCharge,
				(startingRate + (ratePerMile * totalDistance)) * (1 + fuelSurchargePercent / 100)
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 250.0,
			},
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 75.0},
			{Name: "ratePerMile", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.85},
			{
				Name:         "fuelSurchargePercent",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 18.5,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := (75.0 + (2.85 * 575.8)) * 1.185
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_TieredWeightPricing(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Tiered Weight Pricing",
		SchemaID: "shipment",
		Expression: `
			totalWeight < 10000 ? totalWeight * tierOneRate :
			totalWeight < 25000 ? 10000 * tierOneRate + (totalWeight - 10000) * tierTwoRate :
			totalWeight < 40000 ? 10000 * tierOneRate + 15000 * tierTwoRate + (totalWeight - 25000) * tierThreeRate :
			10000 * tierOneRate + 15000 * tierTwoRate + 15000 * tierThreeRate + (totalWeight - 40000) * tierFourRate
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "tierOneRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 0.15},
			{Name: "tierTwoRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 0.12},
			{Name: "tierThreeRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 0.09},
			{Name: "tierFourRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 0.06},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 10000*0.15 + 15000*0.12 + 15000*0.09 + (45000-40000)*0.06
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_CWTWithAccessorials(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "CWT with Accessorials",
		SchemaID: "shipment",
		Expression: `
			round(
				(ratePerCWT * ceil(totalWeight / 100)) +
				(hasHazmat ? hazmatFee : 0) +
				(requiresTemperatureControl ? reeferFee : 0) +
				(totalStops > 2 ? (totalStops - 2) * additionalStopFee : 0),
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "ratePerCWT", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 18.50},
			{Name: "hazmatFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 175.00},
			{Name: "reeferFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 225.00},
			{
				Name:         "additionalStopFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 50.00,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	cwtCharge := 18.50 * 450.0
	hazmatFee := 175.0
	reeferFee := 225.0
	stopFee := (3 - 2) * 50.0
	expected := cwtCharge + hazmatFee + reeferFee + stopFee
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_FullFreightBillCalculation(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Full Freight Bill",
		SchemaID: "shipment",
		Expression: `
			round(
				max(
					minimumCharge,
					clamp(
						(startingRate + (ratePerMile * totalDistance) + (ratePerStop * totalStops)) *
						(1 + fuelSurcharge / 100) +
						(hasHazmat ? hazmatFee : 0) +
						(requiresTemperatureControl ? reeferSurcharge + (temperatureDifferential < 10 ? tightTempFee : 0) : 0),
						minimumCharge,
						maximumCharge
					)
				),
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 350.00,
			},
			{
				Name:         "maximumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 10000.00,
			},
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 125.00},
			{Name: "ratePerMile", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 3.25},
			{Name: "ratePerStop", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 45.00},
			{Name: "fuelSurcharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 22.0},
			{Name: "hazmatFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 200.00},
			{
				Name:         "reeferSurcharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 275.00,
			},
			{
				Name:         "tightTempFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 100.00,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	linehaul := (125.0 + (3.25 * 575.8) + (45.0 * 3)) * 1.22
	hazmat := 200.0
	reefer := 275.0 + 100.0
	total := linehaul + hazmat + reefer
	clamped := min(max(total, 350.0), 10000.0)
	assert.InDelta(t, clamped, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_DistanceBasedTiers(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Distance Based Tiers",
		SchemaID: "shipment",
		Expression: `
			totalDistance <= 100 ? startingRate + (totalDistance * shortHaulRate) :
			totalDistance <= 500 ? startingRate + 100 * shortHaulRate + (totalDistance - 100) * mediumHaulRate :
			startingRate + 100 * shortHaulRate + 400 * mediumHaulRate + (totalDistance - 500) * longHaulRate
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 100.00},
			{Name: "shortHaulRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 4.50},
			{
				Name:         "mediumHaulRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 3.25,
			},
			{Name: "longHaulRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.75},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 100.0 + 100*4.50 + 400*3.25 + (575.8-500)*2.75
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_MinimalShipmentHandling(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Safe Minimal Calculation",
		SchemaID: "shipment",
		Expression: `
			max(
				minimumCharge,
				baseRate + (ratePerMile * totalDistance) +
				(hasHazmat ? hazmatFee : 0) +
				(requiresTemperatureControl ? tempFee : 0)
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 150.00,
			},
			{Name: "baseRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 50.00},
			{Name: "ratePerMile", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.00},
			{Name: "hazmatFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 100.00},
			{Name: "tempFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 75.00},
		},
	}

	shipment := createMinimalShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := max(150.0, 50.0+(2.0*50.0)+0+0)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_RuntimeVariableOverrides(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Negotiated Rate",
		SchemaID: "shipment",
		Expression: `
			round(
				(negotiatedRate * totalDistance) * (1 - discountPercent / 100),
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "negotiatedRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 3.00,
			},
			{
				Name:         "discountPercent",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 0.0,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   shipment,
		Variables: map[string]any{
			"negotiatedRate":  4.25,
			"discountPercent": 12.5,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := (4.25 * 575.8) * (1 - 12.5/100)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_NestedConditionals(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Complex Conditional Logic",
		SchemaID: "shipment",
		Expression: `
			hasHazmat ? (
				requiresTemperatureControl ?
					startingRate * hazmatReeferMultiplier :
					startingRate * hazmatMultiplier
			) : (
				requiresTemperatureControl ?
					startingRate * reeferMultiplier :
					startingRate * standardMultiplier
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1000.00},
			{
				Name:         "standardMultiplier",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.0,
			},
			{
				Name:         "reeferMultiplier",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.35,
			},
			{
				Name:         "hazmatMultiplier",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.5,
			},
			{
				Name:         "hazmatReeferMultiplier",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.85,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 1000.0 * 1.85
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_MathFunctionChaining(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Math Function Chain",
		SchemaID: "shipment",
		Expression: `
			round(
				sqrt(
					pow(totalDistance, 2) + pow(totalWeight / 1000, 2)
				) * rateMultiplier,
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "rateMultiplier", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.5},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 866.33, result.Value.InexactFloat64(), 1.0)
}

func TestComplex_AverageAndSum(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Average and Sum Calculations",
		SchemaID: "shipment",
		Expression: `
			round(
				sum(baseCharge, distanceCharge, weightCharge, stopCharge) *
				(1 + avg(fuelSurcharge, adminFee, insuranceFee) / 100),
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "baseCharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 100.00},
			{
				Name:         "distanceCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 500.00,
			},
			{
				Name:         "weightCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 250.00,
			},
			{Name: "stopCharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 75.00},
			{Name: "fuelSurcharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 18.0},
			{Name: "adminFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 3.0},
			{Name: "insuranceFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 6.0},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	base := 100.0 + 500.0 + 250.0 + 75.0
	avgFee := (18.0 + 3.0 + 6.0) / 3.0
	expected := base * (1 + avgFee/100)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_CoalesceForDefaults(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Coalesce Default Values",
		SchemaID: "shipment",
		Expression: `
			coalesce(customRate, defaultRate) * totalDistance
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "customRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: nil},
			{Name: "defaultRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.50},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   shipment,
		Variables: map[string]any{
			"customRate": nil,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 2.50 * 575.8
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_AbsoluteValueCalculation(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Temperature Variance Charge",
		SchemaID: "shipment",
		Expression: `
			requiresTemperatureControl ?
				baseCharge + (abs(temperatureDifferential - targetDifferential) * variancePenalty) :
				baseCharge
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "baseCharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 500.00},
			{
				Name:         "targetDifferential",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 10.0,
			},
			{
				Name:         "variancePenalty",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 25.00,
			},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	tempDiff := 38.0 - 34.0
	variance := 10.0 - tempDiff
	expected := 500.0 + (variance * 25.0)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_FloorCeilRounding(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Rounded Unit Pricing",
		SchemaID: "shipment",
		Expression: `
			ceil(totalWeight / 100) * ratePerCWT +
			floor(totalDistance / 100) * ratePer100Miles +
			round(totalStops * stopRate, 2)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "ratePerCWT", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 12.50},
			{
				Name:         "ratePer100Miles",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 85.00,
			},
			{Name: "stopRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 33.33},
		},
	}

	shipment := createRealisticShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	cwtUnits := 450.0
	hundredMileUnits := 5.0
	expected := cwtUnits*12.50 + hundredMileUnits*85.0 + 3*33.33
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestComplex_NilShipmentFieldsGracefulHandling(t *testing.T) {
	eng := setupComplexEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Graceful Nil Handling",
		SchemaID: "shipment",
		Expression: `
			startingRate + (ratePerMile * totalDistance) +
			(hasHazmat ? hazmatFee : 0) +
			(requiresTemperatureControl ? tempFee : 0)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 100.00},
			{Name: "ratePerMile", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.00},
			{Name: "hazmatFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 150.00},
			{Name: "tempFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 200.00},
		},
	}

	shipment := &TestShipmentDomain{
		ID:             "SP_NIL",
		ProNumber:      "PRO-NIL",
		Weight:         nil,
		Pieces:         nil,
		TemperatureMin: nil,
		TemperatureMax: nil,
		Moves:          nil,
		Commodities:    nil,
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 100.0 + (2.0 * 0) + 0 + 0
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}
