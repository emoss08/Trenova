package agentdefinitionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func definitionWith(mode agentdefinition.TriggerMode, enabled bool) *agentdefinition.Definition {
	return &agentdefinition.Definition{TriggerMode: mode, Enabled: enabled}
}

func TestRequireAutomationRefusesTurningOnAnAutomatedAgent(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}
	tenant := plantest.Tenant()

	err := svc.requireAutomation(t.Context(), tenant, definitionWith(agentdefinition.TriggerScheduled, true), nil)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	err = svc.requireAutomation(
		t.Context(),
		tenant,
		definitionWith(agentdefinition.TriggerEvent, true),
		definitionWith(agentdefinition.TriggerChat, true),
	)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRequireAutomationAllowsChatAgentsAndUnchangedAutomation(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: mocks.NewMockPlanService(t)}
	tenant := plantest.Tenant()

	require.NoError(t, svc.requireAutomation(t.Context(), tenant, definitionWith(agentdefinition.TriggerChat, true), nil))
	require.NoError(t, svc.requireAutomation(t.Context(), tenant, definitionWith(agentdefinition.TriggerScheduled, false), nil))
	require.NoError(t, svc.requireAutomation(
		t.Context(),
		tenant,
		definitionWith(agentdefinition.TriggerScheduled, true),
		definitionWith(agentdefinition.TriggerScheduled, true),
	))
}
