package toolartifacts

import (
	"maps"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	analysisKeyRunID   = "analysisRunId"
	analysisKeyStatus  = "status"
	analysisKeyTitle   = "title"
	analysisKeyPurpose = "purpose"
	analysisKeyCode    = "code"
	analysisKeyOutput  = "output"
	analysisKeyNote    = "note"
	analysisKeyFigures = "figures"
	analysisKeyTables  = "tables"
	analysisKeyCharts  = "charts"
	analysisTitle      = "Analysis"
)

// modelOnlyKeys are what a script's answer tells the model and the card does
// not show: the note, its reads and their limits, and its one result.
var modelOnlyKeys = []string{
	analysisKeyNote, "calls", "refusals", "limits", "changes", "withheldByAccess", "result",
	"notShown",
}

// analysisArtifact keeps what a script showed beside the conversation: the
// headline figures, the tables, the charts as what they plot, and the code
// that made them. The note and what the script's reads cost are for the
// model. Only a script that ran to the end and showed something is kept: one
// whose code raised is the agent's to fix and run again, and one that only
// printed is the agent looking at the data. A card for that look was one more
// the reply had to name, and a model that loaded a workbook to read its
// columns before analyzing it was sent back to link it.
func analysisArtifact(
	callID string,
	result map[string]any,
	arguments map[string]any,
) *assistantartifact.Artifact {
	if typeutils.StringOfTrimmed(result[analysisKeyRunID]) == "" ||
		typeutils.StringOfTrimmed(result[analysisKeyStatus]) !=
			analysis.StatusSucceeded.String() || !showedAnything(result) {
		return nil
	}

	payload := maps.Clone(result)
	for _, key := range modelOnlyKeys {
		delete(payload, key)
	}
	code := typeutils.StringOfTrimmed(result[analysisKeyCode])
	if code == "" {
		code = typeutils.StringOfTrimmed(arguments[analysisKeyCode])
	}
	payload[analysisKeyCode] = stringutils.TruncateRunes(code, analysis.MaxCodeRunes)
	for artifactpayload.PayloadSize(payload) > assistantartifact.MaxPayloadBytes {
		output, _ := payload[analysisKeyOutput].(string)
		runes := []rune(output)
		if len(runes) == 0 {
			payload[analysisKeyCode] = ""
			break
		}
		payload[analysisKeyOutput] = string(runes[len(runes)/2:])
		payload["truncated"] = true
	}

	title := typeutils.StringOfTrimmed(result[analysisKeyTitle])
	if title == "" {
		title = typeutils.StringOfTrimmed(result[analysisKeyPurpose])
	}
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

// showedAnything reports whether an analysis displayed a figure, a table or a
// chart for the person.
func showedAnything(result map[string]any) bool {
	for _, key := range []string{analysisKeyFigures, analysisKeyTables, analysisKeyCharts} {
		if shown, ok := result[key].([]any); ok && len(shown) > 0 {
			return true
		}
	}

	return false
}
