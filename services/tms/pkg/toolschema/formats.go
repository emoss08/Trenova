package toolschema

import (
	"fmt"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// KeyFormat and KeyExamples are the JSON Schema keywords a date parameter
// carries: the shape its value must have, and one value of that shape.
const (
	KeyFormat   = "format"
	KeyExamples = "examples"
)

// The three shapes a date or a time is sent in. A day is a calendar day; a
// date-time is an instant, with its offset; a local date-time is a wall-clock
// time with no zone, read in the timezone of the place it happens (a stop's
// location, the organization), which is what a person says when they say
// "Tuesday at 8".
const (
	FormatDate          = "date"
	FormatDateTime      = "date-time"
	FormatLocalDateTime = "local-date-time"
)

// LocalDateTimeLayout is the one layout a local date-time is accepted in.
const LocalDateTimeLayout = "2006-01-02T15:04"

// DateFormat is what one date format looks like, written for a model: the
// shape, an example, and the sentence a parameter of that format carries.
type DateFormat struct {
	Name     string
	Shape    string
	Example  string
	Sentence string
}

var dateFormats = map[string]DateFormat{
	FormatDate: {
		Name:     FormatDate,
		Shape:    "YYYY-MM-DD",
		Example:  "2026-10-01",
		Sentence: "A calendar day as YYYY-MM-DD, such as 2026-10-01.",
	},
	FormatDateTime: {
		Name:    FormatDateTime,
		Shape:   "YYYY-MM-DDTHH:MM:SS with its UTC offset",
		Example: "2026-10-01T08:00:00-05:00",
		Sentence: "A date and time with its UTC offset (RFC 3339), such as " +
			"2026-10-01T08:00:00-05:00.",
	},
	FormatLocalDateTime: {
		Name:    FormatLocalDateTime,
		Shape:   "YYYY-MM-DDTHH:MM with no UTC offset",
		Example: "2026-10-01T08:00",
		Sentence: "A local date and time as YYYY-MM-DDTHH:MM with no UTC offset, such as " +
			"2026-10-01T08:00, read in the timezone of where it happens; never Unix seconds.",
	},
}

// DateFormatNamed is the date format of that name, if it is one.
func DateFormatNamed(name string) (DateFormat, bool) {
	format, ok := dateFormats[name]

	return format, ok
}

// FormatOf reads the format a property declares.
func FormatOf(property map[string]any) string {
	format, _ := property[KeyFormat].(string)

	return format
}

// dateFormatValidators replace the validator's own date and date-time, and
// add local-date-time. Each takes an empty string: several tools read an
// empty date as "clear it" or "none", and that is the tool's to decide, not
// the format's. Anything else has to parse in exactly the declared layout.
var dateFormatValidators = []*jsonschema.Format{
	{Name: FormatDate, Validate: layoutValidator(time.DateOnly)},
	{Name: FormatDateTime, Validate: layoutValidator(time.RFC3339)},
	{Name: FormatLocalDateTime, Validate: layoutValidator(LocalDateTimeLayout)},
}

func layoutValidator(layout string) func(any) error {
	return func(value any) error {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil
		}
		if _, err := time.Parse(layout, text); err != nil {
			return fmt.Errorf("not in the layout %s: %w", layout, err)
		}

		return nil
	}
}

// FormatMessage is the one sentence a value of the wrong shape is refused
// with: the shape the parameter takes and an example. A string of digits is
// named as the Unix time it almost certainly is, since that is the mistake
// a model makes and a converted number would land on the wrong day.
func FormatMessage(got any, want string) (string, bool) {
	format, ok := dateFormats[want]
	if !ok {
		return "", false
	}
	text := strings.TrimSpace(fmt.Sprint(got))
	if text != "" && strings.Trim(text, "0123456789") == "" {
		return fmt.Sprintf("%s is a Unix time; send a %s as %s, such as %s",
			text, format.Name, format.Shape, format.Example), true
	}

	return fmt.Sprintf("%q is not a %s; send %s, such as %s",
		text, format.Name, format.Shape, format.Example), true
}
