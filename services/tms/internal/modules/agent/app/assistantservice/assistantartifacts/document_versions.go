package assistantartifacts

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	maxRewriteRunes  = 4000
	maxPromptRunes   = 300
	maxVersionNote   = 120
	rewriteMaxTokens = 1200
)

func latestOf(versions []*assistantartifact.Artifact) *assistantartifact.Artifact {
	latest := versions[0]
	for _, version := range versions[1:] {
		if version.LineageSeq > latest.LineageSeq {
			latest = version
		}
	}

	return latest
}

const rewriteSystem = "You edit one passage of a document a logistics team wrote. " +
	"Rewrite only the passage you are given, in the same language and voice, keeping every " +
	"fact, figure, name, date and citation mark like [^2] exactly as written. Do not add " +
	"facts. Keep markdown emphasis where it still fits. Answer with the rewritten passage only."

func rewriteInstruction(rewrite assistantartifactports.DocumentRewriteRequest) (string, error) {
	text := strings.TrimSpace(rewrite.Text)
	if text == "" || len([]rune(text)) > maxRewriteRunes {
		return "", errortypes.NewValidationError(
			"text", errortypes.ErrInvalid, "Select a passage of the document to rewrite",
		)
	}
	switch rewrite.Mode {
	case assistantartifactports.DocumentRewriteShorter:
		return "Make it shorter: about half the words, keeping what matters.", nil
	case assistantartifactports.DocumentRewritePlainer:
		return "Make it plainer: short sentences and everyday words, no jargon.", nil
	case assistantartifactports.DocumentRewriteAsk:
		prompt := strings.TrimSpace(rewrite.Prompt)
		if prompt == "" || len([]rune(prompt)) > maxPromptRunes {
			return "", errortypes.NewValidationError(
				"prompt", errortypes.ErrInvalid, "Say how to change it in a sentence",
			)
		}

		return prompt, nil
	default:
		return "", errortypes.NewValidationError(
			"mode", errortypes.ErrInvalid, "Choose shorter, plainer or say how to change it",
		)
	}
}
