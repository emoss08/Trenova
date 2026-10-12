package agentruntime

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	askUserName = "ask_user"

	// maxAskOptions bounds one question. A list longer than this is a search,
	// not a choice, and a person scanning it will read none of it.
	maxAskOptions = 8
	// maxAskLabelChars keeps an option to something readable on a chip.
	maxAskLabelChars = 60
	// maxAskFields bounds one form. A write refused for more missing values
	// than this was called with too little looked up, not too little asked.
	maxAskFields = 8

	// maxOpenQuestionRunes is the longest a question answered only by typing
	// may be. One value fits in one short sentence; the question that asked
	// for a whole shipment in a single box ran to well over twice this.
	maxOpenQuestionRunes = 160
	// maxOpenQuestionSeparators is the most commas and semicolons such a
	// question may hold. A second one is a list of the things being asked for.
	maxOpenQuestionSeparators = 1
)

// changeUnshownAsk is the loop treating an ask_user call the runtime refused
// as a refusal: it is marked failed, the turn has not asked anything, and the
// question may be asked again properly. Before it such a call counted as
// asked, so a turn could end in silence on a question nobody saw. It is asked
// only of a call the runtime refused, so a turn whose questions are all shown
// records no change marker.
const changeUnshownAsk = "agent-loop-unshown-ask"

// The names of the properties an option, a typed value and a field carry, as
// the schema declares them and the arguments are read back by.
const (
	askPropKey        = "key"
	askPropLabel      = "label"
	askPropType       = "type"
	askPropRecordKind = "recordKind"
)

// The types a typed value can take. A record is picked from a search of one
// kind of record rather than typed.
const (
	askTypeText     = "text"
	askTypeNumber   = "number"
	askTypeDate     = "date"
	askTypeDateTime = "datetime"
	askTypeRecord   = "record"
)

// askValueTypes is every type the thread can render a box for, in the order
// the schema lists them.
var askValueTypes = []string{
	askTypeText, askTypeNumber, askTypeDate, askTypeDateTime, askTypeRecord,
}

// askUserDescription is a constant for the same reason findToolsDescription is:
// the i18n extractor harvests Description: fields, and this text is addressed to
// a model, so translating it would change what the assistant is told based on
// the operator's locale.
const askUserDescription = "Ask the person for one value you need and cannot safely " +
	"infer: a required parameter that is missing, a request that is ambiguous between a " +
	"few readings, or a choice where picking wrongly would do work nobody asked for. " +
	"Infer everything you can first — a request naming a window, a date range or a " +
	"customer has answered the question already — and never ask for something the " +
	"person has already told you. Ask for one value per call. For a choice, give options, " +
	"shown as buttons; the person can also type their own value unless you set " +
	"allowOther false. For a typed value, set input to its type: a number, a date, a " +
	"date and time, or a record such as a customer or a location, which the person " +
	"picks from a search. To create or change a record, do not collect its details " +
	"first: call the write tool with what you know, and only when it is refused for " +
	"missing values, ask for exactly those with fields, as one form. Never use fields " +
	"as an intake form up front. Ask once, then stop and wait: never guess a value " +
	"after asking for it."

// multiPartAskRefusal answers a question asked into a single box that asks
// for several values. A model asked what shipment to book for a customer,
// pickup and delivery with their windows, service and shipment type, freight
// and price, all in one question: a form in disguise, answered by a person
// retyping a load sheet into a chat box.
const multiPartAskRefusal = "This question asks for several values at once, and a " +
	"person can only answer it by typing all of them into one box. Ask for the one " +
	"value you need most, in one short question, with any examples in otherHint " +
	"rather than the question. To create or change a record, do not collect its " +
	"details first: call the write tool with what you already know, and if it is " +
	"refused for missing values, call ask_user with fields for exactly those values."

// mixedAskRefusal answers a question that offers choices and a form at once.
// The person cannot both pick one value and fill several.
const mixedAskRefusal = "ask_user takes options, for one choice, or fields, for the " +
	"values a write tool said are missing, but not both. Ask for the choice on its " +
	"own, or send only the fields."

// unshowableFieldsRefusal answers a form none of whose fields could be shown.
const unshowableFieldsRefusal = "None of the fields could be shown. Each needs a key, " +
	"a label and a type of text, number, date, datetime or record, and a record field " +
	"needs the recordKind of the record it picks, such as customer or location. Send " +
	"them again with those, or ask for one value without fields."

// unattendedAskRefusal answers ask_user in a run nobody is watching. The spec
// is withheld from those runs, but a model that remembers the tool from its
// instructions can still name it, and a question no one will see used to end
// the run as though it had been answered.
const unattendedAskRefusal = "Nobody is watching this run, so there is no one to ask. " +
	"Decide as your instructions allow, or call raise_exception to hand the question " +
	"to a person."

// repeatedAskRefusal answers a question the conversation has already put to
// the person. The Homepage Widget Builder asked which dashboard twice in a
// row; the second time the person had already answered, and being asked again
// reads as not having been heard.
const repeatedAskRefusal = "You already asked the person this, and their answer is in the " +
	"conversation. Use it. If it does not settle the question, say what you are " +
	"assuming and carry on rather than asking again."

// askedQuestions collects the questions put to the person in the replayed
// conversation's last turns, in comparable form.
//
// A question the runtime refused was never shown, so it does not count: a
// model told to ask for one value instead of several, or to send its form
// again, would otherwise be refused as repeating itself when it did.
func askedQuestions(history []conversation.Message) map[string]struct{} {
	asked := make(map[string]struct{}, 2)
	refused := make(map[string]struct{})
	turns := 0
	for idx := len(history) - 1; idx >= 0 && turns < recentToolTurns; idx-- {
		message := history[idx]
		if message.Role == conversation.RoleUser {
			turns++
			continue
		}
		if message.Role == conversation.RoleTool {
			if message.ToolFailed && message.ToolName == askUserName &&
				message.ToolCallID != "" {
				refused[message.ToolCallID] = struct{}{}
			}
			continue
		}
		for _, call := range message.ToolCalls {
			if call.Name != askUserName {
				continue
			}
			if _, wasRefused := refused[call.ID]; wasRefused {
				continue
			}
			if question := comparableQuestion(
				stringArg(call.Arguments, "question"),
			); question != "" {
				asked[question] = struct{}{}
			}
		}
	}

	return asked
}

// comparableQuestion folds the differences that do not make a question new:
// case, punctuation and spacing.
func comparableQuestion(question string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(question) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		default:
			space = true
		}
	}

	return b.String()
}

// askUserSpec is the second tool the runtime answers itself.
//
// Every turn carries it, disclosed or not, because asking for a missing value is
// part of holding a conversation rather than a capability an organization grants.
// An agent that may read shipments but may not ask which shipment is an agent
// that has to guess, and guessing is what this exists to stop.
func askUserSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        askUserName,
		Description: askUserDescription,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"question": map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "What you need, as one short, plain question about one " +
						"value. \"Which window should the report cover?\" Examples of an " +
						"answer go in otherHint, not here.",
				},
				"options": askOptionsSchema(),
				"allowOther": map[string]any{
					toolschema.KeyType: toolschema.TypeBoolean,
					toolschema.KeyDescription: "Whether the person may type a value instead of " +
						"picking one. True unless the value is genuinely closed, such as " +
						"a fixed set of statuses. Ignored with fields.",
				},
				"otherHint": map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "What a typed value should look like, shown in the " +
						"box. \"Number of days\".",
				},
				"input":  askInputSchema(),
				"fields": askFieldsSchema(),
			},
			toolschema.KeyRequired:             []string{"question"},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func askOptionsSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "The choices, best first, when the value is a choice. Where it " +
			"is constrained — a parameter with allowed values, a status, a category — " +
			"these must be the real ones, not examples. Leave them out when the person " +
			"types the value, and never send them with fields.",
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"value": map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "What you will use if this is picked, " +
						"exactly as the tool needs it. Defaults to the label.",
				},
				askPropLabel: map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "How it reads to a person. \"30 days\".",
				},
				"detail": map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "One short clause of consequence, when " +
						"the label does not carry it.",
				},
			},
			toolschema.KeyRequired:             []string{askPropLabel},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func askInputSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyDescription: "The type of the value the person types when allowOther is " +
			"true. Leave it out for plain text.",
		toolschema.KeyProperties: map[string]any{
			askPropType:       askValueTypeSchema(),
			askPropRecordKind: askRecordKindSchema(),
		},
		toolschema.KeyRequired:             []string{askPropType},
		toolschema.KeyAdditionalProperties: false,
	}
}

func askFieldsSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "Only after a write tool refused for missing values: exactly " +
			"those values, as one form whose answers complete that one write. Never an " +
			"intake form up front, and never with options.",
		toolschema.KeyMaxItems: maxAskFields,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				askPropKey: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "The parameter this value fills, as the write tool " +
						"names it. \"pickupWindow\".",
				},
				askPropLabel: map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "How it reads to a person. \"Pickup window\".",
				},
				askPropType:       askValueTypeSchema(),
				askPropRecordKind: askRecordKindSchema(),
				"hint": map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "What a value should look like, shown in the box, " +
						"when the label does not say.",
				},
				"optional": map[string]any{
					toolschema.KeyType:        toolschema.TypeBoolean,
					toolschema.KeyDescription: "True when the write can go ahead without it.",
				},
			},
			toolschema.KeyRequired:             []string{askPropKey, askPropLabel, askPropType},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func askValueTypeSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyEnum: slices.Clone(askValueTypes),
		toolschema.KeyDescription: "text; number; date, a calendar day; datetime, a day and a " +
			"time; or record, picked from a search of one kind of record.",
	}
}

func askRecordKindSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "With type record, the kind of record to pick from a search: " +
			"customer or location.",
	}
}

// askOption is one choice, as the thread stores and renders it.
type askOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// askInput is the type of the value a person types instead of picking one.
// Plain text has none.
type askInput struct {
	Type       string `json:"type"`
	RecordKind string `json:"recordKind,omitempty"`
}

// askField is one value of a form, as the thread stores and renders it.
type askField struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	RecordKind string `json:"recordKind,omitempty"`
	Hint       string `json:"hint,omitempty"`
	Optional   bool   `json:"optional,omitempty"`
}

// askRequest is what the client renders and what the model gets back.
type askRequest struct {
	Question   string      `json:"question"`
	Options    []askOption `json:"options"`
	AllowOther bool        `json:"allowOther"`
	OtherHint  string      `json:"otherHint,omitempty"`
	Input      *askInput   `json:"input,omitempty"`
	Fields     []askField  `json:"fields,omitempty"`
	Note       string      `json:"note"`
}

// resolveAsk turns a model's ask_user call into the question the thread shows,
// and reports whether it was shown. One that was not is answered with what to
// do instead, and the person sees nothing.
//
// The runtime answers it rather than a registry tool for the same reason
// find_tools is answered here: its effect is on the conversation, not on data.
// Nothing is read, nothing is written, and no tenant is touched — which is also
// why it needs no permission of its own.
func resolveAsk(now time.Time, arguments map[string]any) (content string, shown bool) {
	question := strings.TrimSpace(stringArg(arguments, "question"))
	if question == "" {
		return "ask_user needs a question. Say what you need in one sentence, " +
			"with the choices you can offer.", false
	}

	request, refusal := askRequestFrom(now, question, arguments)
	if refusal != "" {
		return refusal, false
	}

	encoded, err := encodeToolResult(request, 0, "")
	if err != nil {
		return "The question could not be shown. Ask it in your reply instead.", false
	}

	return FenceToolResult(askUserName, encoded), true
}

// askRequestFrom builds the question the thread shows from a model's call, or
// says why it cannot be shown.
//
// A form stands alone: its answers complete one write, so it offers no choice
// beside it and no box of its own. A question answered only by typing in a
// plain box must ask for one value; one that lists several is a form the
// person would have to fill in prose.
func askRequestFrom(
	now time.Time,
	question string,
	arguments map[string]any,
) (request *askRequest, refusal string) {
	rawOptions := arrayArg(arguments, "options")
	if rawFields := arrayArg(arguments, "fields"); len(rawFields) > 0 {
		if len(rawOptions) > 0 {
			return nil, mixedAskRefusal
		}
		fields := askFieldsFrom(rawFields)
		if len(fields) == 0 {
			return nil, unshowableFieldsRefusal
		}
		request = &askRequest{Question: question, Options: []askOption{}, Fields: fields}
		request.Note = askNote(request)

		return request, ""
	}

	options := askOptionsFrom(rawOptions)
	past := datedOptionLabels(options, now)
	allowOther := boolArg(arguments, "allowOther", true)
	if len(options) == 0 && !allowOther {
		return nil, "ask_user needs either options to choose from or allowOther set, " +
			"or there is nothing the person can answer with."
	}

	var input *askInput
	if allowOther {
		input = askInputFrom(arguments)
	}
	if len(options) == 0 && input == nil && asksForSeveral(question) {
		return nil, multiPartAskRefusal
	}

	request = &askRequest{
		Question:   question,
		Options:    options,
		AllowOther: allowOther,
		OtherHint:  strings.TrimSpace(stringArg(arguments, "otherHint")),
		Input:      input,
	}
	request.Note = askNote(request)
	if len(past) > 0 && schedulingQuestion(question) {
		request.Note += pastDateNote(past)
	}

	return request, ""
}

// askNote tells the model the question is on screen and that its turn is over.
//
// Said plainly because the failure it prevents is specific: a model that asks
// and then answers itself has made the choice it just said was the person's.
// The turn ends in silence because the question is pinned above the person's
// message box; a line after it only repeated it, or told them to answer "the
// prompt above".
func askNote(request *askRequest) string {
	var b strings.Builder
	if fields := len(request.Fields); fields > 0 {
		fmt.Fprintf(&b, "The person has been shown a form with %d %s for this question, "+
			"pinned above their message box. Their answers arrive together as their next "+
			"message. ", fields, stringutils.Pluralize("field", "fields", fields))
	} else {
		b.WriteString("The person has been shown this question")
		if options := len(request.Options); options > 0 {
			fmt.Fprintf(&b, " with %d %s", options,
				stringutils.Pluralize("option", "options", options))
		}
		if request.AllowOther {
			b.WriteString(" and a box for their own value")
		}
		b.WriteString(", pinned above their message box. Their answer arrives as their ")
		b.WriteString("next message. ")
	}
	b.WriteString("End your turn now and write nothing more: the question is already ")
	b.WriteString("on screen, so anything you add only repeats it. ")
	b.WriteString("Do not choose for them, do not assume a default, and do not call ")
	b.WriteString("another tool that needs what you just asked for.")

	return b.String()
}

// askOptionsFrom reads the options a model supplied, dropping what cannot be
// rendered. A malformed option is skipped rather than failing the call: a
// question with three usable choices out of four is still worth asking. None
// is an empty list, never null, so the thread reads one shape.
//
// An option with a label and no value answers with its label. A model offered
// three choices as label and detail, and the person was shown none of them.
func askOptionsFrom(raw []any) []askOption {
	options := make([]askOption, 0, min(len(raw), maxAskOptions))
	seen := make(map[string]struct{}, len(raw))
	for _, entry := range raw {
		if len(options) == maxAskOptions {
			break
		}

		fields, fieldsOk := entry.(map[string]any)
		if !fieldsOk {
			continue
		}

		value := strings.TrimSpace(stringArg(fields, "value"))
		label := strings.TrimSpace(stringArg(fields, askPropLabel))
		switch {
		case value == "" && label == "":
			continue
		case value == "":
			value = label
		case label == "":
			label = value
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}

		options = append(options, askOption{
			Value: value,
			Label: truncateRunes(label, maxAskLabelChars),
			Detail: truncateRunes(
				strings.TrimSpace(stringArg(fields, "detail")),
				maxAskLabelChars*2,
			),
		})
	}

	return options
}

// askFieldsFrom reads the fields of a form, dropping what cannot be rendered
// as options are dropped: a field with no key or label, a type the thread has
// no box for, a record field whose kind is not one the registry knows, and a
// key already taken.
func askFieldsFrom(raw []any) []askField {
	fields := make([]askField, 0, min(len(raw), maxAskFields))
	seen := make(map[string]struct{}, len(raw))
	for _, entry := range raw {
		if len(fields) == maxAskFields {
			break
		}

		spec, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		key := strings.TrimSpace(stringArg(spec, askPropKey))
		label := strings.TrimSpace(stringArg(spec, askPropLabel))
		if key == "" || label == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		valueType, recordKind, typed := askValueType(spec)
		if !typed {
			continue
		}
		seen[key] = struct{}{}

		fields = append(fields, askField{
			Key:        key,
			Label:      truncateRunes(label, maxAskLabelChars),
			Type:       valueType,
			RecordKind: recordKind,
			Hint: truncateRunes(
				strings.TrimSpace(stringArg(spec, "hint")),
				maxAskLabelChars*2,
			),
			Optional: boolArg(spec, "optional", false),
		})
	}

	return fields
}

// askInputFrom reads the type of the person's own value. Plain text, a type
// the thread has no box for and a record of no kind the registry knows all
// leave it out, and the person types text.
func askInputFrom(arguments map[string]any) *askInput {
	spec, ok := arguments["input"].(map[string]any)
	if !ok {
		return nil
	}

	valueType, recordKind, typed := askValueType(spec)
	if !typed || valueType == askTypeText {
		return nil
	}

	return &askInput{Type: valueType, RecordKind: recordKind}
}

// askValueType reads a type and, for a record, its kind. The kind is held to
// the permission registry, as a page or a mention naming a record is, so a
// box never searches for a kind of record that does not exist.
func askValueType(spec map[string]any) (valueType, recordKind string, ok bool) {
	valueType = strings.ToLower(strings.TrimSpace(stringArg(spec, askPropType)))
	if !slices.Contains(askValueTypes, valueType) {
		return "", "", false
	}
	if valueType != askTypeRecord {
		return valueType, "", true
	}

	recordKind = strings.ToLower(strings.TrimSpace(stringArg(spec, askPropRecordKind)))
	if !agent.IsKnownPageEntityType(recordKind) {
		return "", "", false
	}

	return valueType, recordKind, true
}

// asksForSeveral reports whether a question answered only by typing asks for
// more than one value. It holds the question to two rules, either of which
// is enough: it runs past maxOpenQuestionRunes, or it lists what it wants,
// which is more than maxOpenQuestionSeparators commas and semicolons, or one
// of them followed later by "and". A comma between digits, as in "2,450",
// separates nothing.
//
// Both rules err towards letting a question through. A short question that
// asks for two things joined by a bare "and" is still shown; what is refused
// is the list.
func asksForSeveral(question string) bool {
	if utf8.RuneCountInString(question) > maxOpenQuestionRunes {
		return true
	}

	separators, rest := 0, -1
	for idx := range len(question) {
		char := question[idx]
		if char != ',' && char != ';' {
			continue
		}
		if char == ',' && idx > 0 && idx+1 < len(question) &&
			unicode.IsDigit(rune(question[idx-1])) && unicode.IsDigit(rune(question[idx+1])) {
			continue
		}
		separators++
		if rest < 0 {
			rest = idx + 1
		}
	}

	if separators > maxOpenQuestionSeparators {
		return true
	}

	return rest >= 0 && listConjunctionPattern.MatchString(question[rest:])
}

func stringArg(arguments map[string]any, key string) string {
	value, _ := arguments[key].(string)

	return value
}

func arrayArg(arguments map[string]any, key string) []any {
	value, _ := arguments[key].([]any)

	return value
}

func boolArg(arguments map[string]any, key string, fallback bool) bool {
	value, ok := arguments[key].(bool)
	if !ok {
		return fallback
	}

	return value
}

// truncateRunes cuts on a rune boundary, so a label is never invalid UTF-8.
func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return strings.TrimSpace(string(runes[:limit])) + "…"
}

const (
	optionDayLayout      = "Mon Jan 2"
	optionDayYearLayout  = "Mon Jan 2, 2006"
	optionClockSeparator = ", "
)

var (
	relativeDayPattern = regexp.MustCompile(
		`(?i)\b(today|tomorrow|yesterday|in \d+ days?|\d+ days? ago)\b`,
	)
	listConjunctionPattern = regexp.MustCompile(`(?i)\band\b`)
	leadingWeekdayPattern  = regexp.MustCompile(
		`(?i)\b(?:mon|tue|tues|wed|thu|thur|thurs|fri|sat|sun)(?:day|nesday|rsday|urday)?\.?,?\s*$`,
	)
	schedulingWords = []string{
		"schedul", "appointment", "appt", "pickup", "pick up", "pick-up", "deliver",
		"arriv", "depart", "dispatch", "book", "when should", "when will", "when do",
		"when can", "ship date", "load date",
	}
)

func datedOptionLabels(options []askOption, now time.Time) []string {
	var past []string
	for idx := range options {
		label, days, dated := humanizeDateLabel(options[idx].Label, now)
		if !dated {
			continue
		}
		options[idx].Label = truncateRunes(label, maxAskLabelChars)
		if days < 0 {
			past = append(past, label)
		}
	}

	return past
}

func humanizeDateLabel(label string, now time.Time) (humanized string, days int64, dated bool) {
	written, found := timeutils.FindWrittenDate(label)
	if !found {
		return label, 0, false
	}

	day := written.Day.On(now)
	days = timeutils.CalendarDaysBetween(now, day)
	if relativeDayPattern.MatchString(label) {
		return label, days, true
	}

	layout := optionDayLayout
	if day.Year() != now.Year() {
		layout = optionDayYearLayout
	}
	formatted := day.Format(layout)
	if written.Clock != "" {
		formatted += optionClockSeparator + written.Clock
	}

	start := written.Start
	if weekday := leadingWeekdayPattern.FindStringIndex(label[:start]); weekday != nil {
		start = weekday[0]
	}

	humanized = strings.TrimSpace(label[:start] + formatted + label[written.End:])

	return humanized + " (" + timeutils.RelativeDays(days) + ")", days, true
}

func schedulingQuestion(question string) bool {
	lowered := strings.ToLower(question)
	for _, word := range schedulingWords {
		if strings.Contains(lowered, word) {
			return true
		}
	}

	return false
}

func pastDateNote(past []string) string {
	return fmt.Sprintf(" %s %s in the past. If that is not what you meant, say so in one "+
		"short line instead of ending in silence; do not schedule anything for a day that "+
		"has already gone.", strings.Join(past, ", "), stringutils.Pluralize("is", "are", len(past)))
}
