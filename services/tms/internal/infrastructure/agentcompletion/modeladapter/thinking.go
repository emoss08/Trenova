package modeladapter

import "strings"

// thinkTags are the wrappers a model puts round its chain of thought when
// the protocol gives it nowhere else to put it. Chat Completions has one
// content field, so a model trained to think first emits the thinking
// inline and marks it — DeepSeek-R1 and everything distilled from it use
// <think>, Qwen's reasoning builds use the same, and several serving
// stacks re-emit it verbatim rather than lifting it into reasoning_content.
var thinkTags = [][2]string{
	{"<think>", "</think>"},
	{"<thinking>", "</thinking>"},
	{"<reasoning>", "</reasoning>"},
}

// SplitInlineThinking separates a chain of thought a model wrote into its
// own reply.
//
// Without this the thinking is the reply. It is shown to the person as the
// assistant's answer, stored as the answer, and replayed to the model next
// turn as though the assistant had said it — so a model that has talked
// itself in circles is then asked to continue from its own circling. That
// is how one bad turn becomes a conversation of them.
//
// Only a marked block is moved. Text with no opening tag is returned
// untouched, because guessing at where unmarked thinking ends would throw
// away answers.
func SplitInlineThinking(text string) (reply, thinking string) {
	for _, tag := range thinkTags {
		open, close := tag[0], tag[1]
		start := indexFold(text, open)
		if start < 0 {
			continue
		}

		rest := text[start+len(open):]
		end := indexFold(rest, close)
		if end < 0 {
			// An unterminated block means the model was cut off mid-thought
			// and never reached an answer. Everything after the tag is
			// thinking; what came before it, if anything, is the reply.
			return strings.TrimSpace(text[:start]), strings.TrimSpace(rest)
		}

		thinking = strings.TrimSpace(rest[:end])
		reply = strings.TrimSpace(text[:start] + rest[end+len(close):])

		return reply, thinking
	}

	return text, ""
}

// mergeInlineThinking folds a reply's inline chain of thought into the
// trace the protocol carried, if any. A server that both marks the
// thinking inline and repeats it in reasoning_content — which is what
// produced a transcript whose visible answer was its own thinking, word
// for word — ends up with one trace and a reply that is only the answer.
func mergeInlineThinking(text string, trace *ReasoningTrace) (string, *ReasoningTrace) {
	reply, thinking := SplitInlineThinking(text)
	if thinking == "" {
		return dedupeTrace(text, trace)
	}

	if trace == nil {
		return reply, textReasoning(thinking)
	}
	if strings.TrimSpace(trace.Text) == "" || strings.Contains(trace.Text, thinking) {
		return reply, trace
	}

	merged := *trace
	merged.Text = strings.TrimSpace(trace.Text + "\n\n" + thinking)

	return reply, &merged
}

// dedupeTrace drops a reply that is only the trace repeated.
//
// Some servers put the whole chain of thought in both fields. Believing
// the content in that case shows a person the model's deliberation where
// the answer should be, and — worse — stores it as the answer. An empty
// reply is the honest outcome: the caller already treats "no content" as
// a failure worth reporting or failing over, which is what happened.
func dedupeTrace(text string, trace *ReasoningTrace) (string, *ReasoningTrace) {
	if trace == nil || strings.TrimSpace(text) == "" {
		return text, trace
	}
	if strings.TrimSpace(text) == strings.TrimSpace(trace.Text) {
		return "", trace
	}

	return text, trace
}

// indexFold finds a tag however the model capitalised it.
func indexFold(haystack, needle string) int {
	return strings.Index(strings.ToLower(haystack), needle)
}
