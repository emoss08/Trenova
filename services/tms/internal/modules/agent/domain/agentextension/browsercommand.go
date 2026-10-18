package agentextension

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const BrowserCommand = "/browser"

func BrowserTask(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < len(BrowserCommand) ||
		!strings.EqualFold(trimmed[:len(BrowserCommand)], BrowserCommand) {
		return "", false
	}

	rest := trimmed[len(BrowserCommand):]
	if rest == "" {
		return "", true
	}
	next, _ := utf8.DecodeRuneInString(rest)
	if !unicode.IsSpace(next) {
		return "", false
	}

	return strings.TrimSpace(rest), true
}

func WorksInBrowser(tools []string) bool {
	return slices.Contains(tools, ToolReadPortal) || slices.Contains(tools, ToolActOnPortal)
}
