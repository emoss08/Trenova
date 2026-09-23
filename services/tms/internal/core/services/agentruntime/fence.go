package agentruntime

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxToolResultChars = 12000
	untrustedOpenTag   = "<untrusted_data>"
	untrustedCloseTag  = "</untrusted_data>"
)

func FenceToolResult(toolName, payload string) string {
	truncated := truncateToolResult(payload)

	var builder strings.Builder
	builder.WriteString("Result from ")
	builder.WriteString(toolName)
	builder.WriteString(":\n")
	builder.WriteString(untrustedOpenTag)
	builder.WriteString("\n")
	builder.WriteString(stringutils.NeutralizeCloseTag(truncated, untrustedCloseTag))
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
