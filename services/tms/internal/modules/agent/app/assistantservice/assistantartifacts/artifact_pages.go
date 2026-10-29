package assistantartifacts

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

// maxTurnRunes bounds the question an artifact is labelled with: a line in
// the pane, not the question again.
const maxTurnRunes = 80

// turnLabel is the first line of a question, cut to a label.
func turnLabel(question string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(question), "\n")
	line = strings.TrimSpace(line)
	if utf8.RuneCountInString(line) <= maxTurnRunes {
		return line
	}

	return strings.TrimSpace(stringutils.TruncateRunes(line, maxTurnRunes-1)) + "…"
}
