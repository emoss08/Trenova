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

// A file's wait is set by the system when the agent finds the file still
// being read, never asked for with wait_until.
func TestAgentsCannotAskForADocumentReadWait(t *testing.T) {
	t.Parallel()

	assert.True(t, agentwait.KindDocumentRead.IsValid())
	assert.Contains(t, agentwait.AllKinds(), agentwait.KindDocumentRead)
	assert.False(t, slices.Contains(agentwait.AgentKinds(), agentwait.KindDocumentRead))
}

// The timer reads the file again, so a reading that saves no extraction (a
// file that could not be read) still ends the wait.
func TestADocumentReadWaitIsTimed(t *testing.T) {
	t.Parallel()

	assert.True(t, agentwait.KindDocumentRead.Timed())
	assert.False(t, agentwait.KindDocumentRead.Visit())
}

func TestADocumentReadWaitNamesItsFileAndWatchesIt(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	wait := &agentwait.Wait{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("adef_"),
		ThreadID:          pulid.MustNew("athr_"),
		Kind:              agentwait.KindDocumentRead,
		Condition:         &agentwait.Condition{DocumentID: documentID},
		Description:       "rate-con.pdf to finish being read",
	}
	wait.Normalize()
	multiErr := errortypes.NewMultiError()
	wait.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, documentID, wait.WatchID)

	missing := &agentwait.Wait{
		OrganizationID:    wait.OrganizationID,
		BusinessUnitID:    wait.BusinessUnitID,
		AgentDefinitionID: wait.AgentDefinitionID,
		ThreadID:          wait.ThreadID,
		Kind:              agentwait.KindDocumentRead,
		Description:       "a file to finish being read",
	}
	missing.Normalize()
	multiErr = errortypes.NewMultiError()
	missing.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "documentId", multiErr.Errors[0].Field)
}
