package maputils_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/utils/maputils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractStringField(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		key  string
		want string
	}{
		{
			name: "valid string field",
			data: map[string]any{"name": "John Doe"},
			key:  "name",
			want: "John Doe",
		},
		{
			name: "missing field",
			data: map[string]any{"name": "John Doe"},
			key:  "email",
			want: "",
		},
		{
			name: "non-string field",
			data: map[string]any{"age": 25},
			key:  "age",
			want: "",
		},
		{
			name: "empty string",
			data: map[string]any{"name": ""},
			key:  "name",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maputils.ExtractStringField(tt.data, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractInt64Field(t *testing.T) {
	tests := []struct {
		name  string
		field any
		want  int64
	}{
		{
			name:  "int64 value",
			field: int64(100),
			want:  100,
		},
		{
			name:  "float64 value",
			field: float64(99.5),
			want:  99,
		},
		{
			name:  "int value",
			field: int(50),
			want:  50,
		},
		{
			name:  "nil value",
			field: nil,
			want:  0,
		},
		{
			name:  "map with long field (int64)",
			field: map[string]any{"long": int64(200)},
			want:  200,
		},
		{
			name:  "map with long field (float64)",
			field: map[string]any{"long": float64(150.7)},
			want:  150,
		},
		{
			name:  "map without long field",
			field: map[string]any{"other": 100},
			want:  0,
		},
		{
			name:  "string value",
			field: "not a number",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maputils.ExtractInt64Field(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid string field",
			data:    map[string]any{"name": "Jane Smith"},
			key:     "name",
			want:    "Jane Smith",
			wantErr: false,
		},
		{
			name:    "missing field",
			data:    map[string]any{"name": "Jane Smith"},
			key:     "email",
			want:    "",
			wantErr: true,
		},
		{
			name:    "non-string field",
			data:    map[string]any{"age": 30},
			key:     "age",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			data:    map[string]any{"name": ""},
			key:     "name",
			want:    "",
			wantErr: false,
		},
		{
			name:    "nil value",
			data:    map[string]any{"name": nil},
			key:     "name",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := maputils.GetString(tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		key     string
		want    int
		wantErr bool
	}{
		{
			name:    "float64 value",
			data:    map[string]any{"count": float64(42)},
			key:     "count",
			want:    42,
			wantErr: false,
		},
		{
			name:    "int value",
			data:    map[string]any{"count": int(25)},
			key:     "count",
			want:    25,
			wantErr: false,
		},
		{
			name:    "int64 value",
			data:    map[string]any{"count": int64(100)},
			key:     "count",
			want:    100,
			wantErr: false,
		},
		{
			name:    "missing field",
			data:    map[string]any{"count": 10},
			key:     "total",
			want:    0,
			wantErr: true,
		},
		{
			name:    "string value",
			data:    map[string]any{"count": "not a number"},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "nil value",
			data:    map[string]any{"count": nil},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "float64 with decimal",
			data:    map[string]any{"count": float64(42.7)},
			key:     "count",
			want:    42,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := maputils.GetInt(tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "true value",
			data:    map[string]any{"active": true},
			key:     "active",
			want:    true,
			wantErr: false,
		},
		{
			name:    "false value",
			data:    map[string]any{"active": false},
			key:     "active",
			want:    false,
			wantErr: false,
		},
		{
			name:    "missing field",
			data:    map[string]any{"active": true},
			key:     "enabled",
			want:    false,
			wantErr: true,
		},
		{
			name:    "non-bool field",
			data:    map[string]any{"active": "true"},
			key:     "active",
			want:    false,
			wantErr: true,
		},
		{
			name:    "nil value",
			data:    map[string]any{"active": nil},
			key:     "active",
			want:    false,
			wantErr: true,
		},
		{
			name:    "int value",
			data:    map[string]any{"active": 1},
			key:     "active",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := maputils.GetBool(tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetMap(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		key     string
		want    map[string]any
		wantErr bool
	}{
		{
			name: "valid map",
			data: map[string]any{
				"config": map[string]any{
					"timeout": 30,
					"retry":   true,
				},
			},
			key: "config",
			want: map[string]any{
				"timeout": 30,
				"retry":   true,
			},
			wantErr: false,
		},
		{
			name: "empty map",
			data: map[string]any{
				"config": map[string]any{},
			},
			key:     "config",
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name: "missing field",
			data: map[string]any{
				"config": map[string]any{},
			},
			key:     "settings",
			want:    nil,
			wantErr: true,
		},
		{
			name: "non-map field",
			data: map[string]any{
				"config": "not a map",
			},
			key:     "config",
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil value",
			data: map[string]any{
				"config": nil,
			},
			key:     "config",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := maputils.GetMap(tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetArray(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		key     string
		want    []any
		wantErr bool
	}{
		{
			name: "valid array",
			data: map[string]any{
				"items": []any{"apple", "banana", "cherry"},
			},
			key:     "items",
			want:    []any{"apple", "banana", "cherry"},
			wantErr: false,
		},
		{
			name: "empty array",
			data: map[string]any{
				"items": []any{},
			},
			key:     "items",
			want:    []any{},
			wantErr: false,
		},
		{
			name: "missing field returns nil without error",
			data: map[string]any{
				"items": []any{},
			},
			key:     "nodes",
			want:    nil,
			wantErr: false,
		},
		{
			name: "non-array field",
			data: map[string]any{
				"items": "not an array",
			},
			key:     "items",
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil value returns nil without error",
			data: map[string]any{
				"items": nil,
			},
			key:     "items",
			want:    nil,
			wantErr: false,
		},
		{
			name: "mixed type array",
			data: map[string]any{
				"items": []any{1, "two", true, nil},
			},
			key:     "items",
			want:    []any{1, "two", true, nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := maputils.GetArray(tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
