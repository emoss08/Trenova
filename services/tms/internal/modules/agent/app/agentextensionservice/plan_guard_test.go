package agentextensionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveCapabilitiesOfferNothingThePlanRestricts(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})
	f.svc.plans = plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)

	active, err := f.svc.ActiveCapabilities(t.Context(), f.tenant)

	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestUpdateConfigRefusesEnablingWebSearchThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)}

	resp, err := svc.UpdateConfig(t.Context(), agentextension.TypeExa, &agentextensionports.UpdateAgentExtensionRequest{
		TenantInfo: plantest.Tenant(),
		Enabled:    true,
	})

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestServingTellsTheAgentWebResearchIsNotOnThePlan(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentWebSearch)}

	settings, err := svc.serving(t.Context(), plantest.Tenant(), agentextension.CapabilityWebResearch)

	require.Nil(t, settings)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "plan")
}
