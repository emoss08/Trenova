package agentreflectionservice

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
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

type DecidedProposal struct {
	ToolName string
	Outcome  agenttooltrust.TrustOutcome
	Reason   string
}

type Window struct {
	Messages        []conversation.Message
	Proposals       []DecidedProposal
	NegativeRatings int
	HasPerson       bool
	Continues       bool
}

func ReadSignals(window *Window) agentreflection.ReflectionSignals {
	signals := make(agentreflection.ReflectionSignals, 0, 8)
	add := func(kind agentreflection.ReflectionSignalKind, count int, detail string) {
		if count > 0 {
			signals = append(
				signals,
				agentreflection.ReflectionSignal{Kind: kind, Count: count, Detail: detail},
			)
		}
	}

	if window.HasPerson {
		corrected, standing := readPerson(window.Messages, window.Continues)
		add(agentreflection.ReflectionSignalPersonCorrected, corrected, "")
		add(agentreflection.ReflectionSignalStandingRequest, standing, "")
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
		case agenttooltrust.TrustOutcomeModified:
			modified = sliceutils.AppendIfMissing(modified, proposal.ToolName)
		case agenttooltrust.TrustOutcomeRejected:
			rejected = sliceutils.AppendIfMissing(rejected, proposal.ToolName)
		}
	}
	add(
		agentreflection.ReflectionSignalProposalModified,
		len(modified),
		strings.Join(modified, ", "),
	)
	add(
		agentreflection.ReflectionSignalProposalRejected,
		len(rejected),
		strings.Join(rejected, ", "),
	)
	add(agentreflection.ReflectionSignalNegativeFeedback, window.NegativeRatings, "")

	if succeeded := succeededTools(window.Messages); window.HasPerson &&
		succeeded >= LongTaskToolCalls {
		add(agentreflection.ReflectionSignalLongTask, succeeded, "")
	}

	return signals
}

var unsuccessfulVerdicts = []string{
	aitrace.OutcomeDenied,
	aitrace.OutcomeDuplicate,
	aitrace.OutcomeOverBudget,
	aitrace.OutcomeInvalid,
	aitrace.OutcomeFailed,
}

func succeededTools(messages []conversation.Message) int {
	succeeded := 0
	for idx := range messages {
		message := &messages[idx]
		if message.Role != conversation.RoleTool || message.ToolName == "" || message.ToolFailed {
			continue
		}
		if !slices.Contains(unsuccessfulVerdicts, message.ToolVerdict) {
			succeeded++
		}
	}

	return succeeded
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
