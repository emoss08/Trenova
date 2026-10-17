package assistantservice

import (
	"maps"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	analysisKeyRunID   = "analysisRunId"
	analysisKeyPurpose = "purpose"
	analysisKeyCode    = "code"
	analysisKeyOutput  = "output"
	analysisKeyNote    = "note"
	analysisTitle      = "Analysis"
)

// analysisArtifact keeps what an analysis showed beside the conversation: the
// figures, the charts by position (the images are read from the run when the
// card is open) and the code that made them. The note is for the model.
func analysisArtifact(
	callID string,
	result map[string]any,
	arguments map[string]any,
) *assistantartifact.Artifact {
	if typeutils.StringOfTrimmed(result[analysisKeyRunID]) == "" {
		return nil
	}

	payload := maps.Clone(result)
	delete(payload, analysisKeyNote)
	payload[analysisKeyCode] = stringutils.TruncateRunes(
		typeutils.StringOfTrimmed(arguments[analysisKeyCode]),
		agentextension.MaxAnalysisCodeRunes,
	)
	for payloadSize(payload) > assistantartifact.MaxPayloadBytes {
		output, _ := payload[analysisKeyOutput].(string)
		runes := []rune(output)
		if len(runes) == 0 {
			payload[analysisKeyCode] = ""
			break
		}
		payload[analysisKeyOutput] = string(runes[len(runes)/2:])
		payload["truncated"] = true
	}

	title := typeutils.StringOfTrimmed(result[analysisKeyPurpose])
	if title == "" {
		title = analysisTitle
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindAnalysis,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}
