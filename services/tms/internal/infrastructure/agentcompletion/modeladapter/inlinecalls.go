package modeladapter

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	toolCallOpen      = "<tool_call>"
	toolCallClose     = "</tool_call>"
	argKeyOpen        = "<arg_key>"
	argKeyClose       = "</arg_key>"
	argValueOpen      = "<arg_value>"
	argValueClose     = "</arg_value>"
	minPartialOpenTag = len("<tool")
)

var (
	inlineToolName    = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)
	inlineJSONName    = regexp.MustCompile(`"name"\s*:\s*"([^"\\]+)"`)
	errInlineArgsCut  = errors.New("the call's arguments stop partway")
	errInlineArgsType = errors.New("the call's arguments are not an object")
)

type liftedCalls struct {
	text   string
	calls  []ToolCall
	cutOff *CutOffToolCall
}

type textSpan struct {
	start int
	end   int
}

func liftInlineToolCalls(text string, tools []ToolSpec, first int) liftedCalls {
	if !strings.Contains(text, "<tool") {
		return liftedCalls{text: text}
	}

	quoted := quotedSpans(text)
	lifted := liftedCalls{}
	var kept strings.Builder
	kept.Grow(len(text))
	changed := false
	cursor := 0

	for {
		start := nextUnquoted(text, toolCallOpen, cursor, quoted)
		if start < 0 {
			break
		}
		changed = true
		kept.WriteString(text[cursor:start])
		bodyStart := start + len(toolCallOpen)
		end := strings.Index(text[bodyStart:], toolCallClose)
		if end < 0 {
			lifted.cutOff = &CutOffToolCall{Name: partialCallName(text[bodyStart:])}
			cursor = len(text)

			break
		}

		if call, ok := parseInlineCall(text[bodyStart:bodyStart+end], tools); ok {
			call.ID = "call_" + strconv.Itoa(first+len(lifted.calls))
			call.SynthesizedID = true
			lifted.calls = append(lifted.calls, call)
		}
		cursor = bodyStart + end + len(toolCallClose)
	}

	rest := text[cursor:]
	if lifted.cutOff == nil {
		if cut := partialOpenTag(text, cursor, quoted); cut >= 0 {
			changed = true
			rest = text[cursor:cut]
			lifted.cutOff = &CutOffToolCall{}
		}
	}
	if !changed {
		return liftedCalls{text: text}
	}

	kept.WriteString(rest)
	lifted.text = strings.TrimSpace(kept.String())

	return lifted
}

func appendToolCalls(native, lifted []ToolCall) []ToolCall {
	if len(lifted) == 0 {
		return native
	}

	return append(native, lifted...)
}

func partialOpenTag(text string, from int, quoted []textSpan) int {
	for size := min(len(toolCallOpen)-1, len(text)-from); size >= minPartialOpenTag; size-- {
		start := len(text) - size
		if text[start:] == toolCallOpen[:size] && !inSpan(start, quoted) {
			return start
		}
	}

	return -1
}

func nextUnquoted(text, needle string, from int, quoted []textSpan) int {
	for from <= len(text) {
		idx := strings.Index(text[from:], needle)
		if idx < 0 {
			return -1
		}
		at := from + idx
		if !inSpan(at, quoted) {
			return at
		}
		from = at + len(needle)
	}

	return -1
}

func inSpan(at int, spans []textSpan) bool {
	for _, span := range spans {
		if at >= span.start && at < span.end {
			return true
		}
	}

	return false
}

func quotedSpans(text string) []textSpan {
	var spans []textSpan
	fence := ""
	fenceStart := 0
	lineStart := 0

	for lineStart <= len(text) {
		lineEnd := strings.IndexByte(text[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(text)
		} else {
			lineEnd += lineStart
		}
		line := text[lineStart:lineEnd]
		marker := fenceMarker(line)

		switch {
		case fence == "" && marker != "":
			fence = marker
			fenceStart = lineStart
		case fence != "" && strings.HasPrefix(strings.TrimSpace(line), fence) &&
			strings.Trim(strings.TrimSpace(line), fence[:1]) == "":
			spans = append(spans, textSpan{start: fenceStart, end: lineEnd})
			fence = ""
		case fence == "":
			spans = append(spans, codeSpans(line, lineStart)...)
		}

		if lineEnd == len(text) {
			break
		}
		lineStart = lineEnd + 1
	}

	if fence != "" {
		spans = append(spans, textSpan{start: fenceStart, end: len(text)})
	}

	return spans
}

func fenceMarker(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return ""
	}
	for _, char := range []byte{'`', '~'} {
		run := 0
		for run < len(trimmed) && trimmed[run] == char {
			run++
		}
		if run >= 3 {
			return trimmed[:run]
		}
	}

	return ""
}

func codeSpans(line string, offset int) []textSpan {
	var spans []textSpan
	for idx := 0; idx < len(line); {
		if line[idx] != '`' {
			idx++
			continue
		}
		run := backtickRun(line, idx)
		closing := closingRun(line, idx+run, run)
		if closing < 0 {
			idx += run
			continue
		}
		spans = append(spans, textSpan{start: offset + idx, end: offset + closing + run})
		idx = closing + run
	}

	return spans
}

func backtickRun(line string, at int) int {
	run := 0
	for at+run < len(line) && line[at+run] == '`' {
		run++
	}

	return run
}

func closingRun(line string, from, size int) int {
	for idx := from; idx < len(line); {
		if line[idx] != '`' {
			idx++
			continue
		}
		run := backtickRun(line, idx)
		if run == size {
			return idx
		}
		idx += run
	}

	return -1
}

func parseInlineCall(body string, tools []ToolSpec) (ToolCall, bool) {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") {
		return parseJSONInlineCall(trimmed)
	}

	return parseTaggedInlineCall(trimmed, tools)
}

func parseJSONInlineCall(body string) (ToolCall, bool) {
	var payload struct {
		Name       string `json:"name"`
		Arguments  any    `json:"arguments"`
		Parameters any    `json:"parameters"`
	}
	if err := sonic.UnmarshalString(body, &payload); err != nil {
		name := partialCallName(body)
		if name == "" {
			return ToolCall{}, false
		}

		return ToolCall{Name: name, Arguments: map[string]any{}, ArgumentsError: err.Error()}, true
	}
	if !inlineToolName.MatchString(payload.Name) {
		return ToolCall{}, false
	}

	raw := payload.Arguments
	if raw == nil {
		raw = payload.Parameters
	}
	call := ToolCall{Name: payload.Name, Arguments: map[string]any{}}
	switch arguments := raw.(type) {
	case nil:
	case map[string]any:
		call.Arguments = arguments
	case string:
		call.Arguments, call.ArgumentsError = decodeArguments(arguments)
	default:
		call.ArgumentsError = errInlineArgsType.Error()
	}

	return call, true
}

func parseTaggedInlineCall(body string, tools []ToolSpec) (ToolCall, bool) {
	name := body
	rest := ""
	if idx := strings.Index(body, argKeyOpen); idx >= 0 {
		name, rest = body[:idx], body[idx:]
	}
	name = strings.TrimSpace(name)
	if !inlineToolName.MatchString(name) {
		return ToolCall{}, false
	}

	properties := toolProperties(tools, name)
	call := ToolCall{Name: name, Arguments: make(map[string]any)}
	for strings.TrimSpace(rest) != "" {
		key, value, remaining, ok := nextTaggedArgument(rest)
		if !ok {
			call.Arguments = map[string]any{}
			call.ArgumentsError = errInlineArgsCut.Error()

			return call, true
		}
		property, _ := properties[key].(map[string]any)
		call.Arguments[key] = inlineArgumentValue(value, property)
		rest = remaining
	}

	return call, true
}

func nextTaggedArgument(text string) (key, value, rest string, ok bool) {
	keyText, afterKey, found := between(strings.TrimSpace(text), argKeyOpen, argKeyClose)
	if !found {
		return "", "", "", false
	}
	valueText, afterValue, found := between(
		strings.TrimSpace(afterKey),
		argValueOpen,
		argValueClose,
	)
	if !found {
		return "", "", "", false
	}
	key = strings.TrimSpace(keyText)
	if key == "" {
		return "", "", "", false
	}

	return key, valueText, afterValue, true
}

func between(text, open, closing string) (inner, rest string, ok bool) {
	if !strings.HasPrefix(text, open) {
		return "", "", false
	}
	body := text[len(open):]
	end := strings.Index(body, closing)
	if end < 0 {
		return "", "", false
	}

	return body[:end], body[end+len(closing):], true
}

func toolProperties(tools []ToolSpec, name string) map[string]any {
	for idx := range tools {
		if tools[idx].Name != name {
			continue
		}
		properties, _ := tools[idx].Parameters["properties"].(map[string]any)

		return properties
	}

	return nil
}

func inlineArgumentValue(raw string, property map[string]any) any {
	value := strings.TrimSpace(raw)
	if kind, _ := property["type"].(string); kind == "string" {
		return value
	}

	var decoded any
	if err := sonic.UnmarshalString(value, &decoded); err == nil {
		return decoded
	}

	return value
}

func partialCallName(body string) string {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") {
		if match := inlineJSONName.FindStringSubmatch(trimmed); len(match) == 2 &&
			inlineToolName.MatchString(match[1]) {
			return match[1]
		}

		return ""
	}

	end := strings.Index(trimmed, argKeyOpen)
	if end < 0 {
		end = strings.IndexAny(trimmed, "\n<")
	}
	if end < 0 {
		return ""
	}
	name := strings.TrimSpace(trimmed[:end])
	if !inlineToolName.MatchString(name) {
		return ""
	}

	return name
}
