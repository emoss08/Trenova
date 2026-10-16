package errortypes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalizedTextFillsTheMessageArguments(t *testing.T) {
	err := NewBusinessError("The {0} falls in a closed fiscal period", "manual journal")

	assert.Equal(t, "The manual journal falls in a closed fiscal period", LocalizedText(t.Context(), err))
}

func TestLocalizedTextFindsAWrappedError(t *testing.T) {
	err := errors.Join(errors.New("posting"), NewBusinessError("Journal entry {0} is already posted", "JE-1"))

	assert.Equal(t, "Journal entry JE-1 is already posted", LocalizedText(t.Context(), err))
}

func TestLocalizedTextReadsAPlainErrorAsWritten(t *testing.T) {
	assert.Equal(t, "connection reset", LocalizedText(t.Context(), errors.New("connection reset")))
	assert.Empty(t, LocalizedText(t.Context(), nil))
}
