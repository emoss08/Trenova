package agentruntime

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

func (s *Service) placeArtifacts(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
	state *replycheck.PlacementState,
) bool {
	if completion == nil || t.req.Delegation != nil || strings.TrimSpace(completion.Text) == "" {
		return false
	}

	problems := replycheck.FindPlacementProblems(t, completion.Text)
	if problems.Empty() {
		return false
	}

	// A reply that only left out a table it searched (Shipments, Invoices,
	// Time offs) was asked again in about one turn in ten, and the person
	// watched a finished answer restart to add a link. A reply that ended with
	// its card on a line of its own was asked again the same way (DB-148).
	// Both are repaired without asking the model again: a stray link is cut and
	// joined to the reply's own sentence nearest above where it stood, a
	// left-out one to the reply's first sentence, and the server writes no
	// words of its own, since the turn's language is the model's (DB-237).
	if !state.Retried && problems.Mendable() {
		if placed, ok := replycheck.PlaceLinks(t, completion.Text, &problems); ok {
			state.Retried = true
			reason := replycheck.PlacementNamedReason
			if len(problems.Stray) > 0 {
				reason = replycheck.PlacementMovedReason
			}
			fx.Emit(replycheck.PassEvent(assistantstreamports.RegroundPlaceArtifacts, reason, nil))
			fx.Emit(replycheck.ReplyReplacedEvent(placed, reason))
			completion.Text = placed

			return false
		}
	}

	if !state.Retried {
		state.Retried = true
		t.sendBack(fx, replycheck.SendBackDraft(
			completion,
			replycheck.PlacementRetryReason,
			replycheck.PassEvent(
				assistantstreamports.RegroundRelinkArtifacts,
				replycheck.PlacementRetryReason,
				nil,
			),
			replycheck.PlacementCorrection(t, &problems),
		))

		return true
	}

	placed, _ := replycheck.PlaceLinks(t, completion.Text, &problems)
	fx.Emit(
		replycheck.PassEvent(
			assistantstreamports.RegroundPlaceArtifacts,
			replycheck.PlacementPlacedReason,
			nil,
		),
	)
	fx.Emit(replycheck.ReplyReplacedEvent(placed, replycheck.PlacementPlacedReason))
	completion.Text = placed

	return false
}
