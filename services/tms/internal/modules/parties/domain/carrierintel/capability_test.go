package carrierintel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilitySetBestDepth(t *testing.T) {
	t.Parallel()

	fmcsaOnly := carrierintel.NewCapabilitySet(carrierintel.CapabilityLookupFMCSA)
	depth, ok := fmcsaOnly.BestDepth(carrierintel.LookupDepthFull)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthFMCSA, depth)

	all := carrierintel.NewCapabilitySet(
		carrierintel.CapabilityLookupFMCSA,
		carrierintel.CapabilityLookupLite,
		carrierintel.CapabilityLookupFull,
	)
	depth, ok = all.BestDepth(carrierintel.LookupDepthLite)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthLite, depth)

	noLite := carrierintel.NewCapabilitySet(
		carrierintel.CapabilityLookupFMCSA,
		carrierintel.CapabilityLookupFull,
	)
	depth, ok = noLite.BestDepth(carrierintel.LookupDepthLite)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthFull, depth)

	_, ok = carrierintel.NewCapabilitySet().BestDepth(carrierintel.LookupDepthFMCSA)
	assert.False(t, ok)
}
