package agentruntime

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxToolResultChars = 12000
	untrustedOpenTag   = "<untrusted_data>"
	untrustedCloseTag  = "</untrusted_data>"
)

func FenceToolResult(toolName, payload string) string {
	return FenceUntrusted("Result from "+toolName+":", truncateToolResult(payload))
}

func FenceUntrusted(header, payload string) string {
	var builder strings.Builder
	builder.WriteString(header)
	builder.WriteString("\n")
	builder.WriteString(untrustedOpenTag)
	builder.WriteString("\n")
	builder.WriteString(stringutils.NeutralizeCloseTag(payload, untrustedCloseTag))
	builder.WriteString("\n")
	builder.WriteString(untrustedCloseTag)

	return builder.String()
}

// truncateToolResult cuts an oversized payload and says so in words the reader
// has to act on.
//
// The old cut was a bare byte slice with "…(truncated)" appended, which left a
// half-finished JSON document and no instruction. A model given one read the
// rows it could see, announced "let me see the remaining two workers from the
// truncated data", and then filled them in from nothing. Naming the loss and
// the remedy is the difference between a short answer and an invented one.
func truncateToolResult(payload string) string {
	if len(payload) <= maxToolResultChars {
		return payload
	}
	if shortened, ok := shortenObjectLists(payload); ok {
		return shortened
	}

	// Cutting mid-rune would put invalid UTF-8 on the wire. Walking back to a
	// boundary costs at most three bytes.
	cut := maxToolResultChars
	for cut > 0 && !utf8.RuneStart(payload[cut]) {
		cut--
	}

	var builder strings.Builder
	builder.WriteString(payload[:cut])
	builder.WriteString("\n\n[This result was cut off here: it was too long to return in full, " +
		"so the text above ends mid-record and the records after it are missing entirely. " +
		"Do not infer, complete, or count anything from the cut-off portion. " +
		"Narrow your filters, or where the tool takes limit and offset ask for a " +
		"smaller page and continue from its nextOffset, and tell the person you are " +
		"working from a partial result until you do.]")

	return builder.String()
}

// listCutNoteBudget is the room kept for the note that says which lists were
// shortened.
const listCutNoteBudget = 600

// shortenObjectLists fits an oversized JSON object by shortening its
// top-level lists and keeping every other field whole, written first. A
// result's keys arrive sorted, so a byte cut kept the dispatch board's drivers
// and moves and dropped its summary and note: asked which loads need a driver,
// a model was handed half a list and none of the totals it was asked for.
func shortenObjectLists(payload string) (string, bool) {
	object, err := jsonflex.DecodeObject([]byte(payload))
	if err != nil {
		return "", false
	}

	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	fields := make([]string, 0, len(keys))
	lists := make([]string, 0, len(keys))
	budget := maxToolResultChars - listCutNoteBudget - 2
	for _, key := range keys {
		if jsonflex.KindOf(object[key]) == jsonflex.KindArray {
			lists = append(lists, key)
			continue
		}
		fields = append(fields, key)
		budget -= len(key) + len(object[key]) + 4
	}
	if len(lists) == 0 || budget <= 0 {
		return "", false
	}

	slices.SortFunc(lists, func(a, b string) int { return len(object[a]) - len(object[b]) })
	kept := make(map[string][]byte, len(lists))
	cuts := make([]string, 0, len(lists))
	for idx, key := range lists {
		items, decodeErr := jsonflex.DecodeArray(object[key])
		if decodeErr != nil {
			return "", false
		}
		share := budget / (len(lists) - idx)
		var list strings.Builder
		list.WriteByte('[')
		shown := 0
		for _, item := range items {
			if list.Len()+len(item)+2 > share-len(key)-4 {
				break
			}
			if shown > 0 {
				list.WriteByte(',')
			}
			list.Write(item)
			shown++
		}
		list.WriteByte(']')
		kept[key] = []byte(list.String())
		budget -= len(key) + list.Len() + 4
		if shown < len(items) {
			cuts = append(cuts, fmt.Sprintf("%s shows %d of %d", key, shown, len(items)))
		}
	}

	var builder strings.Builder
	builder.WriteByte('{')
	for idx, key := range append(fields, lists...) {
		if idx > 0 {
			builder.WriteByte(',')
		}
		name, marshalErr := sonic.Marshal(key)
		if marshalErr != nil {
			return "", false
		}
		builder.Write(name)
		builder.WriteByte(':')
		if value, ok := kept[key]; ok {
			builder.Write(value)
			continue
		}
		builder.Write(object[key])
	}
	builder.WriteByte('}')
	if len(cuts) > 0 {
		builder.WriteString("\n\n[This result was too long to return in full, so its lists were " +
			"shortened: " + strings.Join(cuts, "; ") + ". Every other field, such as totals, " +
			"counts and notes, is complete: answer counts from those. Do not infer or count " +
			"the missing records from the ones shown; narrow your filters, or where the tool " +
			"takes limit and offset ask for a smaller page, and tell the person you are " +
			"working from part of the list.]")
	}

	return builder.String(), true
}

// UnfenceToolResult reads a stored tool result back out of its fence: the
// tool it came from and the payload as the tool returned it, with a close
// tag the payload carried restored. Content that is not a fenced result,
// such as the failure line a tool that errored leaves, is reported as not
// fenced and left to the caller.
func UnfenceToolResult(content string) (toolName, payload string, ok bool) {
	const prefix = "Result from "

	if !strings.HasPrefix(content, prefix) {
		return "", "", false
	}
	rest := content[len(prefix):]
	nameEnd := strings.Index(rest, ":\n"+untrustedOpenTag+"\n")
	if nameEnd < 0 {
		return "", "", false
	}
	toolName = rest[:nameEnd]
	body := rest[nameEnd+len(":\n"+untrustedOpenTag+"\n"):]
	// The payload's own close tags are neutralised, so the first one is the
	// fence's. Whatever follows it is a note to the model, such as the one
	// saying the result is already shown to the person, not part of the result.
	end := strings.Index(body, "\n"+untrustedCloseTag)
	if end < 0 {
		return "", "", false
	}
	body = body[:end]

	return toolName, stringutils.RestoreCloseTag(body, untrustedCloseTag), true
}
