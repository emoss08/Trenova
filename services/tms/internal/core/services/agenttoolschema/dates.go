package agenttoolschema

import "github.com/emoss08/trenova/pkg/toolschema"

// Date is a parameter that takes a calendar day, YYYY-MM-DD.
//
// There were four date helpers, each with its own sentence: "YYYY-MM-DD.",
// "YYYY-MM-DD, a UTC day.", an ISO 8601 date-time and a local time, and a
// model reading forty tools met the same idea written four ways. Every date
// parameter now says its shape in the same words, carries an example, and
// declares a format the runtime asserts before the tool parses it.
func Date(description string) map[string]any {
	return dated(description, toolschema.FormatDate)
}

// DateTime is a parameter that takes an instant: a date and time with its
// UTC offset (RFC 3339).
func DateTime(description string) map[string]any {
	return dated(description, toolschema.FormatDateTime)
}

// LocalDateTime is a parameter that takes a wall-clock time with no zone,
// YYYY-MM-DDTHH:MM, which the tool reads in the timezone of the place it
// happens: a stop's location, or the organization's when that names none.
func LocalDateTime(description string) map[string]any {
	return dated(description, toolschema.FormatLocalDateTime)
}

func dated(description, name string) map[string]any {
	format, _ := toolschema.DateFormatNamed(name)
	text := format.Sentence
	if description != "" {
		text = description + " " + format.Sentence
	}

	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyFormat:      format.Name,
		toolschema.KeyExamples:    []string{format.Example},
		toolschema.KeyDescription: text,
	}
}
