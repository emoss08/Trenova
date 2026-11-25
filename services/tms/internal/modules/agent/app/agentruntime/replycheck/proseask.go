package replycheck

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	GuardQuestion = "question"

	askUserTool = "ask_user"

	questionCorrectionLead = "Your reply ends by asking the person something in prose"
	QuestionRetryReason    = "The reply asked the person a question in prose, so it is " +
		"being asked as a question they can answer with a click."

	// maxQuotedQuestionRunes keeps the question the correction quotes back to
	// one line: enough for the model to find it, not a reprint of the reply.
	maxQuotedQuestionRunes = 200
	// closingSentences is how far back from its end a reply's question can
	// stand and still be what the reply closes on: the last sentence, or the
	// one before a closing line such as "Nothing goes out until you approve."
	closingSentences = 2
)

var (
	codeFence = regexp.MustCompile("(?s)```.*?```")
	// promptLead is a closing line that asks for direction without a question
	// mark: "Let me know which batches to approve."
	promptLead = regexp.MustCompile(`(?i)\b(?:let me know|tell me|confirm)\s+` +
		`(?:which|whether|if|how|what|when|who)\b`)
	// answeredAfter is a sentence that answers the question before it, which
	// makes that question the reply's own, not one put to the person.
	answeredAfter = regexp.MustCompile(`(?i)^\W*(?:yes|no|not)\b`)
	sentenceStop  = regexp.MustCompile(`[.!?](?:\s|$)`)
)

// AskForQuestion sends back a reply that closes on a question to the person
// while ask_user was on offer and went uncalled.
//
// A model that has a question writes it into its reply far more often than it
// calls ask_user: "Want me to go ahead?", "Should I start with batch 1?",
// "Would you like me to move on to batch 2?". The person then has to read the
// whole reply to find what they were asked, and answer by typing. One person
// asked six times in one conversation to be given buttons, and was given prose
// each time but the turns where they had just complained. The rule in the
// instructions was not enough, so the reply is checked: a question at its end
// goes back to the model once, to be asked with ask_user, where it is pinned
// above the composer with its choices as buttons.
//
// Only a turn the person is watching is held to it, because only there is
// someone to answer, and only once a turn: a model that writes the question
// again has a reason this check cannot see.
func AskForQuestion(
	t Turn,
	draft *llmports.ChatCompletionResult,
	state *BoardAskState,
) (SendBack, bool) {
	if state.Questioned || draft == nil || !t.Offers(askUserTool) || t.Request() == nil ||
		!t.Request().AskedByPerson() {
		return SendBack{}, false
	}
	question := ClosingQuestion(draft.Text)
	if question == "" {
		return SendBack{}, false
	}
	state.Questioned = true

	back := SendBackDraft(
		draft,
		QuestionRetryReason,
		PassEvent(assistantstreamports.RegroundAskUser, QuestionRetryReason, nil),
		questionCorrection(question),
	)
	back.Guard = GuardQuestion

	return back, true
}

// ClosingQuestion is the question a reply closes on, or empty when its last
// paragraph asks the person nothing. A question inside a code block is
// quoted, not asked, and is not counted.
func ClosingQuestion(reply string) string {
	text := strings.TrimSpace(codeFence.ReplaceAllString(reply, ""))
	if text == "" {
		return ""
	}
	last := text
	if at := strings.LastIndex(text, "\n\n"); at >= 0 {
		last = strings.TrimSpace(text[at+2:])
	}
	last = strings.Join(strings.Fields(last), " ")

	sentences := sentencesOf(last)
	for idx, sentence := range sentences[:min(len(sentences), closingSentences)] {
		if idx > 0 && answeredAfter.MatchString(sentences[idx-1]) {
			return ""
		}
		trimmed := strings.TrimRight(sentence, " *_)\"'”’")
		if strings.HasSuffix(trimmed, "?") || promptLead.MatchString(sentence) {
			return quotedQuestion(sentence)
		}
	}

	return ""
}

// sentencesOf splits a paragraph into its sentences, last first, because the
// question that matters is the one the reply ends on.
func sentencesOf(paragraph string) []string {
	var sentences []string
	start := 0
	for _, at := range sentenceStop.FindAllStringIndex(paragraph, -1) {
		sentence := strings.TrimSpace(paragraph[start:at[1]])
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
		start = at[1]
	}
	if rest := strings.TrimSpace(paragraph[start:]); rest != "" {
		sentences = append(sentences, rest)
	}
	for left, right := 0, len(sentences)-1; left < right; left, right = left+1, right-1 {
		sentences[left], sentences[right] = sentences[right], sentences[left]
	}

	return sentences
}

func quotedQuestion(sentence string) string {
	sentence = strings.Trim(sentence, " *_")
	if utf8.RuneCountInString(sentence) <= maxQuotedQuestionRunes {
		return sentence
	}

	return strings.TrimSpace(string([]rune(sentence)[:maxQuotedQuestionRunes])) + "…"
}

func questionCorrection(question string) string {
	return questionCorrectionLead + ` ("` + question + `"). Every question to the person ` +
		"is asked with ask_user, never written in your reply: it is pinned above their " +
		"message box with its choices as buttons, so they answer with one click instead of " +
		"reading your reply for what you asked and typing. Write your reply again without " +
		"that question, keeping what the person needs to know before they answer, then " +
		"call ask_user with the question and its choices as options (best first, each with " +
		"a short detail of what happens if it is picked). When you need several things " +
		"settled, ask them together in one ask_user call with questions, not one after " +
		"another. If you were not asking the person anything, reply again as you did." +
		UnseenCorrection
}
