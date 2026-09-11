package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

type pluralForm string

const (
	pluralOne   pluralForm = "one"
	pluralOther pluralForm = "other"
)

func selectPlural(locale Locale, n float64) pluralForm {
	switch locale {
	case ZhTW, ZhCN:
		return pluralOther
	case EN, ES:
		if n == 1 {
			return pluralOne
		}
		return pluralOther
	default:
		if n == 1 {
			return pluralOne
		}
		return pluralOther
	}
}

func format(locale Locale, message string, args []any) string {
	if len(args) == 0 || !strings.ContainsRune(message, '{') {
		return message
	}

	var out strings.Builder
	out.Grow(len(message) + 16*len(args))

	for i := 0; i < len(message); {
		if message[i] != '{' {
			out.WriteByte(message[i])
			i++
			continue
		}

		end, ok := matchBrace(message, i)
		if !ok {
			out.WriteByte(message[i])
			i++
			continue
		}

		rendered, ok := renderPlaceholder(locale, message[i+1:end], args)
		if !ok {
			out.WriteString(message[i : end+1])
		} else {
			out.WriteString(rendered)
		}
		i = end + 1
	}

	return out.String()
}

func matchBrace(s string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func renderPlaceholder(locale Locale, body string, args []any) (string, bool) {
	indexPart, rest, hasRest := strings.Cut(body, ",")

	index, err := strconv.Atoi(strings.TrimSpace(indexPart))
	if err != nil || index < 0 || index >= len(args) {
		return "", false
	}

	value := args[index]
	if !hasRest {
		return stringify(value), true
	}

	kind, forms, hasForms := strings.Cut(strings.TrimSpace(rest), ",")
	if !hasForms || strings.TrimSpace(kind) != "plural" {
		return stringify(value), true
	}

	number, ok := toFloat(value)
	if !ok {
		return stringify(value), true
	}

	chosen, ok := pluralBranch(forms, selectPlural(locale, number))
	if !ok {
		return stringify(value), true
	}

	return strings.ReplaceAll(chosen, "#", stringify(value)), true
}

func pluralBranch(forms string, want pluralForm) (string, bool) {
	var fallback string
	var haveFallback bool

	for i := 0; i < len(forms); {
		if forms[i] == ' ' || forms[i] == '\t' || forms[i] == '\n' {
			i++
			continue
		}

		nameEnd := strings.IndexByte(forms[i:], '{')
		if nameEnd < 0 {
			break
		}

		name := pluralForm(strings.TrimSpace(forms[i : i+nameEnd]))
		open := i + nameEnd

		end, ok := matchBrace(forms, open)
		if !ok {
			break
		}

		body := forms[open+1 : end]
		if name == want {
			return body, true
		}
		if name == pluralOther {
			fallback = body
			haveFallback = true
		}

		i = end + 1
	}

	return fallback, haveFallback
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}
