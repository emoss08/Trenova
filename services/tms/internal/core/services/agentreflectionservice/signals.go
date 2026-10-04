package agentreflectionservice

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const LongTaskToolCalls = 6

var correctionOpenings = []string{
	"no,", "no ", "no.", "nope", "not ", "not that", "wrong", "that's wrong", "that is wrong",
	"that's not", "that is not", "thats not", "actually", "instead", "i said", "i meant",
	"i asked", "don't", "dont", "do not", "stop", "why did you", "you should have",
	"you shouldn't", "you should not", "please don't", "incorrect", "that's incorrect",
	"undo", "redo", "try again",
}

var correctionPhrases = []string{
	"not what i", "i told you", "should have been", "you forgot", "you missed",
	"wrong one", "the wrong", "that isn't right", "that is not right", "that's not right",
	"isn't what", "is not what", "i didn't ask", "i did not ask",
}

var standingPhrases = []string{
	"from now on", "going forward", "in the future", "next time", "every time", "each time",
	"always ", "never ", "by default", "remember that", "remember to", "keep in mind",
	"whenever ", "make sure you", "for future", "from here on", "in future",
}

var refusalVerdicts = []string{
	aitrace.OutcomeDenied,
	aitrace.OutcomeDuplicate,
	aitrace.OutcomeOverBudget,
}

type DecidedProposal struct {
	ToolName string
	Outcome  agent.TrustOutcome
	Reason   string
}

type Window struct {
	Messages        []conversation.Message
	Proposals       []DecidedProposal
	NegativeRatings int
	HasPerson       bool
	Continues       bool
}

func ReadSignals(window *Window) agent.ReflectionSignals {
	signals := make(agent.ReflectionSignals, 0, 8)
	add := func(kind agent.ReflectionSignalKind, count int, detail string) {
		if count > 0 {
			signals = append(
				signals,
				agent.ReflectionSignal{Kind: kind, Count: count, Detail: detail},
			)
		}
	}

	failed, recovered, succeeded := readTools(window.Messages)
	add(agent.ReflectionSignalToolRecovered, len(recovered), strings.Join(recovered, ", "))
	add(agent.ReflectionSignalToolFailed, len(failed), strings.Join(failed, ", "))

	if window.HasPerson {
		corrected, standing := readPerson(window.Messages, window.Continues)
		add(agent.ReflectionSignalPersonCorrected, corrected, "")
		add(agent.ReflectionSignalStandingRequest, standing, "")
	}

	modified, rejected := make(
		[]string,
		0,
		len(window.Proposals),
	), make(
		[]string,
		0,
		len(window.Proposals),
	)
	for _, proposal := range window.Proposals {
		switch proposal.Outcome {
		case agent.TrustOutcomeModified:
			modified = sliceutils.AppendIfMissing(modified, proposal.ToolName)
		case agent.TrustOutcomeRejected:
			rejected = sliceutils.AppendIfMissing(rejected, proposal.ToolName)
		}
	}
	add(agent.ReflectionSignalProposalModified, len(modified), strings.Join(modified, ", "))
	add(agent.ReflectionSignalProposalRejected, len(rejected), strings.Join(rejected, ", "))
	add(agent.ReflectionSignalNegativeFeedback, window.NegativeRatings, "")

	if succeeded >= LongTaskToolCalls {
		add(agent.ReflectionSignalLongTask, succeeded, "")
	}

	return signals
}

func readTools(messages []conversation.Message) ([]string, []string, int) {
	failing := make([]string, 0, 4)
	recovered := make([]string, 0, 4)
	succeeded := 0

	for idx := range messages {
		message := &messages[idx]
		if message.Role != conversation.RoleTool || message.ToolName == "" {
			continue
		}
		if slices.Contains(refusalVerdicts, message.ToolVerdict) {
			continue
		}
		if message.ToolFailed || message.ToolVerdict == aitrace.OutcomeInvalid ||
			message.ToolVerdict == aitrace.OutcomeFailed {
			failing = sliceutils.AppendIfMissing(failing, message.ToolName)
			continue
		}

		succeeded++
		if at := slices.Index(failing, message.ToolName); at >= 0 {
			failing = slices.Delete(failing, at, at+1)
			recovered = sliceutils.AppendIfMissing(recovered, message.ToolName)
		}
	}

	return failing, recovered, succeeded
}

func readPerson(messages []conversation.Message, continues bool) (int, int) {
	corrected, standing := 0, 0
	answered := continues

	for idx := range messages {
		message := &messages[idx]
		switch message.Role {
		case conversation.RoleAssistant:
			if strings.TrimSpace(message.Content) != "" || len(message.ToolCalls) > 0 {
				answered = true
			}
		case conversation.RoleUser:
			if message.Kind != conversation.MessageKindMessage || message.Refused {
				continue
			}
			text := strings.ToLower(strings.TrimSpace(message.Content))
			if text == "" {
				continue
			}
			if answered && corrects(text) {
				corrected++
			}
			if stringutils.ContainsAny(text, standingPhrases...) {
				standing++
			}
		}
	}

	return corrected, standing
}

func corrects(text string) bool {
	for _, opening := range correctionOpenings {
		if strings.HasPrefix(text, opening) {
			return true
		}
	}

	return stringutils.ContainsAny(text, correctionPhrases...)
}
