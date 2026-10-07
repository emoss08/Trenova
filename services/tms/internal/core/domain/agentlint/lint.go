// Package agentlint reads an agent's instructions against the tools it
// holds and points out each thing it is told to do that none of them can.
//
// What it looks for comes from the tool registry: the records a tool reads
// or changes, by the name the permission registry gives them, and the
// operation it performs. A sentence that names a record and an operation no
// held tool performs on it is a finding. Sentences that say what not to do
// are guardrails, not capabilities, and are skipped.
package agentlint

import (
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/permission"
)

// MaxFindings bounds what one read of the instructions reports.
const MaxFindings = 20

// Tool is one entry of the tool registry, as the lint reads it.
type Tool struct {
	Name          string
	Resource      permission.Resource
	ResourceLabel string
	Operation     permission.Operation
	Held          bool
}

// Finding is a sentence of the instructions asking for something no held
// tool does. Start and End are byte offsets of the sentence.
type Finding struct {
	Start         int
	End           int
	Excerpt       string
	Resource      permission.Resource
	ResourceLabel string
	Operation     permission.Operation
	// Tools are the tools in the registry that would do it, by name. Empty
	// when none does, so no change of tools would help.
	Tools []string
}

// operationVerbs are the words a person writes for each operation the
// registry knows. They name operations, not capabilities: which record an
// operation applies to, and whether any tool performs it, is the registry's.
var operationVerbs = map[permission.Operation][]string{
	permission.OpRead:    {"check", "look up", "look at", "find", "read", "review", "search", "see", "pull up"},
	permission.OpCreate:  {"create", "add", "make", "open", "enter", "book", "draft", "build"},
	permission.OpUpdate:  {"update", "change", "set", "edit", "correct", "fix", "mark", "adjust", "move"},
	permission.OpDelete:  {"delete", "remove"},
	permission.OpExport:  {"export", "download"},
	permission.OpImport:  {"import", "upload"},
	permission.OpApprove: {"approve", "commit", "accept"},
	permission.OpReject:  {"reject", "decline", "deny"},
	permission.OpAssign:  {"assign", "dispatch"},
	permission.OpArchive: {"archive"},
	permission.OpSubmit:  {"submit", "send"},
	permission.OpCancel:  {"cancel", "void"},
	permission.OpClose:   {"close"},
	permission.OpReopen:  {"reopen"},
	permission.OpResolve: {"resolve", "settle"},
}

// guardWords open a sentence that says what the agent must not do.
var guardWords = []string{"never", "not", "don't", "dont", "do not", "avoid", "without", "nor", "no"}

type term struct {
	text     string
	resource permission.Resource
	label    string
}

type sentence struct {
	start, end int
	text       string
	lower      string
}

// Lint returns, in the order they appear, the things the instructions ask
// for that the held tools cannot do.
func Lint(instructions string, registry []Tool) []Finding {
	if strings.TrimSpace(instructions) == "" || len(registry) == 0 {
		return []Finding{}
	}

	terms := resourceTerms(registry)
	byCapability := toolsByCapability(registry)
	findings := make([]Finding, 0)
	reported := make(map[capability]struct{})

	for _, sent := range sentences(instructions) {
		if guarded(sent.lower) {
			continue
		}
		for _, found := range mentionedResources(sent.lower, terms) {
			for _, op := range mentionedOperations(sent.lower) {
				key := capability{resource: found.resource, operation: op}
				if _, ok := reported[key]; ok {
					continue
				}
				tools := byCapability[key]
				if tools.held {
					continue
				}
				if len(tools.names) == 0 && op != permission.OpRead {
					continue
				}
				reported[key] = struct{}{}
				findings = append(findings, Finding{
					Start:         sent.start,
					End:           sent.end,
					Excerpt:       sent.text,
					Resource:      found.resource,
					ResourceLabel: found.label,
					Operation:     op,
					Tools:         tools.names,
				})
				if len(findings) == MaxFindings {
					return findings
				}
			}
		}
	}

	return findings
}

type capability struct {
	resource  permission.Resource
	operation permission.Operation
}

type capabilityTools struct {
	names []string
	held  bool
}

func toolsByCapability(registry []Tool) map[capability]capabilityTools {
	out := make(map[capability]capabilityTools, len(registry))
	for idx := range registry {
		tool := &registry[idx]
		if tool.Resource == "" {
			continue
		}
		key := capability{resource: tool.Resource, operation: tool.Operation}
		entry := out[key]
		entry.names = append(entry.names, tool.Name)
		entry.held = entry.held || tool.Held
		out[key] = entry
	}
	for key, entry := range out {
		slices.Sort(entry.names)
		out[key] = entry
	}
	return out
}

// resourceTerms are the names of every record some tool reaches, singular
// and plural, longest first so "invoice run" is read before "invoice".
func resourceTerms(registry []Tool) []term {
	seen := make(map[string]struct{}, len(registry)*2)
	terms := make([]term, 0, len(registry)*2)
	for idx := range registry {
		tool := &registry[idx]
		label := strings.ToLower(strings.TrimSpace(tool.ResourceLabel))
		if label == "" || tool.Resource == "" {
			continue
		}
		for _, form := range []string{label, plural(label)} {
			if _, ok := seen[form]; ok {
				continue
			}
			seen[form] = struct{}{}
			terms = append(terms, term{text: form, resource: tool.Resource, label: tool.ResourceLabel})
		}
	}
	sort.SliceStable(terms, func(i, j int) bool { return len(terms[i].text) > len(terms[j].text) })
	return terms
}

func plural(word string) string {
	switch {
	case strings.HasSuffix(word, "y") && len(word) > 1 && !strings.ContainsRune("aeiou", rune(word[len(word)-2])):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s"), strings.HasSuffix(word, "x"), strings.HasSuffix(word, "ch"), strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}

func mentionedResources(lower string, terms []term) []term {
	taken := make([]bool, len(lower))
	found := make([]term, 0)
	seen := make(map[permission.Resource]struct{})
	for _, candidate := range terms {
		for _, at := range wordIndexes(lower, candidate.text) {
			if slices.Contains(taken[at:at+len(candidate.text)], true) {
				continue
			}
			for idx := at; idx < at+len(candidate.text); idx++ {
				taken[idx] = true
			}
			if _, ok := seen[candidate.resource]; ok {
				continue
			}
			seen[candidate.resource] = struct{}{}
			found = append(found, candidate)
		}
	}
	return found
}

func mentionedOperations(lower string) []permission.Operation {
	ops := make([]permission.Operation, 0, 2)
	for op, verbs := range operationVerbs {
		for _, verb := range verbs {
			if mentionsVerb(lower, verb) {
				ops = append(ops, op)
				break
			}
		}
	}
	slices.Sort(ops)
	return ops
}

// mentionsVerb finds a verb in its common forms: "update", "updates",
// "updated", "updating".
func mentionsVerb(lower, verb string) bool {
	stem := strings.TrimSuffix(verb, "e")
	for _, form := range []string{verb, verb + "s", verb + "es", stem + "ed", stem + "ing", verb + "ed", verb + "ing"} {
		if len(wordIndexes(lower, form)) > 0 {
			return true
		}
	}
	return false
}

func guarded(lower string) bool {
	for _, word := range guardWords {
		if len(wordIndexes(lower, word)) > 0 {
			return true
		}
	}
	return false
}

// wordIndexes finds whole-word occurrences of needle in haystack.
func wordIndexes(haystack, needle string) []int {
	var out []int
	for from := 0; from <= len(haystack)-len(needle); {
		idx := strings.Index(haystack[from:], needle)
		if idx < 0 {
			break
		}
		at := from + idx
		end := at + len(needle)
		if boundaryBefore(haystack, at) && boundaryAfter(haystack, end) {
			out = append(out, at)
		}
		from = at + 1
	}
	return out
}

func boundaryBefore(text string, at int) bool {
	if at == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:at])
	return !isWordRune(r)
}

func boundaryAfter(text string, at int) bool {
	if at >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[at:])
	return !isWordRune(r)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\''
}

// sentences splits instructions at sentence ends and line breaks, keeping
// each sentence's offsets in the original text.
func sentences(text string) []sentence {
	out := make([]sentence, 0, strings.Count(text, ".")+strings.Count(text, "\n")+1)
	start := 0
	flush := func(end int) {
		raw := text[start:end]
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			lead := strings.Index(raw, trimmed)
			out = append(out, sentence{
				start: start + lead,
				end:   start + lead + len(trimmed),
				text:  trimmed,
				lower: strings.ToLower(trimmed),
			})
		}
		start = end
	}
	for idx := 0; idx < len(text); idx++ {
		switch text[idx] {
		case '\n':
			flush(idx)
		case '.', '!', '?', ';':
			if idx+1 == len(text) || text[idx+1] == ' ' || text[idx+1] == '\n' {
				flush(idx + 1)
			}
		}
	}
	flush(len(text))
	return out
}
