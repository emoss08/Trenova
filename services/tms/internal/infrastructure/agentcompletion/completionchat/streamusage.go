package completionchat

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/llmtokens"
)

// streamedOutcome is what an attempt that failed or was stopped after the
// provider began writing consumed. The provider's own count arrives with the
// end of the stream, so a stream that never got there reported nothing; the
// reply and the thinking it did write were billed all the same, and are
// counted from what arrived rather than recorded as free. Nil when nothing
// arrived.
func streamedOutcome(
	provider *aiprovider.Provider,
	streamed chatStream,
	cause error,
) *completionengine.RunOutcome {
	if streamed.text == "" && streamed.reasoningRunes == 0 {
		return nil
	}

	reasoning := llmtokens.FromRunes(streamed.reasoningRunes)

	return &completionengine.RunOutcome{
		Model:           modeladapter.ServedModel(cause, provider.Model),
		OutputTokens:    llmtokens.Estimate(streamed.text) + reasoning,
		ReasoningTokens: reasoning,
	}
}
