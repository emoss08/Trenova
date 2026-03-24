package resolver_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FieldTestEntity struct {
	FloatVal       float64
	Float32Val     float32
	IntVal         int
	Int64Val       int64
	Int16Val       int16
	StringVal      string
	BoolVal        bool
	PtrFloat64Val  *float64
	PtrInt64Val    *int64
	PtrIntVal      *int
	PtrInt16Val    *int16
	SliceVal       []int
	NonSliceVal    int
	StructVal      InnerStruct
	PtrStructVal   *InnerStruct
	NilPtrStruct   *InnerStruct
	DecimalLike    FakeDecimal
	NullDecimal    FakeNullDecimal
	PtrDecimalLike *FakeDecimal
}

type InnerStruct struct {
	Value float64
}

type FakeDecimal struct {
	InnerDecimal FakeInnerDecimal
}

type FakeInnerDecimal struct{}

func (FakeInnerDecimal) InexactFloat64() float64 {
	return 99.99
}

type FakeNullDecimal struct {
	Valid   bool
	Decimal FakeInnerDecimal
}

type FakeValidDecimal struct {
	Valid   bool
	Decimal FakeInnerDecimal
}

func TestGetFieldFloat64_ViaResolveField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	ptrVal := 42.5
	tests := []struct {
		name    string
		entity  any
		field   string
		wantErr bool
	}{
		{
			name:   "float64 field",
			entity: &FieldTestEntity{FloatVal: 3.14},
			field:  "FloatVal",
		},
		{
			name:   "ptr float64 non-nil",
			entity: &FieldTestEntity{PtrFloat64Val: &ptrVal},
			field:  "PtrFloat64Val",
		},
		{
			name:    "ptr float64 nil returns error for non-nullable",
			entity:  &FieldTestEntity{PtrFloat64Val: nil},
			field:   "PtrFloat64Val",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := &formulatypes.FieldSource{Field: tt.field}
			_, err := r.ResolveField(tt.entity, source)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetFieldFloat64_NullableNilPtr(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{
		Field:    "PtrFloat64Val",
		Nullable: true,
	}

	entity := &FieldTestEntity{PtrFloat64Val: nil}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetFieldFloat64_NonNumericType(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithStringDistance struct {
		Distance string
		Moves    []EntityWithStringDistance
	}

	entity := &struct {
		Moves []EntityWithStringDistance
	}{
		Moves: []EntityWithStringDistance{
			{Distance: "not_a_number"},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldInt64_ViaComputed(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	tests := []struct {
		name   string
		entity *ShipmentEntity
		fn     string
		want   any
	}{
		{
			name:   "int64 weight field",
			entity: &ShipmentEntity{Weight: 5000},
			fn:     "computeTotalWeight",
			want:   5000.0,
		},
		{
			name:   "int64 pieces field",
			entity: &ShipmentEntity{Pieces: 25},
			fn:     "computeTotalPieces",
			want:   int64(25),
		},
		{
			name: "zero weight falls through to commodities",
			entity: &ShipmentEntity{
				Weight: 0,
				Commodities: []Commodity{
					{Weight: 1000},
					{Weight: 2000},
				},
			},
			fn:   "computeTotalWeight",
			want: 3000.0,
		},
		{
			name: "zero pieces falls through to commodities",
			entity: &ShipmentEntity{
				Pieces: 0,
				Commodities: []Commodity{
					{Pieces: 5},
					{Pieces: 10},
				},
			},
			fn:   "computeTotalPieces",
			want: int64(15),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.ResolveComputed(tt.entity, tt.fn)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetFieldInt64_NonNumericType(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithStringWeight struct {
		Weight string
		Pieces string
		Moves  []struct{}
	}

	entity := &EntityWithStringWeight{
		Weight: "heavy",
		Pieces: "many",
	}

	_, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.Error(t, err)
}

func TestGetFieldInt64_PtrIntNil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithPtrInt struct {
		Weight *int
		Pieces *int64
		Moves  []struct{}
	}

	entity := &EntityWithPtrInt{
		Weight: nil,
		Pieces: nil,
	}

	_, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.Error(t, err)
}

func TestGetFieldInt64_PtrIntNonNil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithPtrInt struct {
		Weight      *int
		Pieces      *int64
		Moves       []struct{}
		Commodities []struct{}
	}

	w := 5000
	entity := &EntityWithPtrInt{
		Weight: &w,
	}

	got, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.NoError(t, err)
	assert.InDelta(t, 5000.0, got, 0.001)
}

func TestGetFieldInt64_IntType(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithInt struct {
		Weight      int
		Pieces      int
		Moves       []struct{}
		Commodities []struct{}
	}

	entity := &EntityWithInt{
		Weight: 3000,
		Pieces: 50,
	}

	got, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.NoError(t, err)
	assert.InDelta(t, 3000.0, got, 0.001)

	gotPieces, err := r.ResolveComputed(entity, "computeTotalPieces")
	require.NoError(t, err)
	assert.Equal(t, int64(50), gotPieces)
}

func TestGetFieldInt16_ViaComputed(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	min16 := int16(30)
	max16 := int16(50)

	tests := []struct {
		name    string
		entity  *ShipmentEntity
		want    float64
		wantErr bool
	}{
		{
			name: "valid int16 pointers",
			entity: &ShipmentEntity{
				TemperatureMin: &min16,
				TemperatureMax: &max16,
			},
			want: 20.0,
		},
		{
			name: "nil min returns error",
			entity: &ShipmentEntity{
				TemperatureMin: nil,
				TemperatureMax: &max16,
			},
			wantErr: true,
		},
		{
			name: "nil max returns zero",
			entity: &ShipmentEntity{
				TemperatureMin: &min16,
				TemperatureMax: nil,
			},
			want: 0.0,
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
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestGetFieldInt16_NonInt16Type(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithStringTemps struct {
		TemperatureMin string
		TemperatureMax string
	}

	entity := &EntityWithStringTemps{
		TemperatureMin: "cold",
		TemperatureMax: "hot",
	}

	_, err := r.ResolveComputed(entity, "computeTemperatureDifferential")
	require.Error(t, err)
}

func TestGetFieldInt16_DirectInt16Value(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithDirectInt16 struct {
		TemperatureMin int16
		TemperatureMax int16
	}

	entity := &EntityWithDirectInt16{
		TemperatureMin: 20,
		TemperatureMax: 60,
	}

	got, err := r.ResolveComputed(entity, "computeTemperatureDifferential")
	require.NoError(t, err)
	assert.InDelta(t, 40.0, got, 0.001)
}

func TestGetFieldSlice_NonSliceField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithNonSliceMoves struct {
		Moves int
	}

	entity := &EntityWithNonSliceMoves{
		Moves: 5,
	}

	_, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.Error(t, err)
}

func TestGetFieldSlice_NilSlice(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithNilSlice struct {
		Moves []struct {
			Distance float64
		}
	}

	entity := &EntityWithNilSlice{
		Moves: nil,
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldSlice_FieldNotFound(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithoutMoves struct {
		Name string
	}

	entity := &EntityWithoutMoves{Name: "test"}

	_, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.Error(t, err)
}

func TestGetFieldDecimal_NilValue(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	entity := &ShipmentEntity{
		FreightChargeAmount: nil,
		OtherChargeAmount:   nil,
		TotalChargeAmount:   nil,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)

	got, err = r.ResolveComputed(entity, "computeOtherChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)

	got, err = r.ResolveComputed(entity, "computeCurrentTotalCharge")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldDecimal_Float64Value(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	val := 1234.56
	entity := &ShipmentEntity{
		FreightChargeAmount: &val,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 1234.56, got, 0.001)
}

func TestGetFieldDecimal_StructWithValidFalse(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithNullDecimal struct {
		FreightChargeAmount FakeNullDecimal
	}

	entity := &EntityWithNullDecimal{
		FreightChargeAmount: FakeNullDecimal{
			Valid:   false,
			Decimal: FakeInnerDecimal{},
		},
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldDecimal_StructWithValidTrue(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithValidDecimal struct {
		FreightChargeAmount FakeValidDecimal
	}

	entity := &EntityWithValidDecimal{
		FreightChargeAmount: FakeValidDecimal{
			Valid:   true,
			Decimal: FakeInnerDecimal{},
		},
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 99.99, got, 0.001)
}

func TestGetFieldDecimal_StructWithInexactFloat64Method(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithDecimalMethod struct {
		FreightChargeAmount FakeInnerDecimal
	}

	entity := &EntityWithDecimalMethod{
		FreightChargeAmount: FakeInnerDecimal{},
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 99.99, got, 0.001)
}

func TestGetFieldDecimal_PtrToDecimalStruct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type PtrDecimalEntity struct {
		FreightChargeAmount *FakeNullDecimal
	}

	entity := &PtrDecimalEntity{
		FreightChargeAmount: &FakeNullDecimal{
			Valid:   true,
			Decimal: FakeInnerDecimal{},
		},
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 99.99, got, 0.001)
}

func TestGetFieldDecimal_PtrToDecimalStructNil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type PtrDecimalEntity struct {
		FreightChargeAmount *FakeNullDecimal
	}

	entity := &PtrDecimalEntity{
		FreightChargeAmount: nil,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldDecimal_NonNumericNonStruct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithStringDecimal struct {
		FreightChargeAmount string
	}

	entity := &EntityWithStringDecimal{
		FreightChargeAmount: "not_a_decimal",
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldDecimal_IntField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithIntDecimal struct {
		FreightChargeAmount int
	}

	entity := &EntityWithIntDecimal{
		FreightChargeAmount: 500,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldDecimal_Float64Direct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithFloat64Decimal struct {
		FreightChargeAmount float64
	}

	entity := &EntityWithFloat64Decimal{
		FreightChargeAmount: 250.75,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 250.75, got, 0.001)
}

func TestGetFieldDecimal_FieldNotFound(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithoutField struct {
		Name string
	}

	entity := &EntityWithoutField{Name: "test"}

	_, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.Error(t, err)
}

func TestGetFieldFloat64_IntField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithIntDistance struct {
		Moves []struct {
			Distance int
		}
	}

	entity := &EntityWithIntDistance{
		Moves: []struct {
			Distance int
		}{
			{Distance: 100},
			{Distance: 200},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 300.0, got, 0.001)
}

func TestGetFieldFloat64_Int64Field(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithInt64Distance struct {
		Moves []struct {
			Distance int64
		}
	}

	entity := &EntityWithInt64Distance{
		Moves: []struct {
			Distance int64
		}{
			{Distance: 150},
			{Distance: 250},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 400.0, got, 0.001)
}

func TestGetFieldFloat64_Float32Field(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithFloat32Distance struct {
		Moves []struct {
			Distance float32
		}
	}

	entity := &EntityWithFloat32Distance{
		Moves: []struct {
			Distance float32
		}{
			{Distance: 100.5},
			{Distance: 50.5},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 151.0, got, 0.1)
}

func TestGetFieldFloat64_PtrFloat64Field(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	d1 := 100.0
	d2 := 200.0

	type EntityWithPtrDistance struct {
		Moves []struct {
			Distance *float64
		}
	}

	entity := &EntityWithPtrDistance{
		Moves: []struct {
			Distance *float64
		}{
			{Distance: &d1},
			{Distance: &d2},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 300.0, got, 0.001)
}

func TestGetFieldFloat64_PtrFloat64Nil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithNilPtrDistance struct {
		Moves []struct {
			Distance *float64
		}
	}

	entity := &EntityWithNilPtrDistance{
		Moves: []struct {
			Distance *float64
		}{
			{Distance: nil},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestResolveField_NonStructEntity(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{Field: "SomeField"}
	_, err := r.ResolveField("not_a_struct", source)
	require.Error(t, err)
}

func TestResolveField_NilSource(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	_, err := r.ResolveField(&ShipmentEntity{}, nil)
	require.Error(t, err)
}

func TestResolveField_ComputedSource(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{Computed: true, Function: "computeSomething"}
	_, err := r.ResolveField(&ShipmentEntity{}, source)
	require.Error(t, err)
}

func TestResolveField_EmptyPathAndField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{}
	_, err := r.ResolveField(&ShipmentEntity{}, source)
	require.Error(t, err)
}

func TestResolveField_WithTransform(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		Value int64
	}

	source := &formulatypes.FieldSource{
		Field:     "Value",
		Transform: "int64ToFloat64",
	}

	entity := &Entity{Value: 42}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, float64(42), got)
}

func TestResolveField_WithUnknownTransform(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		Value int64
	}

	source := &formulatypes.FieldSource{
		Field:     "Value",
		Transform: "unknownTransform",
	}

	entity := &Entity{Value: 42}
	_, err := r.ResolveField(entity, source)
	require.Error(t, err)
}

func TestResolveAllFields(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	entity := &ShipmentEntity{
		Weight: 5000,
		Pieces: 100,
		Moves: []Move{
			{Distance: 100.0},
		},
	}

	sources := map[string]*formulatypes.FieldSource{
		"weight": {Field: "Weight"},
		"pieces": {Field: "Pieces"},
		"totalDistance": {
			Computed: true,
			Function: "computeTotalDistance",
		},
	}

	result, err := r.ResolveAllFields(entity, sources)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), result["weight"])
	assert.Equal(t, int64(100), result["pieces"])
	assert.InDelta(t, 100.0, result["totalDistance"], 0.001)
}

func TestResolveAllFields_NullableFieldError(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	entity := &ShipmentEntity{}

	sources := map[string]*formulatypes.FieldSource{
		"missing": {
			Field:    "NonExistentField",
			Nullable: true,
		},
	}

	result, err := r.ResolveAllFields(entity, sources)
	require.NoError(t, err)
	assert.Nil(t, result["missing"])
}

func TestResolveAllFields_NonNullableFieldError(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	entity := &ShipmentEntity{}

	sources := map[string]*formulatypes.FieldSource{
		"missing": {
			Field:    "NonExistentField",
			Nullable: false,
		},
	}

	_, err := r.ResolveAllFields(entity, sources)
	require.Error(t, err)
}

func TestComputeWithNonStructInput(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	fnsExpectError := []string{
		"computeTotalDistance",
		"computeTotalStops",
		"computeHasHazmat",
		"computeTemperatureDifferential",
		"computeTotalWeight",
		"computeTotalPieces",
		"computeFreightChargeAmount",
		"computeOtherChargeAmount",
		"computeCurrentTotalCharge",
	}

	for _, fn := range fnsExpectError {
		t.Run(fn, func(t *testing.T) {
			t.Parallel()
			_, err := r.ResolveComputed(42, fn)
			require.Error(t, err)
		})
	}
}

func TestComputeRequiresTemperatureControl_NonStruct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	got, err := r.ResolveComputed(42, "computeRequiresTemperatureControl")
	require.NoError(t, err)
	assert.Equal(t, false, got)
}

func TestComputeTotalLinearFeet_NonStruct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	got, err := r.ResolveComputed(42, "computeTotalLinearFeet")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestGetFieldInt64_PtrInt64Nil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithPtrInt64 struct {
		Weight      *int64
		Pieces      *int64
		Moves       []struct{}
		Commodities []struct{ Weight int64 }
	}

	entity := &EntityWithPtrInt64{
		Weight: nil,
		Pieces: nil,
		Commodities: []struct{ Weight int64 }{
			{Weight: 100},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.NoError(t, err)
	assert.InDelta(t, 100.0, got, 0.001)
}

func TestGetFieldInt64_PtrInt64NonNil(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithPtrInt64 struct {
		Weight      *int64
		Pieces      *int64
		Moves       []struct{}
		Commodities []struct{ Weight int64 }
	}

	w := int64(7000)
	entity := &EntityWithPtrInt64{
		Weight: &w,
	}

	got, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.NoError(t, err)
	assert.InDelta(t, 7000.0, got, 0.001)
}

func TestGetFieldSlice_PtrToSlice(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithPtrSlice struct {
		Moves *[]struct {
			Distance float64
		}
	}

	moves := []struct {
		Distance float64
	}{
		{Distance: 100.0},
	}

	entity := &EntityWithPtrSlice{
		Moves: &moves,
	}

	got, err := r.ResolveComputed(entity, "computeTotalDistance")
	require.NoError(t, err)
	assert.InDelta(t, 100.0, got, 0.001)
}

func TestComputeLinearFeet_WithCommodityMissingLinearFeetPerUnit(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type CommodityNoLF struct {
		Name string
	}

	type CommodityItem struct {
		Pieces    int64
		Commodity *CommodityNoLF
	}

	type Entity struct {
		Commodities []CommodityItem
	}

	entity := &Entity{
		Commodities: []CommodityItem{
			{Pieces: 10, Commodity: &CommodityNoLF{Name: "test"}},
		},
	}

	got, err := r.ResolveComputed(entity, "computeTotalLinearFeet")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestResolveField_WithPath(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Inner struct {
		Name string
	}
	type Outer struct {
		Inner Inner
	}

	source := &formulatypes.FieldSource{Path: "Inner.Name"}
	entity := &Outer{Inner: Inner{Name: "hello"}}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestResolveField_PathNotFound(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		Name string
	}

	source := &formulatypes.FieldSource{Path: "NonExistent.Field"}
	entity := &Entity{Name: "test"}
	_, err := r.ResolveField(entity, source)
	require.Error(t, err)
}

func TestResolveField_NilPointerInPath(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Inner struct {
		Name string
	}
	type Outer struct {
		Inner *Inner
	}

	source := &formulatypes.FieldSource{Path: "Inner.Name"}
	entity := &Outer{Inner: nil}
	_, err := r.ResolveField(entity, source)
	require.Error(t, err)
}

func TestResolveField_NullableNilPointerInPath(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Inner struct {
		Name string
	}
	type Outer struct {
		Inner *Inner
	}

	source := &formulatypes.FieldSource{
		Path:     "Inner.Name",
		Nullable: true,
	}
	entity := &Outer{Inner: nil}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetFieldDecimal_PtrFloat64Direct(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	val := 333.33
	entity := &ShipmentEntity{
		OtherChargeAmount: &val,
	}

	got, err := r.ResolveComputed(entity, "computeOtherChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 333.33, got, 0.001)
}

func TestGetFieldDecimal_BoolField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	type EntityWithBoolDecimal struct {
		FreightChargeAmount bool
	}

	entity := &EntityWithBoolDecimal{
		FreightChargeAmount: true,
	}

	got, err := r.ResolveComputed(entity, "computeFreightChargeAmount")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
}

func TestResolveField_JSONTagLookup(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		MyField string `json:"my_field"`
	}

	source := &formulatypes.FieldSource{Path: "my_field"}
	entity := &Entity{MyField: "found_via_tag"}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, "found_via_tag", got)
}

func TestResolveField_FieldPrecedesPath(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		Weight int64
		Pieces int64
	}

	source := &formulatypes.FieldSource{
		Field: "Weight",
		Path:  "Pieces",
	}
	entity := &Entity{Weight: 100, Pieces: 200}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, int64(200), got)
}

func TestResolveField_UsesPathWhenFieldEmpty(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	type Entity struct {
		Weight int64
	}

	source := &formulatypes.FieldSource{
		Field: "",
		Path:  "Weight",
	}
	entity := &Entity{Weight: 100}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, int64(100), got)
}
