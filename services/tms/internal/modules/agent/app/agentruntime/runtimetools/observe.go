package runtimetools

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// Observe hands a finished call to the observer and folds what it showed the
// person back into the result the model reads.
func Observe(
	observe agentruntimeports.ToolObserver,
	call llmports.ToolCall,
	outcome toolcall.Result,
	earlier []string,
) toolcall.Result {
	if request, requested := outcome.Data.(agentruntimeports.DecisionRequest); requested &&
		outcome.Publishes {
		return DecisionRequested(observe, &call, request)
	}
	if observe == nil {
		if _, boards := outcome.Data.(agentruntimeports.BoardRequest); boards {
			return toolcall.Failed("%s", UnboardableRefusal)
		}
		if outcome.Publishes {
			return toolcall.Failed("%s", UnpublishableRefusal)
		}
		return outcome
	}

	shown, err := observe(agentruntimeports.ToolObservation{
		Call:    call,
		RanWith: outcome.RanWith,
		Data:    outcome.Data,
		Failed:  outcome.Failed,
		Action:  outcome.Action,
		Earlier: earlier,
	})

	if shown != nil {
		outcome.Shown = shown
	}

	_, boards := outcome.Data.(agentruntimeports.BoardRequest)
	mistake, fixable := errors.AsType[*agentruntimeports.PublishMistake](err)
	switch {
	case boards && fixable:
		return toolcall.Failed(
			"Tool %q did not show the board: %s. Fix the call and send it "+
				"again.",
			ShowBoardName,
			mistake.Reason,
		)
	case boards && (err != nil || shown == nil):
		return toolcall.Failed("Tool %q could not show the board just now. Answer in your "+
			"reply instead.", ShowBoardName)
	case boards:
		outcome.Content = BoardContent(shown)
		outcome.Summary = toolsummary.Line(shown.Title)
	case outcome.Publishes && fixable:
		return toolcall.Failed(
			"Tool %q did not keep the document: %s. Fix the call and send "+
				"it again.",
			PublishArtifactName,
			mistake.Reason,
		)
	case outcome.Publishes && err != nil:
		return toolcall.Failed(
			"Tool %q could not keep the document: %s. Put the text in your "+
				"reply instead.",
			PublishArtifactName,
			err.Error(),
		)
	case outcome.Publishes && shown == nil:
		return toolcall.Failed("Tool %q could not keep the document. Put the text in your "+
			"reply instead.", PublishArtifactName)
	case outcome.Publishes:
		outcome.Content = PublishedContent(shown)
		outcome.Summary = toolsummary.Line(shown.Title)
	case shown != nil && !outcome.Failed:
		outcome.Content += ShownNote(shown)
		if outcome.Summary == "" {
			outcome.Summary = toolsummary.Line(shown.Title)
		}
	}

	return outcome
}
