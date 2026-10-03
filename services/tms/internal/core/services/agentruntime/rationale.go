package agentruntime

import (
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

/*
Why a step was taken, from the model that took it.

"Why this step?" under a footnote shows three short lines: what the agent
looked at, why it chose this step, and what it passed over. Only the model
knows those, and it knows them when it makes the call, so every tool it is
offered takes one more optional argument for them. The runtime lifts it off
the call as soon as the reply arrives: the tool never sees it, the arguments
that identify a call (for repeats, idempotency and approval) are what they
would have been without it, and the call carries it alongside to the thread
and the stream.

The argument is named like the runtime's other own parameter, _owner, so no
tool's parameter can collide with it.
*/

// StepRationaleParam is the argument a step's rationale travels in.
const StepRationaleParam = "_why"

// maxRationaleRunes bounds each line: a phrase, not a paragraph.
const maxRationaleRunes = 200

var rationaleSchema = map[string]any{
	"type": "object",
	"description": "Optional. Why you are taking this step, for the person reading " +
		"the conversation, in a short phrase each: what you looked at, why this step, " +
		"and what you chose not to do instead. Leave out what you have nothing to say about.",
	"properties": map[string]any{
		"saw": map[string]any{
			"type":        "string",
			"description": "What you looked at that led here, e.g. \"8 items have no biller\".",
			"maxLength":   maxRationaleRunes,
		},
		"because": map[string]any{
			"type":        "string",
			"description": "The reason or rule behind this step.",
			"maxLength":   maxRationaleRunes,
		},
		"insteadOf": map[string]any{
			"type":        "string",
			"description": "The alternative you passed over, and why.",
			"maxLength":   maxRationaleRunes,
		},
	},
}

// withRationale offers the rationale argument on every tool. The specs are
// copied: the toolset's own stay as the tools declare them.
func withRationale(specs []serviceports.ToolSpec) []serviceports.ToolSpec {
	if len(specs) == 0 {
		return specs
	}

	out := make([]serviceports.ToolSpec, len(specs))
	for idx, spec := range specs {
		out[idx] = spec
		properties, _ := spec.Parameters["properties"].(map[string]any)
		if _, taken := properties[StepRationaleParam]; taken {
			continue
		}
		parameters := maps.Clone(spec.Parameters)
		if parameters == nil {
			parameters = map[string]any{"type": "object"}
		}
		withWhy := maps.Clone(properties)
		if withWhy == nil {
			withWhy = make(map[string]any, 1)
		}
		withWhy[StepRationaleParam] = rationaleSchema
		parameters["properties"] = withWhy
		out[idx].Parameters = parameters
	}

	return out
}

// liftRationales takes each call's rationale out of its arguments and onto
// the call. The arguments are copied, never edited in place.
func liftRationales(calls []serviceports.ToolCall) {
	for idx := range calls {
		raw, present := calls[idx].Arguments[StepRationaleParam]
		if !present {
			continue
		}
		arguments := maps.Clone(calls[idx].Arguments)
		delete(arguments, StepRationaleParam)
		calls[idx].Arguments = arguments
		calls[idx].Why = rationaleOf(raw)
	}
}

func rationaleOf(raw any) *conversation.StepRationale {
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	line := func(key string) string {
		text, _ := fields[key].(string)
		text = strings.Join(strings.Fields(text), " ")
		if runes := []rune(text); len(runes) > maxRationaleRunes {
			text = string(runes[:maxRationaleRunes-1]) + "…"
		}

		return text
	}
	why := &conversation.StepRationale{
		Saw:       line("saw"),
		Because:   line("because"),
		InsteadOf: line("insteadOf"),
	}
	if why.Saw == "" && why.Because == "" && why.InsteadOf == "" {
		return nil
	}

	return why
}
