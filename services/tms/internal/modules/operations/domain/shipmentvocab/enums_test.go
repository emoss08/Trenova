package shipmentvocab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreightTerms_IsValid(t *testing.T) {
	t.Parallel()

	require.True(t, FreightTermsPrepaid.IsValid())
	require.True(t, FreightTermsCollect.IsValid())
	require.True(t, FreightTermsThirdParty.IsValid())
	require.False(t, FreightTerms("").IsValid())
	require.False(t, FreightTerms("Consignee").IsValid())
}

func TestMoveCoverageType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value MoveCoverageType
		want  bool
	}{
		{name: "unassigned", value: MoveCoverageTypeUnassigned, want: true},
		{name: "driver", value: MoveCoverageTypeDriver, want: true},
		{name: "carrier", value: MoveCoverageTypeCarrier, want: true},
		{name: "empty", value: MoveCoverageType(""), want: false},
		{name: "unknown", value: MoveCoverageType("broker"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.value.IsValid())
		})
	}
}
