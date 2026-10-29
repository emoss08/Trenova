package carrierintelsourcingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaidCarrierLookupsRefuseWhenThePlanRestrictsThem(t *testing.T) {
	t.Parallel()

	svc := &Service{
		plans: plantest.Restricting(t, platformplan.CapabilityCarrierIntelligencePaid),
	}
	tenant := plantest.Tenant()

	page, err := svc.SearchCarriers(t.Context(), &SourcingQuery{TenantInfo: tenant, Text: "Acme"})
	require.Nil(t, page)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	suggestions, err := svc.AutocompleteCarriers(t.Context(), tenant, "Acme Trucking", 5)
	require.NoError(t, err)
	assert.Empty(t, suggestions)
}
