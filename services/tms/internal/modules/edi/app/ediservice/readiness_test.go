package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPartnerReadiness(t *testing.T) {
	t.Parallel()

	ready := buildPartnerReadiness(&ediports.EDIPartnerReadinessRow{
		PartnerID:          pulid.MustNew("edip_"),
		ContactEmail:       "ops@partner.example",
		Timezone:           "America/Chicago",
		HasActiveProfile:   true,
		HasMappingProfile:  true,
		HasLadingProject:   true,
		HasActiveVersion:   true,
		EnabledForInbound:  true,
		EnabledForOutbound: true,
		Kind:               "External",
	})
	assert.True(t, ready.Ready)
	assert.Equal(t, ready.TotalCount, ready.CompletedCount)
	assert.Len(t, ready.Items, 5)

	partial := buildPartnerReadiness(&ediports.EDIPartnerReadinessRow{
		PartnerID:          pulid.MustNew("edip_"),
		HasActiveProfile:   true,
		EnabledForOutbound: true,
		Kind:               "External",
	})
	assert.False(t, partial.Ready)
	assert.Equal(t, 5, partial.TotalCount)
	assert.Equal(t, 1, partial.CompletedCount)

	internal := buildPartnerReadiness(&ediports.EDIPartnerReadinessRow{
		PartnerID:         pulid.MustNew("edip_"),
		ContactEmail:      "edi@internal.example",
		Timezone:          "UTC",
		HasActiveProfile:  true,
		HasMappingProfile: true,
		Kind:              "Internal",
	})
	require.NotEmpty(t, internal.Items)
	for _, item := range internal.Items {
		assert.NotEqual(t, "lading-project", item.Key)
		assert.NotEqual(t, "lading-active-version", item.Key)
	}
}
