package helpers_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestEditConflictIsAConflictCarryingWhatChanged(t *testing.T) {
	t.Parallel()

	edit := &errortypes.EditConflict{
		Version:       5,
		UpdatedByName: "Sarah Alvarez",
		UpdatedAt:     1_800_000_500,
		Changes:       []errortypes.EditConflictChange{{Field: "instructions", Label: "Instructions"}},
	}
	err := errortypes.NewEditConflictError(edit)

	problemType := helpers.NewDefaultClassifier().Classify(err)
	assert.Equal(t, helpers.ProblemTypeConflict, problemType)
	assert.Equal(t, http.StatusConflict, problemType.Info().StatusCode)
	assert.Same(t, edit, helpers.NewSanitizer(false).ExtractConflict(err))

	problem := helpers.NewProblemBuilder("https://example.test/").
		WithType(problemType).
		WithConflict(edit).
		Build()
	assert.Same(t, edit, problem.Conflict)
}
