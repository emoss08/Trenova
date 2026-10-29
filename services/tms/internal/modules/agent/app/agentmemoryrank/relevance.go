package agentmemoryrank

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	minRelevantTermLen = 3
	// minSharedWords is how many of the turn's words a memory must share
	// when none of them is rare among the organization's memories.
	minSharedWords = 2
	// minFocusWords is how many words the turn's own message must carry
	// before it is read alone; a shorter one ("yes, send it") is read with
	// the message before it.
	minFocusWords = 2
	// A word is rare enough to bear on its own when it appears in at most
	// this share of the candidates, and never in more than one of a few.
	rareWordShare = 0.1
	rareWordDocs  = 1
)

// conversationalWords carry a reply rather than a subject. They are left out
// of what a turn is about so "yes, go ahead and send that now" does not bear on
// every memory that says "now".
var conversationalWords = map[string]struct{}{
	"yes": {}, "yeah": {}, "yep": {}, "nope": {}, "okay": {}, "sure": {},
	"thanks": {}, "thank": {}, "you": {}, "your": {}, "yours": {}, "now": {},
	"just": {}, "also": {}, "still": {}, "will": {}, "should": {}, "shall": {},
	"let": {}, "lets": {}, "know": {}, "not": {}, "out": {}, "into": {},
	"make": {}, "take": {}, "going": {}, "got": {}, "like": {}, "more": {},
	"most": {}, "much": {}, "many": {}, "than": {}, "then": {}, "here": {},
	"been": {}, "being": {}, "every": {}, "each": {}, "they": {}, "him": {},
	"her": {}, "his": {}, "she": {}, "only": {}, "very": {}, "really": {},
	"use": {}, "using": {}, "help": {}, "dont": {}, "don": {},
	"doesn": {}, "isn": {}, "right": {}, "ahead": {}, "again": {}, "else": {},
	"thing": {}, "things": {}, "something": {}, "anything": {}, "way": {},
	"same": {}, "other": {}, "another": {}, "back": {}, "over": {}, "had": {},
}

// queryWord is one word the turn used, with the forms a memory may spell it
// in: the word, its singular and the product's words for it.
type queryWord struct {
	raw   string
	forms map[string]struct{}
}

// Relevance marks the candidates that bear on the turn: those near it in
// meaning, and those sharing its words. A shared word bears on its own only
// when it is rare among the candidates, so in an organization whose memories
// all say "shipment", asking about a shipment does not make every one of them
// bear; two shared words always do. The turn's own message is read for its
// words, and the one before it only when the message is too short to say
// what it is about. With neither a vector nor a word to go on, nothing is
// judged and every candidate bears.
func Relevance(
	focus, text string,
	ranked agentmemoryports.RankedMemories,
) agentmemory.MemoryRelevance {
	words := turnWords(focus, text)
	if !ranked.Semantic && len(words) == 0 {
		return agentmemory.MemoryRelevance{}
	}

	ids := make(map[pulid.ID]struct{}, len(ranked.Similar)+len(ranked.Memories)/4)
	for _, id := range ranked.Similar {
		ids[id] = struct{}{}
	}
	if len(words) == 0 {
		return agentmemory.MemoryRelevance{Judged: true, IDs: ids}
	}

	tokens := make([]map[string]struct{}, len(ranked.Memories))
	for idx, memory := range ranked.Memories {
		if memory != nil {
			tokens[idx] = memoryTokens(memory)
		}
	}

	rare := rareWords(words, tokens)
	need := min(minSharedWords, len(words))
	for idx, memory := range ranked.Memories {
		if memory == nil {
			continue
		}
		if _, ok := ids[memory.ID]; ok {
			continue
		}
		if sharesEnough(words, rare, tokens[idx], need) {
			ids[memory.ID] = struct{}{}
		}
	}

	return agentmemory.MemoryRelevance{Judged: true, IDs: ids}
}

func turnWords(focus, text string) []queryWord {
	words := queryWords(focus)
	if len(words) >= minFocusWords || strings.TrimSpace(text) == "" {
		return words
	}

	return queryWords(focus + "\n" + text)
}

func queryWords(text string) []queryWord {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	raw := agentsearch.Tokens(text)
	words := make([]queryWord, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, token := range raw {
		if len(token) < minRelevantTermLen {
			continue
		}
		if _, chatter := conversationalWords[token]; chatter {
			continue
		}
		singular := agentsearch.Singularize(token)
		if _, dup := seen[singular]; dup {
			continue
		}
		seen[singular] = struct{}{}
		words = append(words, queryWord{raw: token, forms: agentsearch.Terms(token)})
	}

	return words
}

// RecallTerms are the words of a recall query a memory can be found by when
// it does not match every word: the meaningful ones, as typed, because the
// search matches them as prefixes of what a memory spells ("policies" finds
// "policies"; "policy" would not).
func RecallTerms(query string) []string {
	words := queryWords(query)
	terms := make([]string, 0, len(words))
	for _, word := range words {
		terms = append(terms, word.raw)
	}

	return terms
}

func memoryTokens(memory *agentmemory.Memory) map[string]struct{} {
	tokens := agentsearch.TokenSet(memory.Content)
	for _, text := range [...]string{
		memory.SubjectLabel,
		strings.ReplaceAll(memory.ToolName, "_", " "),
	} {
		if text == "" {
			continue
		}
		for token := range agentsearch.TokenSet(text) {
			tokens[token] = struct{}{}
		}
	}

	return tokens
}

// rareWords marks the turn's words that few candidates share, the ones that
// say which memory the turn is about rather than what kind of work it is. A
// word most of a small set shares is not rare in it, however small the set;
// with one candidate there is nothing to compare, and a shared word stands.
func rareWords(words []queryWord, tokens []map[string]struct{}) []bool {
	limit := max(rareWordDocs, int(float64(len(tokens))*rareWordShare))
	rare := make([]bool, len(words))
	for idx, word := range words {
		docs := 0
		for _, set := range tokens {
			if word.in(set) {
				docs++
			}
		}
		rare[idx] = docs > 0 && docs <= limit && (docs < len(tokens) || len(tokens) == 1)
	}

	return rare
}

func sharesEnough(words []queryWord, rare []bool, tokens map[string]struct{}, need int) bool {
	shared := 0
	for idx, word := range words {
		if !word.in(tokens) {
			continue
		}
		if rare[idx] {
			return true
		}
		shared++
		if shared >= need {
			return true
		}
	}

	return false
}

func (w queryWord) in(tokens map[string]struct{}) bool {
	for form := range w.forms {
		if _, ok := tokens[form]; ok {
			return true
		}
	}

	return false
}
