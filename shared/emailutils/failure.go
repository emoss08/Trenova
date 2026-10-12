package emailutils

import (
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
)

const maxFailureTextRunes = 500

var providerMessageKeys = []string{"Message", "message", "error", "detail", "description"}

func ProviderFailureText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}

	if start := strings.IndexByte(text, '{'); start >= 0 {
		if message := providerMessage(text[start:]); message != "" {
			return CleanFailureText(message)
		}
		text = strings.TrimSpace(text[:start])
	}

	if idx := strings.LastIndex(text, ": "); idx >= 0 && idx+2 < len(text) {
		text = text[idx+2:]
	}

	return CleanFailureText(text)
}

func providerMessage(body string) string {
	end := strings.LastIndexByte(body, '}')
	if end < 0 {
		return ""
	}

	var payload map[string]any
	if err := sonic.UnmarshalString(body[:end+1], &payload); err != nil {
		return ""
	}

	for _, key := range providerMessageKeys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func CleanFailureText(text string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, text)
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	runes := []rune(cleaned)
	if len(runes) > maxFailureTextRunes {
		return string(runes[:maxFailureTextRunes])
	}

	return cleaned
}
