package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPartnerReadiness(t *testing.T) {
	t.Parallel()

	ready := buildPartnerReadiness(&repositories.EDIPartnerReadinessRow{
		PartnerID:             pulid.MustNew("edip_"),
		ContactEmail:          "ops@partner.example",
		Timezone:              "America/Chicago",
		HasActiveProfile:      true,
		HasMappingProfile:     true,
		HasInboundDocProfile:  true,
		HasOutboundDocProfile: true,
		HasPassingTestCase:    true,
		EnabledForInbound:     true,
		EnabledForOutbound:    true,
		Kind:                  "External",
	})
	assert.True(t, ready.Ready)
	assert.Equal(t, ready.TotalCount, ready.CompletedCount)
	assert.Len(t, ready.Items, 6)

	partial := buildPartnerReadiness(&repositories.EDIPartnerReadinessRow{
		PartnerID:          pulid.MustNew("edip_"),
		HasActiveProfile:   true,
		EnabledForOutbound: true,
		Kind:               "External",
	})
	assert.False(t, partial.Ready)
	assert.Equal(t, 5, partial.TotalCount)
	assert.Equal(t, 1, partial.CompletedCount)

	internal := buildPartnerReadiness(&repositories.EDIPartnerReadinessRow{
		PartnerID:         pulid.MustNew("edip_"),
		ContactEmail:      "edi@internal.example",
		Timezone:          "UTC",
		HasActiveProfile:  true,
		HasMappingProfile: true,
		Kind:              "Internal",
	})
	require.NotEmpty(t, internal.Items)
	for _, item := range internal.Items {
		assert.NotEqual(t, "test-case", item.Key)
	}
}
