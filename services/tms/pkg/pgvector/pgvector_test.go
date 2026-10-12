package pgvector

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
			name: "empty extension version",
			row:  vectorProbeRow{ExtensionVersion: strptr("")},
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

func TestVectorSupportReadyOnlyWhenReady(t *testing.T) {
	t.Parallel()

	assert.True(t, VectorSupport{State: VectorReady, ExtensionVersion: "0.8.1"}.Ready())
	assert.False(t, VectorSupport{State: VectorSchemaMissing, ExtensionVersion: "0.8.1"}.Ready())
	assert.False(t, VectorSupport{State: VectorExtensionTooOld, ExtensionVersion: "0.7.4"}.Ready())
	assert.False(t, VectorSupport{State: VectorExtensionMissing}.Ready())
	assert.False(t, VectorSupport{}.Ready())
}

func TestVectorSupportExtensionInstalled(t *testing.T) {
	t.Parallel()

	assert.True(t, VectorSupport{State: VectorSchemaMissing, ExtensionVersion: "0.8.1"}.
		ExtensionInstalled())
	assert.False(t, VectorSupport{State: VectorExtensionMissing}.ExtensionInstalled())
}
