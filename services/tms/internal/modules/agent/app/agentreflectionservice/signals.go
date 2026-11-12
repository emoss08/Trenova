package agentreflectionservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

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

var askingOpenings = stringSet(
	"show", "list", "find", "get", "give", "create", "make", "add", "update", "change", "set",
	"send", "draft", "write", "open", "check", "look", "pull", "run", "book", "assign",
	"cancel", "delete", "remove", "tell", "help", "can", "could", "would", "will", "please",
	"what", "what's", "whats", "when", "where", "who", "who's", "whos", "why", "how", "which",
	"is", "are", "do", "does", "did", "should", "need", "want", "let's", "lets", "schedule",
	"dispatch", "move", "mark", "close", "search", "summarize", "explain", "compare",
	"calculate", "build", "generate", "remind", "put", "email", "call", "try", "use",
)

var reactingOpenings = stringSet(
	"ok", "okay", "thanks", "thank", "great", "perfect", "yes", "yep", "yeah", "no", "nope",
	"sure", "cool", "good", "nice", "got", "sounds", "looks", "that", "that's", "thats",
	"this", "it", "it's", "its", "hi", "hello", "hey", "sorry", "nevermind", "never",
)

var askingPhrases = []string{
	"i need",
	"i want",
	"i'd like",
	"i would like",
	"i am looking",
	"i'm looking",
}

const minToldWords = 5

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
		read := readPerson(window.Messages, window.Continues)
		add(agentreflection.ReflectionSignalPersonCorrected, read.corrected, "")
		add(agentreflection.ReflectionSignalStandingRequest, read.standing, "")
		if window.InPassing {
			add(agentreflection.ReflectionSignalPersonTold, read.told, "")
		}
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
			corrected := answered && corrects(text)
			standing := stringutils.ContainsAny(text, standingPhrases...)
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

func Tells(text string) bool {
	for _, sentence := range sentences(strings.ToLower(text)) {
		if states(sentence) {
			return true
		}
	}

	return false
}

func sentences(text string) []string {
	out := make([]string, 0, 4)
	start := 0
	for idx, r := range text {
		ends := r == '?' || r == '!' || r == '\n' ||
			(r == '.' && (idx+1 == len(text) || text[idx+1] == ' ' || text[idx+1] == '\n'))
		if !ends {
			continue
		}
		end := idx
		if r == '?' {
			end = idx + 1
		}
		if sentence := strings.TrimSpace(text[start:end]); sentence != "" {
			out = append(out, sentence)
		}
		start = idx + 1
	}
	if sentence := strings.TrimSpace(text[start:]); sentence != "" {
		out = append(out, sentence)
	}

	return out
}

func states(sentence string) bool {
	if strings.HasSuffix(sentence, "?") {
		return false
	}
	words := strings.Fields(sentence)
	if len(words) < minToldWords {
		return false
	}
	first := strings.Trim(words[0], ",:;\"'()-")
	if _, asking := askingOpenings[first]; asking {
		return false
	}
	if _, reacting := reactingOpenings[first]; reacting {
		return false
	}
	for _, phrase := range askingPhrases {
		if strings.HasPrefix(sentence, phrase) {
			return false
		}
	}

	return true
}

func stringSet(words ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(words))
	for _, word := range words {
		set[word] = struct{}{}
	}

	return set
}

func corrects(text string) bool {
	for _, opening := range correctionOpenings {
		if strings.HasPrefix(text, opening) {
			return true
		}
	}

	return stringutils.ContainsAny(text, correctionPhrases...)
}
