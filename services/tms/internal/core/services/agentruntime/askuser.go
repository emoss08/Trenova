package agentruntime

import (
	"fmt"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	askUserName = "ask_user"

	// maxAskOptions bounds one question. A list longer than this is a search,
	// not a choice, and a person scanning it will read none of it.
	maxAskOptions = 8
	// maxAskLabelChars keeps an option to something readable on a chip.
	maxAskLabelChars = 60
)

// askUserDescription is a constant for the same reason findToolsDescription is:
// the i18n extractor harvests Description: fields, and this text is addressed to
// a model, so translating it would change what the assistant is told based on
// the operator's locale.
const askUserDescription = "Ask the person to choose a value you need and cannot " +
	"safely infer. Use it when a required parameter is missing, when a request is " +
	"ambiguous between a few readings, or when picking wrongly would do work nobody " +
	"asked for. The options are shown as buttons, and the person can also type their " +
	"own value unless you turn that off. Prefer inferring from what they already " +
	"said — a request naming a window, a date range or a customer has answered the " +
	"question already — and ask only for what is genuinely missing. Ask once, for " +
	"one thing, then stop and wait: never guess a value after asking for it, and " +
	"never ask for something the person has already told you."

// unattendedAskRefusal answers ask_user in a run nobody is watching. The spec
// is withheld from those runs, but a model that remembers the tool from its
// instructions can still name it, and a question no one will see used to end
// the run as though it had been answered.
const unattendedAskRefusal = "Nobody is watching this run, so there is no one to ask. " +
	"Decide as your instructions allow, or call raise_exception to hand the question " +
	"to a person."

// askUserSpec is the second tool the runtime answers itself.
//
// Every turn carries it, disclosed or not, because asking for a missing value is
// part of holding a conversation rather than a capability an organization grants.
// An agent that may read shipments but may not ask which shipment is an agent
// that has to guess, and guessing is what this exists to stop.
func askUserSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        askUserName,
		Description: askUserDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type": "string",
					"description": "What you need, as one plain question. " +
						"\"Which window should the report cover?\"",
				},
				"options": map[string]any{
					"type": "array",
					"description": "The choices, best first. Where the value is " +
						"constrained — a parameter with allowed values, a status, a " +
						"category — these must be the real ones, not examples.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"value": map[string]any{
								"type": "string",
								"description": "What you will use if this is picked, " +
									"exactly as the tool needs it.",
							},
							"label": map[string]any{
								"type":        "string",
								"description": "How it reads to a person. \"30 days\".",
							},
							"detail": map[string]any{
								"type": "string",
								"description": "One short clause of consequence, when " +
									"the label does not carry it.",
							},
						},
						"required":             []string{"value", "label"},
						"additionalProperties": false,
					},
				},
				"allowOther": map[string]any{
					"type": "boolean",
					"description": "Whether the person may type a value instead of " +
						"picking one. True unless the value is genuinely closed, such as " +
						"a fixed set of statuses.",
				},
				"otherHint": map[string]any{
					"type": "string",
					"description": "What a typed value should look like, shown in the " +
						"box. \"Number of days\".",
				},
			},
			"required":             []string{"question", "options"},
			"additionalProperties": false,
		},
	}
}

// askOption is one choice, as the thread stores and renders it.
type askOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// askRequest is what the client renders and what the model gets back.
type askRequest struct {
	Question   string      `json:"question"`
	Options    []askOption `json:"options"`
	AllowOther bool        `json:"allowOther"`
	OtherHint  string      `json:"otherHint,omitempty"`
	Note       string      `json:"note"`
}

// resolveAsk turns a model's ask_user call into the question the thread shows.
//
// The runtime answers it rather than a registry tool for the same reason
// find_tools is answered here: its effect is on the conversation, not on data.
// Nothing is read, nothing is written, and no tenant is touched — which is also
// why it needs no permission of its own.
func resolveAsk(arguments map[string]any) string {
	question := strings.TrimSpace(stringArg(arguments, "question"))
	if question == "" {
		return "ask_user needs a question. Say what you need in one sentence, " +
			"with the choices you can offer."
	}

	options := askOptionsFrom(arguments)
	allowOther := boolArg(arguments, "allowOther", true)

	if len(options) == 0 && !allowOther {
		return "ask_user needs either options to choose from or allowOther set, " +
			"or there is nothing the person can answer with."
	}

	request := askRequest{
		Question:   question,
		Options:    options,
		AllowOther: allowOther,
		OtherHint:  strings.TrimSpace(stringArg(arguments, "otherHint")),
	}
	request.Note = askNote(len(options), allowOther)

	encoded, err := encodeToolResult(request, 0, "")
	if err != nil {
		return "The question could not be shown. Ask it in your reply instead."
	}

	return FenceToolResult(askUserName, encoded)
}

// askNote tells the model the question is on screen and that its turn is over.
//
// Said plainly because the failure it prevents is specific: a model that asks
// and then answers itself has made the choice it just said was the person's.
func askNote(optionCount int, allowOther bool) string {
	var b strings.Builder
	b.WriteString("The person has been shown this question")
	if optionCount > 0 {
		fmt.Fprintf(&b, " with %d option", optionCount)
		if optionCount != 1 {
			b.WriteString("s")
		}
	}
	if allowOther {
		b.WriteString(" and a box for their own value")
	}
	b.WriteString(". Their answer arrives as their next message. ")
	b.WriteString("End your turn now with at most one short line — the question is ")
	b.WriteString("already on screen, so do not repeat it or list the options again. ")
	b.WriteString("Do not choose for them, do not assume a default, and do not call ")
	b.WriteString("another tool that needs the value you just asked for.")

	return b.String()
}

// askOptionsFrom reads the options a model supplied, dropping what cannot be
// rendered. A malformed option is skipped rather than failing the call: a
// question with three usable choices out of four is still worth asking.
func askOptionsFrom(arguments map[string]any) []askOption {
	raw, ok := arguments["options"].([]any)
	if !ok {
		return nil
	}

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
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}

		label := strings.TrimSpace(stringArg(fields, "label"))
		if label == "" {
			label = value
		}

		options = append(options, askOption{
			Value:  value,
			Label:  truncateRunes(label, maxAskLabelChars),
			Detail: truncateRunes(strings.TrimSpace(stringArg(fields, "detail")), maxAskLabelChars*2),
		})
	}

	return options
}

func stringArg(arguments map[string]any, key string) string {
	value, _ := arguments[key].(string)

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
