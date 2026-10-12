package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Models send numbers as strings often enough — "days": "30", "windowDays":
// "90" — that a reader which dropped the string form turned valid requests
// into a validation error the model then retried identically.
func TestOptionalInt_ReadsANumberSentAsAString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30, optionalInt(map[string]any{"days": "30"}, "days", 0))
	assert.Equal(t, 30, optionalInt(map[string]any{"days": " 30 "}, "days", 0))
	assert.Equal(t, 30, optionalInt(map[string]any{"days": float64(30)}, "days", 0))
}

func TestOptionalInt_FallsBackOnAStringThatIsNotANumber(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 7, optionalInt(map[string]any{"days": "thirty"}, "days", 7))
	assert.Equal(t, 7, optionalInt(map[string]any{"days": ""}, "days", 7))
	assert.Equal(t, 7, optionalInt(map[string]any{}, "days", 7))
}
