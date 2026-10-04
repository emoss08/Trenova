package agentextensionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveExtensionsOffersNothingThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)}

	active, err := svc.ActiveExtensions(t.Context(), plantest.Tenant())

	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestUpdateConfigRefusesEnablingWebSearchThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)}

	resp, err := svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo: plantest.Tenant(),
		Enabled:    true,
	})

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRuntimeTellsTheAgentWebSearchIsNotOnThePlan(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)}

	settings, err := svc.runtime(t.Context(), plantest.Tenant(), agentextension.TypeExa, true)

	require.Nil(t, settings)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "plan")
}
