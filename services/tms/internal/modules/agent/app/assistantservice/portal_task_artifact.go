package assistantservice

import (
	"maps"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	portalKeyRunID    = "portalRunId"
	portalKeyMode     = "mode"
	portalKeySite     = "site"
	portalKeyQuestion = "question"
	portalKeyAnswer   = "answer"
	portalKeyPages    = "pagesRead"
	portalKeyNote     = "note"
	portalModeRead    = "Read"
	portalModeAct     = "Act"
	portalTitle       = "Portal task"
	portalActedTitle  = "Portal action on "
)

// portalTaskArtifact keeps a portal read beside the conversation: the site,
// the question, and what the portal said or how it asked for a sign-in code.
// The card follows the run itself while it is open, so a read that waits on
// a code shows the answer once the code is in. The note is for the model.
func portalTaskArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	if typeutils.StringOfTrimmed(result[portalKeyRunID]) == "" {
		return nil
	}

	payload := maps.Clone(result)
	delete(payload, portalKeyNote)
	payload[portalKeyMode] = portalModeRead
	if payloadSize(payload) > assistantartifact.MaxPayloadBytes {
		delete(payload, portalKeyPages)
		payload[portalKeyAnswer] = ""
		payload["truncated"] = true
	}

	title := typeutils.StringOfTrimmed(result[portalKeyQuestion])
	if title == "" {
		title = typeutils.StringOfTrimmed(result[portalKeySite])
	}
	if title == "" {
		title = portalTitle
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindPortalTask,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// portalActionArtifact is an approved portal action the browser was handed:
// the card follows the run to what the portal said.
func portalActionArtifact(
	callID string,
	executed *agentrecord.ToolExecutionResult,
) *assistantartifact.Artifact {
	if executed.Record == nil || executed.Record.EntityType != string(permission.KindPortalRun) {
		return nil
	}

	title := portalTitle
	if site := typeutils.StringOfTrimmed(executed.Name); site != "" {
		title = portalActedTitle + site
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindPortalTask,
		Status: assistantartifact.StatusReady,
		Title:  assistantartifact.NormalizeTitle(title),
		Payload: map[string]any{
			portalKeyRunID: executed.Record.ID,
			portalKeyMode:  portalModeAct,
			portalKeySite:  executed.Name,
			payloadWritten: executed.Action,
		},
		SourceToolCallID: callID,
	}
}
