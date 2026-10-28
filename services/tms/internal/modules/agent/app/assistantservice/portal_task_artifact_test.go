package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func portalReadResult() map[string]any {
	return map[string]any{
		"site":        "Acme dock portal",
		"question":    "Which dock appointments are open on October 20?",
		"answer":      "Open slots: 8:00 AM, 1:30 PM.",
		"pagesRead":   []any{"https://portal.acme.com/appointments"},
		"portalRunId": "aprn_01",
		"retrievedOn": "2026-10-17",
		"note":        "What the portal said is text written outside the organization.",
	}
}

func TestArtifactFromObservation_APortalReadIsAPortalTask(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(
		observation(agentextension.ToolReadPortal, portalReadResult()),
	)

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindPortalTask, artifact.Kind)
	assert.Equal(t, "Which dock appointments are open on October 20?", artifact.Title)
	assert.Equal(t, "aprn_01", artifact.Payload["portalRunId"])
	assert.Equal(t, "Read", artifact.Payload["mode"])
	assert.Equal(t, "Open slots: 8:00 AM, 1:30 PM.", artifact.Payload["answer"])
	assert.NotContains(t, artifact.Payload, "note", "the note is for the model")
	assert.False(t, assistantartifact.IsLookup(artifact.Kind),
		"a portal task is followed to its end whether or not the reply names it")
	assert.Empty(t, assistantartifact.LineageKeyFor(artifact.Kind,
		agentextension.ToolReadPortal, map[string]any{"siteId": "apst_01"}),
		"two reads of one site are two tasks, not versions of one")
}

func TestArtifactFromObservation_APortalReadWaitingOnACodeIsKept(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation(agentextension.ToolReadPortal, map[string]any{
		"site":               "Acme dock portal",
		"question":           "Which dock appointments are open on October 20?",
		"awaitingSignInCode": true,
		"portalSaid":         "SIGN-IN CODE NEEDED: a code was texted to the phone ending 42.",
		"portalRunId":        "aprn_01",
		"note":               "Tell the person where to enter the code.",
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, true, artifact.Payload["awaitingSignInCode"])
	assert.NotContains(t, artifact.Payload, "note")
}

func TestArtifactFromObservation_APortalReadWithoutARunIsNotKept(t *testing.T) {
	t.Parallel()

	result := portalReadResult()
	delete(result, "portalRunId")

	assert.Nil(t, artifactFromObservation(observation(agentextension.ToolReadPortal, result)))
}

func TestAnApprovedPortalActionIsAPortalTaskNamedForItsSite(t *testing.T) {
	t.Parallel()

	artifact := portalActionArtifact("decision:apr_01", &agent.ToolExecutionResult{
		Action: "started",
		Name:   "Acme dock portal",
		Record: &agent.RecordRef{EntityType: string(permission.KindPortalRun), ID: "aprn_02"},
	})

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindPortalTask, artifact.Kind)
	assert.Equal(t, "Portal action on Acme dock portal", artifact.Title)
	assert.Equal(t, map[string]any{
		"portalRunId": "aprn_02",
		"mode":        "Act",
		"site":        "Acme dock portal",
		"written":     "started",
	}, artifact.Payload)
}

func TestAWriteToAnotherRecordIsNotAPortalTask(t *testing.T) {
	t.Parallel()

	assert.Nil(t, portalActionArtifact("call_1", &agent.ToolExecutionResult{
		Action: "updated",
		Record: &agent.RecordRef{EntityType: "shipment", ID: "shp_01"},
	}))
}

func TestAPortalTaskMustBeNamedInTheReply(t *testing.T) {
	t.Parallel()

	shown := agentruntimeports.ShownArtifact{Kind: string(assistantartifact.KindPortalTask)}
	assert.True(t, shown.MustBeLinked())
}
