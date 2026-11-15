package entitlementservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/entitlementservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowAllEnablesEveryCatalogFeature(t *testing.T) {
	t.Parallel()

	gate := entitlementservice.NewAllowAll()
	features := platformcatalog.Features()
	require.NotEmpty(t, features)

	for _, feature := range features {
		enabled, err := gate.Enabled(t.Context(), feature.Key)
		require.NoError(t, err, feature.Key)
		assert.True(t, enabled, feature.Key)
		require.NoError(t, platformcatalogports.RequireFeature(t.Context(), gate, feature.Key))
	}
}
