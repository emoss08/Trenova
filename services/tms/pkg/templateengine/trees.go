package templateengine

import (
	"strconv"
	"strings"
)

// templateErrorPrefix is what text/template puts in front of every parse failure.
const templateErrorPrefix = "template: "

// extractTemplatePosition splits a parse error into its position and message.
//
// text/template formats parse errors as "template: <name>:<line>: <message>" or
// "template: <name>:<line>:<column>: <message>". The template name can itself
// contain colons, so this parses from the right — anchoring on the "<colon><space>"
// that separates the location from the message, then reading the trailing numeric
// fields — rather than assuming a colon-free name.
//
// A message with no recognizable position comes back with zeros, which callers
// render as a non-positional diagnostic.
func extractTemplatePosition(message string) (line, column int, cleaned string) {
	trimmed := strings.TrimPrefix(message, templateErrorPrefix)

	location, rest, found := strings.Cut(trimmed, ": ")
	if !found {
		return 0, 0, message
	}

	text := strings.TrimSpace(rest)
	if text == "" {
		return 0, 0, message
	}

	// Read up to two trailing numeric fields: line, or line and column.
	fields := strings.Split(location, ":")
	numeric := make([]int, 0, 2)
	for i := len(fields) - 1; i >= 0 && len(numeric) < 2; i-- {
		value, err := strconv.Atoi(fields[i])
		if err != nil {
			break
		}
		numeric = append(numeric, value)
	}

	switch len(numeric) {
	case 1:
		return numeric[0], 0, text
	case 2:
		// Collected right-to-left, so the column came first.
		return numeric[1], numeric[0], text
	default:
		return 0, 0, message
	}
}
