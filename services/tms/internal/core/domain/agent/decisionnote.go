package agent

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const MaxDecisionNoteLength = 2000

func NormalizeDecisionNote(note string) string {
	return strings.TrimSpace(note)
}

func CheckDecisionNote(note string) error {
	if utf8.RuneCountInString(note) <= MaxDecisionNoteLength {
		return nil
	}

	return errortypes.NewValidationError(
		"note",
		errortypes.ErrInvalid,
		"A note to the agent can be at most {0} characters",
		MaxDecisionNoteLength,
	)
}
