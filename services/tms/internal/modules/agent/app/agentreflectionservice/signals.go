package agentreflectionservice

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

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
	InPassing       bool
	// Corrections are records an agent wrote whose same field people then
	// changed by hand, and CorrectedWhat names that field in plain words.
	Corrections   []agentreflectionports.CorrectedWork
	CorrectedWhat string
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

	told := 0
	if window.HasPerson {
		read := readPerson(window.Messages, window.Continues)
		add(agentreflection.ReflectionSignalPersonCorrected, read.corrected, "")
		add(agentreflection.ReflectionSignalStandingRequest, read.standing, "")
		told = read.told
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
	add(
		agentreflection.ReflectionSignalChangedAfterwards,
		len(window.Corrections),
		window.CorrectedWhat,
	)
	if window.InPassing {
		add(agentreflection.ReflectionSignalPersonTold, told, "")
	}

	return signals
}

type personRead struct {
	corrected int
	standing  int
	told      int
}

func readPerson(messages []conversation.Message, continues bool) personRead {
	var read personRead
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
			words := lexiconOf(text)
			corrected := answered && words.corrects(text)
			standing := stringutils.ContainsAny(text, words.standing...)
			if corrected {
				read.corrected++
			}
			if standing {
				read.standing++
			}
			if !corrected && !standing && Tells(text) {
				read.told++
			}
		}
	}

	return read
}

// Tells reports whether a message states something rather than asking for
// it: some sentence long enough to carry a fact that is not a question and
// does not open as a request or a reaction to the last reply. It is a gate on
// spending a look back, not a judgment of what was said; the model reads the
// conversation and decides whether anything in it lasts.
func Tells(text string) bool {
	text = strings.ToLower(text)
	words := lexiconOf(text)
	for _, sentence := range sentences(text) {
		if words.states(sentence) {
			return true
		}
	}

	return false
}

func sentences(text string) []string {
	out := make([]string, 0, 4)
	start := 0
	for idx, r := range text {
		size := utf8.RuneLen(r)
		next := idx + size
		question := r == '?' || r == '？'
		ends := question || r == '!' || r == '！' || r == '。' || r == '；' || r == '\n' ||
			(r == '.' && (next == len(text) || text[next] == ' ' || text[next] == '\n'))
		if !ends {
			continue
		}
		end := idx
		if question {
			end = next
		}
		if sentence := strings.TrimSpace(text[start:end]); sentence != "" {
			out = append(out, sentence)
		}
		start = next
	}
	if sentence := strings.TrimSpace(text[start:]); sentence != "" {
		out = append(out, sentence)
	}

	return out
}
