package resolver_test

import (
	"github.com/shopspring/decimal"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
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

	_, err := r.ResolveField(&FieldTestEntity{}, nil)
	require.Error(t, err)
}

func TestResolveField_ComputedSource(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{Computed: true, Function: "computeSomething"}
	_, err := r.ResolveField(&FieldTestEntity{}, source)
	require.Error(t, err)
}

func TestResolveField_EmptyPathAndField(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	source := &formulatypes.FieldSource{}
	_, err := r.ResolveField(&FieldTestEntity{}, source)
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
		Transform: "integer",
	}

	entity := &Entity{Value: 42}
	got, err := r.ResolveField(entity, source)
	require.NoError(t, err)
	assert.Equal(t, int64(42), got)
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

	weight := int64(5000)
	pieces := int64(100)
	miles := 100.0
	entity := &shipment.Shipment{
		Weight: &weight,
		Pieces: &pieces,
		Moves:  []*shipment.ShipmentMove{{Distance: &miles}},
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
	assert.Equal(t, "100", result["totalDistance"].(decimal.Decimal).String())
}

func TestResolveAllFields_NullableFieldError(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()

	entity := &FieldTestEntity{}

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

	entity := &FieldTestEntity{}

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
