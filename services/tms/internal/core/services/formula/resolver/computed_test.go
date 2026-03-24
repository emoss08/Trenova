package resolver_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Move struct {
	Distance float64
	Stops    []Stop
}

type Stop struct {
	Name string
}

type Commodity struct {
	Weight    int64
	Pieces    int64
	Commodity *CommodityDetail
}

type CommodityDetail struct {
	HazardousMaterial *HazardousMaterial
	LinearFeetPerUnit float64
}

type HazardousMaterial struct {
	Class string
}

type ShipmentEntity struct {
	Weight              int64
	Pieces              int64
	TemperatureMin      *int16
	TemperatureMax      *int16
	Moves               []Move
	Commodities         []Commodity
	FreightChargeAmount *float64
	OtherChargeAmount   *float64
	TotalChargeAmount   *float64
}

func TestRegisterDefaultComputed(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	expectedFunctions := []string{
		"computeTotalDistance",
		"computeTotalStops",
		"computeHasHazmat",
		"computeRequiresTemperatureControl",
		"computeTemperatureDifferential",
		"computeTotalWeight",
		"computeTotalPieces",
		"computeTotalLinearFeet",
		"computeFreightChargeAmount",
		"computeOtherChargeAmount",
		"computeCurrentTotalCharge",
	}

	for _, fn := range expectedFunctions {
		t.Run(fn, func(t *testing.T) {
			t.Parallel()
			got, ok := r.GetComputed(fn)
			assert.True(t, ok)
			assert.NotNil(t, got)
		})
	}
}

func TestComputeTotalDistance(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "sum of move distances",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Distance: 100.5},
					{Distance: 200.3},
					{Distance: 50.2},
				},
			},
			want: 351.0,
		},
		{
			name: "empty moves",
			entity: &ShipmentEntity{
				Moves: []Move{},
			},
			want: 0.0,
		},
		{
			name:   "nil moves slice",
			entity: &ShipmentEntity{},
			want:   0.0,
		},
		{
			name: "single move",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Distance: 250.0},
				},
			},
			want: 250.0,
		},
		{
			name: "zero distances",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Distance: 0.0},
					{Distance: 0.0},
				},
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTotalDistance")
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.1)
		})
	}
}

func TestComputeTotalDistanceNonStruct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	_, err := r.ResolveComputed("not a struct", "computeTotalDistance")
	require.Error(t, err)
}

func TestComputeTotalStops(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   int
	}{
		{
			name: "sum of stops across moves",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Stops: []Stop{{Name: "A"}, {Name: "B"}}},
					{Stops: []Stop{{Name: "C"}}},
				},
			},
			want: 3,
		},
		{
			name: "no stops",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Stops: []Stop{}},
				},
			},
			want: 0,
		},
		{
			name:   "nil moves",
			entity: &ShipmentEntity{},
			want:   0,
		},
		{
			name: "multiple moves no stops",
			entity: &ShipmentEntity{
				Moves: []Move{
					{Stops: nil},
					{Stops: nil},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTotalStops")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeHasHazmat(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   bool
	}{
		{
			name: "has hazmat",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{
						Commodity: &CommodityDetail{
							HazardousMaterial: &HazardousMaterial{Class: "3"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "no hazmat nil material",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{
						Commodity: &CommodityDetail{
							HazardousMaterial: nil,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "nil commodity detail",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{Commodity: nil},
				},
			},
			want: false,
		},
		{
			name: "empty commodities",
			entity: &ShipmentEntity{
				Commodities: []Commodity{},
			},
			want: false,
		},
		{
			name:   "nil commodities",
			entity: &ShipmentEntity{},
			want:   false,
		},
		{
			name: "mixed hazmat and non-hazmat",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{Commodity: &CommodityDetail{HazardousMaterial: nil}},
					{
						Commodity: &CommodityDetail{
							HazardousMaterial: &HazardousMaterial{Class: "2"},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeHasHazmat")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeRequiresTemperatureControl(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	minTemp := int16(32)
	maxTemp := int16(40)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   bool
	}{
		{
			name: "has min and max",
			entity: &ShipmentEntity{
				TemperatureMin: &minTemp,
				TemperatureMax: &maxTemp,
			},
			want: true,
		},
		{
			name: "has only min",
			entity: &ShipmentEntity{
				TemperatureMin: &minTemp,
				TemperatureMax: nil,
			},
			want: true,
		},
		{
			name: "has only max",
			entity: &ShipmentEntity{
				TemperatureMin: nil,
				TemperatureMax: &maxTemp,
			},
			want: true,
		},
		{
			name: "no temperature control",
			entity: &ShipmentEntity{
				TemperatureMin: nil,
				TemperatureMax: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeRequiresTemperatureControl")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeTemperatureDifferential(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	minTemp := int16(32)
	maxTemp := int16(40)

	tests := []struct {
		name    string
		entity  *ShipmentEntity
		want    float64
		wantErr bool
	}{
		{
			name: "calculate differential",
			entity: &ShipmentEntity{
				TemperatureMin: &minTemp,
				TemperatureMax: &maxTemp,
			},
			want:    8.0,
			wantErr: false,
		},
		{
			name: "missing min temperature returns error",
			entity: &ShipmentEntity{
				TemperatureMin: nil,
				TemperatureMax: &maxTemp,
			},
			want:    0.0,
			wantErr: true,
		},
		{
			name: "missing max temperature returns zero no error",
			entity: &ShipmentEntity{
				TemperatureMin: &minTemp,
				TemperatureMax: nil,
			},
			want:    0.0,
			wantErr: false,
		},
		{
			name: "both nil returns error",
			entity: &ShipmentEntity{
				TemperatureMin: nil,
				TemperatureMax: nil,
			},
			want:    0.0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTemperatureDifferential")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeTotalWeight(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "use shipment weight if set",
			entity: &ShipmentEntity{
				Weight: 5000,
				Commodities: []Commodity{
					{Weight: 1000},
					{Weight: 2000},
				},
			},
			want: 5000.0,
		},
		{
			name: "sum commodity weights if shipment weight zero",
			entity: &ShipmentEntity{
				Weight: 0,
				Commodities: []Commodity{
					{Weight: 1000},
					{Weight: 2000},
				},
			},
			want: 3000.0,
		},
		{
			name: "empty commodities with zero weight",
			entity: &ShipmentEntity{
				Weight:      0,
				Commodities: []Commodity{},
			},
			want: 0.0,
		},
		{
			name: "single commodity",
			entity: &ShipmentEntity{
				Weight: 0,
				Commodities: []Commodity{
					{Weight: 4500},
				},
			},
			want: 4500.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTotalWeight")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeTotalPieces(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   int64
	}{
		{
			name: "use shipment pieces if set",
			entity: &ShipmentEntity{
				Pieces: 100,
				Commodities: []Commodity{
					{Pieces: 10},
					{Pieces: 20},
				},
			},
			want: 100,
		},
		{
			name: "sum commodity pieces if shipment pieces zero",
			entity: &ShipmentEntity{
				Pieces: 0,
				Commodities: []Commodity{
					{Pieces: 10},
					{Pieces: 20},
				},
			},
			want: 30,
		},
		{
			name: "empty commodities",
			entity: &ShipmentEntity{
				Pieces:      0,
				Commodities: []Commodity{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTotalPieces")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeTotalLinearFeet(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "sum linear feet",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{
						Pieces:    10,
						Commodity: &CommodityDetail{LinearFeetPerUnit: 2.5},
					},
					{
						Pieces:    5,
						Commodity: &CommodityDetail{LinearFeetPerUnit: 3.0},
					},
				},
			},
			want: 40.0,
		},
		{
			name: "zero pieces skipped",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{
						Pieces:    0,
						Commodity: &CommodityDetail{LinearFeetPerUnit: 2.5},
					},
					{
						Pieces:    5,
						Commodity: &CommodityDetail{LinearFeetPerUnit: 3.0},
					},
				},
			},
			want: 15.0,
		},
		{
			name: "nil commodity detail skipped",
			entity: &ShipmentEntity{
				Commodities: []Commodity{
					{
						Pieces:    10,
						Commodity: nil,
					},
				},
			},
			want: 0.0,
		},
		{
			name: "empty commodities",
			entity: &ShipmentEntity{
				Commodities: []Commodity{},
			},
			want: 0.0,
		},
		{
			name:   "nil commodities",
			entity: &ShipmentEntity{},
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeTotalLinearFeet")
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestComputeFreightChargeAmount(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "with freight charge",
			entity: &ShipmentEntity{
				FreightChargeAmount: new(1500.50),
			},
			want: 1500.50,
		},
		{
			name: "nil freight charge",
			entity: &ShipmentEntity{
				FreightChargeAmount: nil,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeFreightChargeAmount")
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestComputeOtherChargeAmount(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "with other charge",
			entity: &ShipmentEntity{
				OtherChargeAmount: new(250.00),
			},
			want: 250.00,
		},
		{
			name: "nil other charge",
			entity: &ShipmentEntity{
				OtherChargeAmount: nil,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeOtherChargeAmount")
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestComputeOtherChargeAmount_UsesShipmentAdditionalChargeOverrides(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	entity := &shipment.Shipment{
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(50)),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Method: accessorialcharge.MethodFlat,
				Amount: decimal.NewFromInt(200),
				Unit:   1,
			},
			{
				Method: accessorialcharge.MethodPercentage,
				Amount: decimal.NewFromInt(5),
				Unit:   1,
			},
		},
	}

	got, err := r.ResolveComputed(entity, "computeOtherChargeAmount")

	require.NoError(t, err)
	assert.InDelta(t, 250.0, got, 0.001)
}

func TestComputeCurrentTotalCharge(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		want   float64
	}{
		{
			name: "with total charge",
			entity: &ShipmentEntity{
				TotalChargeAmount: new(3500.75),
			},
			want: 3500.75,
		},
		{
			name: "nil total charge",
			entity: &ShipmentEntity{
				TotalChargeAmount: nil,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, "computeCurrentTotalCharge")
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestComputedFunctionNotFound(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	_, err := r.ResolveComputed(&ShipmentEntity{}, "nonExistentFunction")
	require.Error(t, err)
}

func TestComputedWithNilEntity(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	_, err := r.ResolveComputed((*ShipmentEntity)(nil), "computeTotalDistance")
	require.Error(t, err)
}

//go:fix inline
func ptrFloat64(v float64) *float64 {
	return new(v)
}
