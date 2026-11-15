package agentdefinitionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func definitionWith(mode agentdefinition.TriggerMode, enabled bool) *agentdefinition.Definition {
	return &agentdefinition.Definition{TriggerMode: mode, Enabled: enabled}
}

func TestRequireAutomationRefusesTurningOnAnAutomatedAgent(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAgentAutomation)}

	err := svc.requireAutomation(t.Context(), definitionWith(agentdefinition.TriggerScheduled, true), nil)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	err = svc.requireAutomation(
		t.Context(),
		definitionWith(agentdefinition.TriggerEvent, true),
		definitionWith(agentdefinition.TriggerChat, true),
	)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRequireAutomationAllowsChatAgentsAndUnchangedAutomation(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.AllFeatures()}

	require.NoError(t, svc.requireAutomation(t.Context(), definitionWith(agentdefinition.TriggerChat, true), nil))
	require.NoError(t, svc.requireAutomation(t.Context(), definitionWith(agentdefinition.TriggerScheduled, false), nil))
	require.NoError(t, svc.requireAutomation(
		t.Context(),
		definitionWith(agentdefinition.TriggerScheduled, true),
		definitionWith(agentdefinition.TriggerScheduled, true),
	))
}
