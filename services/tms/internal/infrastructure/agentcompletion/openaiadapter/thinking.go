package openaiadapter

import (
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

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
		opening, closing := tag[0], tag[1]
		start := indexFold(text, opening)
		if start < 0 {
			continue
		}

		rest := text[start+len(opening):]
		end := indexFold(rest, closing)
		if end < 0 {
			// An unterminated block means the model was cut off mid-thought
			// and never reached an answer. Everything after the tag is
			// thinking; what came before it, if anything, is the reply.
			return strings.TrimSpace(text[:start]), strings.TrimSpace(rest)
		}

		thinking = strings.TrimSpace(rest[:end])
		reply = strings.TrimSpace(text[:start] + rest[end+len(closing):])

		return reply, thinking
	}

	return text, ""
}

// mergeInlineThinking folds a reply's inline chain of thought into the
// trace the protocol carried, if any. A server that both marks the
// thinking inline and repeats it in reasoning_content — which is what
// produced a transcript whose visible answer was its own thinking, word
// for word — ends up with one trace and a reply that is only the answer.
func mergeInlineThinking(
	text string,
	trace *modeladapter.ReasoningTrace,
) (answer string, kept *modeladapter.ReasoningTrace) {
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
func dedupeTrace(
	text string,
	trace *modeladapter.ReasoningTrace,
) (answer string, kept *modeladapter.ReasoningTrace) {
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

// inlineThinkingStream routes a streamed reply's marked thinking to the
// thinking sink as it arrives, so a person sees only that the model is
// thinking and then its answer, never the thought written out as the reply.
// A tag split across deltas is held back until it is whole or ruled out.
type inlineThinkingStream struct {
	reply   modeladapter.StreamSink
	think   func(string)
	pending string
	closing string
}

func newInlineThinkingStream(
	reply modeladapter.StreamSink,
	think func(string),
) *inlineThinkingStream {
	return &inlineThinkingStream{reply: reply, think: think}
}

func (s *inlineThinkingStream) Write(delta string) {
	buf := s.pending + delta
	s.pending = ""

	for buf != "" {
		if s.closing == "" {
			at, tag := firstOpeningTag(buf)
			if at >= 0 {
				s.emitReply(buf[:at])
				s.closing = tag[1]
				buf = buf[at+len(tag[0]):]
				continue
			}
			keep := partialTagSuffix(buf, openingTags())
			s.emitReply(buf[:len(buf)-keep])
			s.pending = buf[len(buf)-keep:]
			return
		}

		if at := indexTagFold(buf, s.closing); at >= 0 {
			s.think(buf[:at])
			buf = buf[at+len(s.closing):]
			s.closing = ""
			continue
		}
		keep := partialTagSuffix(buf, []string{s.closing})
		s.think(buf[:len(buf)-keep])
		s.pending = buf[len(buf)-keep:]
		return
	}
}

func (s *inlineThinkingStream) Flush() {
	held := s.pending
	s.pending = ""
	if s.closing != "" {
		s.think(held)
		return
	}
	s.emitReply(held)
}

func (s *inlineThinkingStream) emitReply(text string) {
	if text != "" {
		s.reply(text)
	}
}

func openingTags() []string {
	tags := make([]string, 0, len(thinkTags))
	for _, tag := range thinkTags {
		tags = append(tags, tag[0])
	}
	return tags
}

func firstOpeningTag(text string) (int, [2]string) {
	first, found := -1, [2]string{}
	for _, tag := range thinkTags {
		if at := indexTagFold(text, tag[0]); at >= 0 && (first < 0 || at < first) {
			first, found = at, tag
		}
	}
	return first, found
}

func indexTagFold(text, tag string) int {
	for i := 0; i+len(tag) <= len(text); i++ {
		if strings.EqualFold(text[i:i+len(tag)], tag) {
			return i
		}
	}
	return -1
}

func partialTagSuffix(text string, tags []string) int {
	longest := 0
	for _, tag := range tags {
		for size := min(len(tag)-1, len(text)); size > longest; size-- {
			if strings.EqualFold(text[len(text)-size:], tag[:size]) {
				longest = size
				break
			}
		}
	}
	return longest
}
