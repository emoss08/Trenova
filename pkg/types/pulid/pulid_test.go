package pulid_test

import (
	"database/sql/driver"
	"testing"

	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID_IsNil(t *testing.T) {
	tests := []struct {
		name string
		id   pulid.ID
		want bool
	}{
		{
			name: "empty ID is nil",
			id:   pulid.ID(""),
			want: true,
		},
		{
			name: "non-empty ID is not nil",
			id:   pulid.ID("prefix01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.id.IsNil())
			assert.Equal(t, !tt.want, tt.id.IsNotNil())
		})
	}
}

func TestMustNew(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "creates ID with prefix",
			prefix: "test",
		},
		{
			name:   "creates ID with empty prefix",
			prefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := pulid.MustNew(tt.prefix)
			assert.NotEmpty(t, id)
			if tt.prefix != "" {
				assert.Greater(t, len(id), len(tt.prefix))
				assert.Equal(t, tt.prefix, string(id[:len(tt.prefix)]))
			}
		})
	}
}

func TestMustNewPtr(t *testing.T) {
	prefix := "test"
	id := pulid.MustNewPtr(prefix)
	require.NotNil(t, id)
	assert.NotEmpty(t, *id)
	assert.Greater(t, len(*id), len(prefix))
	assert.Equal(t, prefix, string((*id)[:len(prefix)]))
}

func TestMust(t *testing.T) {
	prefix := "test"
	id := pulid.Must(prefix)
	require.NotNil(t, id)
	assert.NotEmpty(t, *id)
	assert.Greater(t, len(*id), len(prefix))
	assert.Equal(t, prefix, string((*id)[:len(prefix)]))
}

func TestID_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
		want    pulid.ID
	}{
		{
			name:    "scan from string",
			input:   "test01H2XJKJ8J1RQTF5TZQJ1GY4RX",
			wantErr: false,
			want:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
		},
		{
			name:    "scan from ID",
			input:   pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			wantErr: false,
			want:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
		},
		{
			name:    "scan from nil",
			input:   nil,
			wantErr: true,
			want:    "",
		},
		{
			name:    "scan from invalid type",
			input:   123,
			wantErr: true,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id pulid.ID
			err := id.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, id)
			}
		})
	}
}

func TestID_Value(t *testing.T) {
	tests := []struct {
		name    string
		id      pulid.ID
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "non-nil ID",
			id:      pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			want:    "test01H2XJKJ8J1RQTF5TZQJ1GY4RX",
			wantErr: false,
		},
		{
			name:    "nil ID",
			id:      pulid.ID(""),
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.id.Value()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestID_String(t *testing.T) {
	id := pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX")
	assert.Equal(t, "test01H2XJKJ8J1RQTF5TZQJ1GY4RX", id.String())
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    pulid.ID
		wantErr bool
	}{
		{
			name:    "valid PULID",
			input:   "test01H2XJKJ8J1RQTF5TZQJ1GY4RX",
			want:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			wantErr: false,
		},
		{
			name:    "too short",
			input:   "test",
			want:    pulid.Nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pulid.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    pulid.ID
		wantErr bool
	}{
		{
			name:    "valid PULID",
			input:   "test01H2XJKJ8J1RQTF5TZQJ1GY4RX",
			want:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			wantErr: false,
		},
		{
			name:    "invalid PULID",
			input:   "test",
			want:    pulid.Nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pulid.MustParse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		name string
		a    pulid.ID
		b    pulid.ID
		want bool
	}{
		{
			name: "equal IDs",
			a:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			b:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			want: true,
		},
		{
			name: "different IDs",
			a:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			b:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RY"),
			want: false,
		},
		{
			name: "one nil ID",
			a:    pulid.ID("test01H2XJKJ8J1RQTF5TZQJ1GY4RX"),
			b:    pulid.Nil,
			want: false,
		},
		{
			name: "both nil IDs",
			a:    pulid.Nil,
			b:    pulid.Nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pulid.Equals(tt.a, tt.b))
		})
	}
}

func TestDefaultEntropySource(t *testing.T) {
	// Test that multiple IDs generated in quick succession are unique
	ids := make(map[pulid.ID]bool)
	for i := 0; i < 1000; i++ {
		id := pulid.MustNew("test")
		assert.False(t, ids[id], "Generated duplicate ID: %s", id)
		ids[id] = true
	}
}
