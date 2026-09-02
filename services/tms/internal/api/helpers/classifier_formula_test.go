package helpers_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	formulaerrors "github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClassifier_FormulaErrorsAreValidationProblems(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	cases := map[string]error{
		"schema": formulaerrors.NewSchemaError(
			"x * y",
			"compile",
			errors.New("mismatched types"),
		),
		"compute": formulaerrors.NewComputeError(
			"x / 0",
			"expression",
			errors.New("division by zero"),
		),
		"transform": formulaerrors.NewTransformError(
			"result",
			"decimal",
			"abc",
			errors.New("nope"),
		),
		"missing": formulaerrors.NewMissingFieldError(
			"weight * 2",
			[]string{"weight"},
			[]string{"coalesce(weight, 0)"},
		),
		"wrapped": fmt.Errorf(
			"rating: %w",
			formulaerrors.NewComputeError("x", "expression", errors.New("boom")),
		),
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, helpers.ProblemTypeValidation, classifier.Classify(err))
		})
	}
}

func TestDefaultClassifier_CallerDeadlineStaysATimeout(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	assert.Equal(t, helpers.ProblemTypeTimeout, classifier.Classify(context.DeadlineExceeded))
	assert.Equal(t,
		helpers.ProblemTypeValidation,
		classifier.Classify(formulaerrors.NewComputeError(
			"slow()", "expression", errors.New("formula evaluation exceeded its time budget"),
		)),
	)
}
