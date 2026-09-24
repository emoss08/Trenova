package dbdialect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func strptr(s string) *string { return &s }

func TestClassifyVector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  vectorProbeRow
		want VectorSupport
	}{
		{
			name: "no extension",
			row:  vectorProbeRow{},
			want: VectorSupport{State: VectorExtensionMissing},
		},
		{
			name: "extension too old",
			row:  vectorProbeRow{ExtensionVersion: strptr("0.7.4"), SchemaPresent: true},
			want: VectorSupport{State: VectorExtensionTooOld, ExtensionVersion: "0.7.4"},
		},
		{
			name: "extension installed after the migrations ran",
			row:  vectorProbeRow{ExtensionVersion: strptr("0.8.1"), VersionSupported: true},
			want: VectorSupport{State: VectorSchemaMissing, ExtensionVersion: "0.8.1"},
		},
		{
			name: "ready",
			row: vectorProbeRow{
				ExtensionVersion: strptr("0.8.1"),
				VersionSupported: true,
				SchemaPresent:    true,
			},
			want: VectorSupport{State: VectorReady, ExtensionVersion: "0.8.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, classifyVector(tt.row))
		})
	}
}

func TestVectorSearchIsNeverAssumedFromTheDialect(t *testing.T) {
	t.Parallel()

	assert.False(t, Postgres.Supports(CapVectorSearch))
	assert.False(t, SQLite.Supports(CapVectorSearch))
	assert.True(t, CapVectorSearch.IsProbed())
	assert.False(t, CapPostGIS.IsProbed())
}

func TestCapabilitiesSupportsVectorSearchOnlyWhenReady(t *testing.T) {
	t.Parallel()

	ready := VectorSupport{State: VectorReady, ExtensionVersion: "0.8.1"}

	assert.True(t, NewCapabilities(Postgres, ready).Supports(CapVectorSearch))
	assert.False(t, NewCapabilities(Postgres, VectorSupport{State: VectorSchemaMissing}).
		Supports(CapVectorSearch))
	assert.False(t, NewCapabilities(SQLite, ready).Supports(CapVectorSearch))
	assert.Equal(t, VectorUnsupported, NewCapabilities(SQLite, ready).Vector().State)
	assert.True(t, NewCapabilities(Postgres, ready).Supports(CapPostGIS))
	assert.False(t, NewCapabilities(SQLite, ready).Supports(CapPostGIS))
}
