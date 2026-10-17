package agentwait_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentsCannotAskForAPortalRunWait(t *testing.T) {
	t.Parallel()

	assert.True(t, agentwait.KindPortalRun.IsValid())
	assert.Contains(t, agentwait.AllKinds(), agentwait.KindPortalRun)
	assert.False(t, slices.Contains(agentwait.AgentKinds(), agentwait.KindPortalRun))
	assert.Len(t, agentwait.AgentKinds(), len(agentwait.AllKinds())-1)
}

func TestAPortalRunWaitNamesItsRunAndWatchesIt(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("aprn_")
	wait := &agentwait.Wait{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("adef_"),
		ThreadID:          pulid.MustNew("athr_"),
		Kind:              agentwait.KindPortalRun,
		Condition:         &agentwait.Condition{PortalRunID: runID},
		Description:       "the task on Acme DC to finish",
	}
	wait.Normalize()
	multiErr := errortypes.NewMultiError()
	wait.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, runID, wait.WatchID)

	missing := &agentwait.Wait{
		OrganizationID:    wait.OrganizationID,
		BusinessUnitID:    wait.BusinessUnitID,
		AgentDefinitionID: wait.AgentDefinitionID,
		ThreadID:          wait.ThreadID,
		Kind:              agentwait.KindPortalRun,
		Description:       "the task to finish",
	}
	missing.Normalize()
	multiErr = errortypes.NewMultiError()
	missing.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "portalRunId", multiErr.Errors[0].Field)
}

func TestOnlyAPortalRunsAnswerIsOutsideText(t *testing.T) {
	t.Parallel()

	answered := &agentwait.Wait{Kind: agentwait.KindPortalRun, Outcome: "Booked for 8:00 AM"}
	assert.True(t, answered.CarriesOutsideText())

	assert.False(t, (&agentwait.Wait{Kind: agentwait.KindPortalRun}).CarriesOutsideText())
	assert.False(t, (&agentwait.Wait{Kind: agentwait.KindReply, Outcome: "A reply came in"}).
		CarriesOutsideText())
}
