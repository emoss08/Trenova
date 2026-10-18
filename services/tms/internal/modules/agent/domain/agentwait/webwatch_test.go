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

func TestAgentsCannotAskForAWebWatchWait(t *testing.T) {
	t.Parallel()

	assert.True(t, agentwait.KindWebWatch.IsValid())
	assert.Contains(t, agentwait.AllKinds(), agentwait.KindWebWatch)
	assert.False(t, slices.Contains(agentwait.AgentKinds(), agentwait.KindWebWatch))
}

func TestAWebWatchWaitLastsAsLongAsTheWatch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(agentwait.MaxWebWatchLifetimeSeconds),
		agentwait.KindWebWatch.MaxLifetime())
	assert.Greater(t, agentwait.KindWebWatch.MaxLifetime(), int64(90*24*60*60),
		"a ninety-day watch keeps its wait to its last check")
	assert.Equal(t, int64(agentwait.MaxLifetimeSeconds), agentwait.KindTime.MaxLifetime())
}

func TestAWebWatchWaitNamesItsWatchAndWatchesIt(t *testing.T) {
	t.Parallel()

	watchID := pulid.MustNew("awwt_")
	wait := &agentwait.Wait{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("adef_"),
		ThreadID:          pulid.MustNew("athr_"),
		Kind:              agentwait.KindWebWatch,
		Condition:         &agentwait.Condition{WebWatchID: watchID},
		Description:       "the web watch for the broker rule to find something new",
	}
	wait.Normalize()
	multiErr := errortypes.NewMultiError()
	wait.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, watchID, wait.WatchID)

	missing := &agentwait.Wait{
		OrganizationID:    wait.OrganizationID,
		BusinessUnitID:    wait.BusinessUnitID,
		AgentDefinitionID: wait.AgentDefinitionID,
		ThreadID:          wait.ThreadID,
		Kind:              agentwait.KindWebWatch,
		Description:       "the web watch to find something",
	}
	missing.Normalize()
	multiErr = errortypes.NewMultiError()
	missing.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "webWatchId", multiErr.Errors[0].Field)
}

func TestWhatAWebWatchFoundIsOutsideText(t *testing.T) {
	t.Parallel()

	found := &agentwait.Wait{Kind: agentwait.KindWebWatch, Outcome: "The web watch found…"}
	assert.True(t, found.CarriesOutsideText())
	assert.False(t, (&agentwait.Wait{Kind: agentwait.KindWebWatch}).CarriesOutsideText())
}
