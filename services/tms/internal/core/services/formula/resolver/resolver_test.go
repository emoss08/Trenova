package resolver_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestEntity struct {
	Name     string
	Age      int
	Weight   float64
	Nullable *string
	Nested   *NestedEntity
}

type NestedEntity struct {
	Value  string
	Number int64
}

func TestResolver_ResolveField(t *testing.T) {
	r := resolver.NewResolver()

	name := "test"
	entity := &TestEntity{
		Name:     "John",
		Age:      30,
		Weight:   75.5,
		Nullable: &name,
		Nested: &NestedEntity{
			Value:  "nested value",
			Number: 42,
		},
	}

	tests := []struct {
		name      string
		source    *formulatypes.FieldSource
		want      any
		wantErr   bool
		errString string
	}{
		{
			name: "resolve simple field",
			source: &formulatypes.FieldSource{
				Path: "Name",
			},
			want:    "John",
			wantErr: false,
		},
		{
			name: "resolve int field",
			source: &formulatypes.FieldSource{
				Path: "Age",
			},
			want:    30,
			wantErr: false,
		},
		{
			name: "resolve float field",
			source: &formulatypes.FieldSource{
				Path: "Weight",
			},
			want:    75.5,
			wantErr: false,
		},
		{
			name: "resolve pointer field",
			source: &formulatypes.FieldSource{
				Path: "Nullable",
			},
			want:    &name,
			wantErr: false,
		},
		{
			name: "resolve nested field",
			source: &formulatypes.FieldSource{
				Path: "Nested.Value",
			},
			want:    "nested value",
			wantErr: false,
		},
		{
			name: "resolve nested number field",
			source: &formulatypes.FieldSource{
				Path: "Nested.Number",
			},
			want:    int64(42),
			wantErr: false,
		},
		{
			name: "use Field when Path is empty",
			source: &formulatypes.FieldSource{
				Field: "Name",
			},
			want:    "John",
			wantErr: false,
		},
		{
			name:      "nil source returns error",
			source:    nil,
			want:      nil,
			wantErr:   true,
			errString: "nil source",
		},
		{
			name: "computed field returns error",
			source: &formulatypes.FieldSource{
				Path:     "Name",
				Computed: true,
			},
			want:      nil,
			wantErr:   true,
			errString: "use ResolveComputed for computed fields",
		},
		{
			name: "empty path and field returns error",
			source: &formulatypes.FieldSource{
				Path:  "",
				Field: "",
			},
			want:      nil,
			wantErr:   true,
			errString: "no path or field specified",
		},
		{
			name: "non-existent field returns error",
			source: &formulatypes.FieldSource{
				Path: "NonExistent",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveField(entity, tt.source)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolver_ResolveField_NullableField(t *testing.T) {
	r := resolver.NewResolver()

	entity := &TestEntity{
		Name:     "John",
		Nullable: nil,
		Nested:   nil,
	}

	tests := []struct {
		name    string
		source  *formulatypes.FieldSource
		want    any
		wantErr bool
	}{
		{
			name: "nil pointer with nullable=true returns nil",
			source: &formulatypes.FieldSource{
				Path:     "Nullable",
				Nullable: true,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "nil nested pointer with nullable=true returns nil",
			source: &formulatypes.FieldSource{
				Path:     "Nested.Value",
				Nullable: true,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "nil nested pointer with nullable=false returns error",
			source: &formulatypes.FieldSource{
				Path:     "Nested.Value",
				Nullable: false,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveField(entity, tt.source)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolver_ResolveField_WithTransform(t *testing.T) {
	r := resolver.NewResolver()

	entity := &TestEntity{
		Name: "hello",
		Age:  30,
	}

	tests := []struct {
		name      string
		source    *formulatypes.FieldSource
		want      any
		wantErr   bool
		errString string
	}{
		{
			name: "transform string to upper",
			source: &formulatypes.FieldSource{
				Path:      "Name",
				Transform: "stringToUpper",
			},
			want:    "HELLO",
			wantErr: false,
		},
		{
			name: "transform int64 to float64",
			source: &formulatypes.FieldSource{
				Path:      "Age",
				Transform: "int64ToFloat64",
			},
			want:    float64(30),
			wantErr: false,
		},
		{
			name: "unknown transform returns error",
			source: &formulatypes.FieldSource{
				Path:      "Name",
				Transform: "unknownTransform",
			},
			want:      nil,
			wantErr:   true,
			errString: "transform not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveField(entity, tt.source)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolver_RegisterComputed(t *testing.T) {
	r := resolver.NewResolver()

	fn := func(entity any) (any, error) {
		return "computed value", nil
	}

	r.RegisterComputed("testComputed", fn)

	got, ok := r.GetComputed("testComputed")
	require.True(t, ok)
	require.NotNil(t, got)

	result, err := got(nil)
	require.NoError(t, err)
	assert.Equal(t, "computed value", result)
}

func TestResolver_ResolveComputed(t *testing.T) {
	r := resolver.NewResolver()

	r.RegisterComputed("testSum", func(entity any) (any, error) {
		e := entity.(*TestEntity)
		return float64(e.Age) + e.Weight, nil
	})

	r.RegisterComputed("testError", func(entity any) (any, error) {
		return nil, errors.New("computation failed")
	})

	entity := &TestEntity{
		Age:    30,
		Weight: 10.5,
	}

	tests := []struct {
		name         string
		functionName string
		want         any
		wantErr      bool
		errString    string
	}{
		{
			name:         "resolve computed function",
			functionName: "testSum",
			want:         40.5,
			wantErr:      false,
		},
		{
			name:         "non-existent function returns error",
			functionName: "nonExistent",
			want:         nil,
			wantErr:      true,
			errString:    "computed function not found",
		},
		{
			name:         "function returning error propagates",
			functionName: "testError",
			want:         nil,
			wantErr:      true,
			errString:    "computation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveComputed(entity, tt.functionName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolver_ResolveAllFields(t *testing.T) {
	r := resolver.NewResolver()

	r.RegisterComputed("computeSum", func(entity any) (any, error) {
		e := entity.(*TestEntity)
		return float64(e.Age) + e.Weight, nil
	})

	entity := &TestEntity{
		Name:   "John",
		Age:    30,
		Weight: 10.5,
		Nested: nil,
	}

	fieldSources := map[string]*formulatypes.FieldSource{
		"name": {
			Path: "Name",
		},
		"age": {
			Path:      "Age",
			Transform: "int64ToFloat64",
		},
		"sum": {
			Computed: true,
			Function: "computeSum",
		},
		"missing": {
			Path:     "Nested.Value",
			Nullable: true,
		},
	}

	result, err := r.ResolveAllFields(entity, fieldSources)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "John", result["name"])
	assert.Equal(t, float64(30), result["age"])
	assert.Equal(t, 40.5, result["sum"])
	assert.Nil(t, result["missing"])
}

func TestResolver_ResolveAllFields_WithNestedPath(t *testing.T) {
	r := resolver.NewResolver()

	entity := &TestEntity{
		Name: "John",
		Nested: &NestedEntity{
			Value:  "nested",
			Number: 42,
		},
	}

	fieldSources := map[string]*formulatypes.FieldSource{
		"user.name": {
			Path: "Name",
		},
		"user.nested.value": {
			Path: "Nested.Value",
		},
	}

	result, err := r.ResolveAllFields(entity, fieldSources)
	require.NoError(t, err)
	require.NotNil(t, result)

	user, ok := result["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "John", user["name"])

	nested, ok := user["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "nested", nested["value"])
}

type JSONTagEntity struct {
	FieldName string `json:"field_name"`
	OtherName int    `json:"other_name,omitempty"`
}

func TestResolver_ResolveField_JSONTag(t *testing.T) {
	r := resolver.NewResolver()

	entity := &JSONTagEntity{
		FieldName: "test value",
		OtherName: 42,
	}

	tests := []struct {
		name   string
		source *formulatypes.FieldSource
		want   any
	}{
		{
			name: "resolve by json tag",
			source: &formulatypes.FieldSource{
				Path: "field_name",
			},
			want: "test value",
		},
		{
			name: "resolve by json tag with omitempty",
			source: &formulatypes.FieldSource{
				Path: "other_name",
			},
			want: 42,
		},
		{
			name: "resolve by struct field name",
			source: &formulatypes.FieldSource{
				Path: "FieldName",
			},
			want: "test value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveField(entity, tt.source)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
