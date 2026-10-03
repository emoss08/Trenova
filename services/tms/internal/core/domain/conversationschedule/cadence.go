package conversationschedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
)

// requestPattern reads a message as a scheduled request: an optional
// "/schedule", "every" or "each", a day word, an optional "at" time, and the
// request. It is the Desk composer's own pattern, so a message the composer
// sends as a schedule is one this reads as one. Saturday and Sunday are read
// as well as the weekdays; the cadence suffix (\w*) lets "weekdays" and
// "mondays" through.
var requestPattern = regexp.MustCompile(
	`(?is)^(?:/schedule\s+)?(every|each)\s+` +
		`(weekday|day|morning|monday|tuesday|wednesday|thursday|friday|saturday|sunday|week)\w*` +
		`(?:\s+at\s+([\d:]+(?:\s*(?:am|pm)\b)?))?[,:]?\s*(.*)$`,
)

// commandPattern is the /schedule command, whatever follows it.
var commandPattern = regexp.MustCompile(`(?i)^/schedule(?:\s|$)`)

// leadingFiller is what is dropped from the front of the request: the comma
// after the time and a polite "please".
var leadingFiller = regexp.MustCompile(`(?i)^(?:,|\s)*(?:please\b[,\s]*)?`)

// clockPattern is a time as typed: "7", "7:30", "730", "17:00", each with or
// without am/pm.
var clockPattern = regexp.MustCompile(`(?i)^(\d{1,2})(?::?(\d{2}))?\s*(am|pm)?$`)

// defaultHour is when a request with no time runs: 8:00 AM.
const defaultHour = 8

var weekdayNumbers = map[string]int{
	"sunday":    0,
	"monday":    1,
	"tuesday":   2,
	"wednesday": 3,
	"thursday":  4,
	"friday":    5,
	"saturday":  6,
}

// Request is a message read as a scheduled request.
type Request struct {
	// Prompt is what each run asks, with its first letter capitalised.
	Prompt string
	// Cadence is the cadence as the person reads it: "Every weekday · 7:30 AM".
	Cadence string
	// CronExpression is the same cadence as a five-field cron.
	CronExpression string
}

// IsRequest reports whether a message asks for a schedule rather than an
// answer: the /schedule command, or a message that opens with a cadence the
// parser reads. "Every time I open this…" is a question, not a schedule.
func IsRequest(text string) bool {
	text = strings.TrimSpace(text)

	return commandPattern.MatchString(text) || requestPattern.MatchString(text)
}

// ParseRequest reads a message as a scheduled request.
//
// The cadence is "every" or "each" and one of weekday, day, morning, a day of
// the week or week, then an optional "at h[:mm] am|pm". A day or a morning
// runs every day, a week runs on Mondays, and no time means 8:00 AM. A time
// without am or pm is read on a 24-hour clock. The rest is the request.
func ParseRequest(text string) (*Request, error) {
	text = strings.TrimSpace(text)
	match := requestPattern.FindStringSubmatch(text)
	if match == nil {
		return nil, errortypes.NewValidationError(
			"content",
			errortypes.ErrInvalid,
			"Start with when it should run, like “every Monday at 8am”, then what to ask",
		)
	}

	unit := strings.ToLower(match[2])
	hour, minute, err := parseClock(match[3])
	if err != nil {
		return nil, err
	}

	prompt := leadingFiller.ReplaceAllString(strings.TrimSpace(match[4]), "")
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, errortypes.NewValidationError(
			"content",
			errortypes.ErrRequired,
			"Say what to ask after when it should run",
		)
	}
	if utf8.RuneCountInString(prompt) > MaxPromptLength {
		return nil, errortypes.NewValidationError(
			"content",
			errortypes.ErrInvalid,
			"A scheduled request cannot be longer than 2000 characters",
		)
	}

	label, days := cadenceOf(unit)
	cron := fmt.Sprintf("%d %d * * %s", minute, hour, days)
	if err = cronutils.Validate(cron); err != nil {
		return nil, fmt.Errorf("schedule cadence: %w", err)
	}

	return &Request{
		Prompt:         capitalise(prompt),
		Cadence:        label + " · " + clockLabel(hour, minute),
		CronExpression: cron,
	}, nil
}

// cadenceOf is the label and the cron day-of-week field for a unit.
func cadenceOf(unit string) (label, days string) {
	switch unit {
	case "day", "morning":
		return "Every day", "*"
	case "weekday":
		return "Every weekday", "1-5"
	case "week":
		return "Every Monday", "1"
	default:
		return "Every " + capitalise(unit), strconv.Itoa(weekdayNumbers[unit])
	}
}

// parseClock reads the time after "at"; an empty one is the default hour.
func parseClock(raw string) (hour, minute int, err error) {
	// "at 8: what's late?" leaves the colon on the time.
	raw = strings.TrimSuffix(strings.TrimSpace(raw), ":")
	if raw == "" {
		return defaultHour, 0, nil
	}

	invalid := errortypes.NewValidationError(
		"content",
		errortypes.ErrInvalid,
		"“{0}” isn't a time of day. Try “7:30 am” or “17:30”.",
		raw,
	)
	match := clockPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, 0, invalid
	}

	hour, _ = strconv.Atoi(match[1])
	if match[2] != "" {
		minute, _ = strconv.Atoi(match[2])
	}
	if minute > 59 {
		return 0, 0, invalid
	}

	switch strings.ToLower(match[3]) {
	case "am":
		if hour < 1 || hour > 12 {
			return 0, 0, invalid
		}
		if hour == 12 {
			hour = 0
		}
	case "pm":
		if hour < 1 || hour > 12 {
			return 0, 0, invalid
		}
		if hour != 12 {
			hour += 12
		}
	default:
		if hour > 23 {
			return 0, 0, invalid
		}
	}

	return hour, minute, nil
}

// clockLabel writes a time the way the card shows it: "7:30 AM".
func clockLabel(hour, minute int) string {
	suffix := "AM"
	if hour >= 12 {
		suffix = "PM"
	}
	display := hour % 12
	if display == 0 {
		display = 12
	}

	return fmt.Sprintf("%d:%02d %s", display, minute, suffix)
}

func capitalise(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if first == utf8.RuneError {
		return text
	}

	return string(unicode.ToUpper(first)) + text[size:]
}
