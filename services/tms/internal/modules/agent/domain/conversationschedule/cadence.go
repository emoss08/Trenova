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

const spaceClass = `[\t\n\f\r \x{00a0}\x{3000}]`

// requestPattern reads a message as a scheduled request: an optional
// "/schedule", "every" or "each", a day word, an optional "at" time, and the
// request. It is the Desk composer's own pattern, so a message the composer
// sends as a schedule is one this reads as one. Saturday and Sunday are read
// as well as the weekdays; the cadence suffix (\w*) lets "weekdays" and
// "mondays" through.
var requestPattern = cadencePattern(
	`(?is)^(/schedule\s+)?((?:every|each)\s+` +
		`(weekday|day|morning|monday|tuesday|wednesday|thursday|friday|saturday|sunday|week)\w*` +
		`(?:\s+at\s+([\d:]+(?:\s*(?:am|pm)\b)?))?)([,:]?)\s*(.*)$`,
)

var spanishRequestPattern = cadencePattern(
	`(?is)^(/schedule\s+)?((?:cada|todos\s+los|todas\s+las)\s+` +
		`(d[ií]as?\s+(?:laborables?|h[aá]bil(?:es)?)|d[ií]as?|ma[nñ]anas?|lunes|martes|` +
		`mi[eé]rcoles|jueves|viernes|s[aá]bados?|domingos?|semanas?)\b` +
		`(?:\s+a\s+las?\s+([\d:]+(?:\s+y\s+(?:media|cuarto)\b)?` +
		`(?:\s*(?:a\.\s?m\.|p\.\s?m\.|(?:am|pm)\b)|\s+de\s+la\s+(?:ma[nñ]ana|tarde|noche)\b)?))?)` +
		`([,:]?)\s*(.*)$`,
)

var chineseRequestPattern = cadencePattern(
	`(?is)^(/schedule\s+)?((每(?:天|日)(?:早上|上午|早晨)|每(?:个|個)?工作(?:日|天)|工作日每天|` +
		`每(?:个|個)?(?:周|週|星期|礼拜|禮拜)[一二三四五六日天]|每(?:个|個)?(?:周|週|星期|礼拜|禮拜)|每(?:天|日))` +
		`(?:\s*((?:早上|上午|早晨|凌晨|中午|下午|晚上|傍晚)?\s*` +
		`(?:\d{1,2}\s*[:：]\s*\d{2}|(?:\d{1,2}|十[一二]?|[一二三四五六七八九两兩])\s*[点點]` +
		`(?:\s*(?:半|\d{1,2}(?:\s*分)?|[钟鐘整]))?)))?)\s*([,:，：、]?)\s*(.*)$`,
)

// commandPattern is the /schedule command, whatever follows it.
var commandPattern = regexp.MustCompile(`(?i)^/schedule(?:\s|$)`)

// leadingFiller is what is dropped from the front of the request: the comma
// after the time and a polite "please".
var leadingFiller = cadencePattern(
	`(?i)^(?:[,，、:：]|\s)*(?:(?:please|por\s+favor)\b(?:[,，]|\s)*|[请請][问問]?(?:[,，]|\s)*)?`,
)

// clockPattern is a time as typed: "7", "7:30", "730", "17:00", each with or
// without am/pm.
var clockPattern = cadencePattern(`(?i)^(\d{1,2})(?::?(\d{2}))?\s*(am|pm)?$`)

var spanishClockPattern = cadencePattern(
	`(?i)^([\d:]+?)(?:\s+y\s+(media|cuarto))?` +
		`(?:\s*(a\.\s?m\.|p\.\s?m\.|am|pm)|\s+de\s+la\s+(ma[nñ]ana|tarde|noche))?$`,
)

var chineseClockPattern = cadencePattern(
	`^(早上|上午|早晨|凌晨|中午|下午|晚上|傍晚)?\s*(?:(\d{1,2})\s*[:：]\s*(\d{2})|` +
		`(\d{1,2}|十[一二]?|[一二三四五六七八九两兩])\s*[点點](?:\s*(半|\d{1,2})(?:\s*分)?|\s*[钟鐘整])?)$`,
)

func cadencePattern(pattern string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(pattern, `\s`, spaceClass))
}

type meridiem int

const (
	meridiemNone meridiem = iota
	meridiemAM
	meridiemPM
	meridiemNight
)

type grammar struct {
	pattern     *regexp.Regexp
	unit        func(string) string
	clock       func(string) (hour, minute int, err error)
	needsMarker bool
}

var grammars = []grammar{
	{pattern: requestPattern, unit: strings.ToLower, clock: parseClock},
	{pattern: spanishRequestPattern, unit: spanishUnit, clock: parseSpanishClock},
	{
		pattern:     chineseRequestPattern,
		unit:        chineseUnit,
		clock:       parseChineseClock,
		needsMarker: true,
	},
}

type cadenceMatch struct {
	grammar grammar
	unit    string
	clock   string
	request string
}

func matchCadence(text string) (*cadenceMatch, bool) {
	for _, g := range grammars {
		match := g.pattern.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		if g.needsMarker && match[1] == "" && match[4] == "" && match[5] == "" {
			return nil, false
		}

		return &cadenceMatch{
			grammar: g,
			unit:    match[3],
			clock:   match[4],
			request: match[6],
		}, true
	}

	return nil, false
}

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

	if commandPattern.MatchString(text) {
		return true
	}
	_, ok := matchCadence(text)

	return ok
}

// ParseRequest reads a message as a scheduled request.
//
// The cadence is "every" or "each" and one of weekday, day, morning, a day of
// the week or week, then an optional "at h[:mm] am|pm". A day or a morning
// runs every day, a week runs on Mondays, and no time means 8:00 AM. A time
// without am or pm is read on a 24-hour clock. The rest is the request.
func ParseRequest(text string) (*Request, error) {
	text = strings.TrimSpace(text)
	match, ok := matchCadence(text)
	if !ok {
		return nil, errortypes.NewValidationError(
			"content",
			errortypes.ErrInvalid,
			"Start with when it should run, like “every Monday at 8am”, then what to ask",
		)
	}

	unit := match.grammar.unit(match.unit)
	hour, minute, err := match.grammar.clock(match.clock)
	if err != nil {
		return nil, err
	}

	prompt := leadingFiller.ReplaceAllString(strings.TrimSpace(match.request), "")
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

	match := clockPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, 0, invalidClock(raw)
	}

	hour, _ = strconv.Atoi(match[1])
	if match[2] != "" {
		minute, _ = strconv.Atoi(match[2])
	}

	return resolveClock(raw, hour, minute, latinMeridiem(match[3]))
}

func parseSpanishClock(raw string) (hour, minute int, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultHour, 0, nil
	}

	match := spanishClockPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, 0, invalidClock(raw)
	}
	digits := strings.TrimSuffix(match[1], ":")
	clock := clockPattern.FindStringSubmatch(digits)
	if clock == nil || clock[3] != "" {
		return 0, 0, invalidClock(raw)
	}

	hour, _ = strconv.Atoi(clock[1])
	if clock[2] != "" {
		minute, _ = strconv.Atoi(clock[2])
	}
	if match[2] != "" {
		if clock[2] != "" {
			return 0, 0, invalidClock(raw)
		}
		minute = 30
		if strings.EqualFold(match[2], "cuarto") {
			minute = 15
		}
	}

	period := latinMeridiem(match[3])
	switch strings.ToLower(match[4]) {
	case "mañana", "manana":
		period = meridiemAM
	case "tarde":
		period = meridiemPM
	case "noche":
		period = meridiemNight
	}

	return resolveClock(raw, hour, minute, period)
}

func parseChineseClock(raw string) (hour, minute int, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultHour, 0, nil
	}

	match := chineseClockPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, 0, invalidClock(raw)
	}

	if match[2] != "" {
		hour, _ = strconv.Atoi(match[2])
		minute, _ = strconv.Atoi(match[3])
	} else {
		hour = chineseNumber(match[4])
		switch match[5] {
		case "":
		case "半":
			minute = 30
		default:
			minute, _ = strconv.Atoi(match[5])
		}
	}

	period := meridiemNone
	switch match[1] {
	case "早上", "上午", "早晨", "凌晨":
		period = meridiemAM
	case "中午", "下午":
		period = meridiemPM
	case "晚上", "傍晚":
		period = meridiemNight
	}

	return resolveClock(raw, hour, minute, period)
}

func latinMeridiem(raw string) meridiem {
	switch strings.ToLower(strings.Join(strings.FieldsFunc(raw, isClockNoise), "")) {
	case "am":
		return meridiemAM
	case "pm":
		return meridiemPM
	default:
		return meridiemNone
	}
}

func isClockNoise(r rune) bool {
	return r == '.' || unicode.IsSpace(r)
}

func chineseNumber(raw string) int {
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}

	digits := map[rune]int{
		'一': 1, '二': 2, '两': 2, '兩': 2, '三': 3, '四': 4,
		'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	}
	total := 0
	for _, r := range raw {
		if r == '十' {
			total += 10
			continue
		}
		total += digits[r]
	}

	return total
}

func resolveClock(
	raw string,
	hour, minute int,
	period meridiem,
) (resolvedHour, resolvedMinute int, err error) {
	if minute > 59 {
		return 0, 0, invalidClock(raw)
	}

	switch period {
	case meridiemNone:
		if hour > 23 {
			return 0, 0, invalidClock(raw)
		}
	default:
		if hour < 1 || hour > 12 {
			return 0, 0, invalidClock(raw)
		}
		switch {
		case hour == 12 && period == meridiemPM:
		case hour == 12:
			hour = 0
		case period != meridiemAM:
			hour += 12
		}
	}

	return hour, minute, nil
}

func invalidClock(raw string) error {
	return errortypes.NewValidationError(
		"content",
		errortypes.ErrInvalid,
		"“{0}” isn't a time of day. Try “7:30 am” or “17:30”.",
		raw,
	)
}

func spanishUnit(raw string) string {
	unit := strings.ToLower(strings.Join(strings.Fields(raw), " "))
	unit = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ñ", "n").Replace(unit)

	switch {
	case strings.HasPrefix(unit, "dia") && strings.Contains(unit, " "):
		return "weekday"
	case strings.HasPrefix(unit, "dia"):
		return "day"
	case strings.HasPrefix(unit, "manana"):
		return "morning"
	case strings.HasPrefix(unit, "semana"):
		return "week"
	}

	return spanishWeekdays[strings.TrimSuffix(unit, "s")]
}

var spanishWeekdays = map[string]string{
	"lune":     "monday",
	"marte":    "tuesday",
	"miercole": "wednesday",
	"jueve":    "thursday",
	"vierne":   "friday",
	"sabado":   "saturday",
	"domingo":  "sunday",
}

func chineseUnit(raw string) string {
	switch {
	case strings.Contains(raw, "工作"):
		return "weekday"
	case strings.ContainsAny(raw, "周週期拜"):
		last, _ := utf8.DecodeLastRuneInString(raw)
		if day, ok := chineseWeekdays[last]; ok {
			return day
		}
		return "week"
	case strings.ContainsAny(raw, "早上晨"):
		return "morning"
	default:
		return "day"
	}
}

var chineseWeekdays = map[rune]string{
	'一': "monday",
	'二': "tuesday",
	'三': "wednesday",
	'四': "thursday",
	'五': "friday",
	'六': "saturday",
	'日': "sunday",
	'天': "sunday",
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
