package helpers_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/pkg/rules"
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
		"evaluation": formulaerrors.NewEvaluationError("expression", &rules.FormulaError{
			Code:    "missingField",
			Message: rules.Message{Template: "no value for {0}", Args: []any{"weight"}},
			Fields:  []string{"weight"},
		}),
		"unresolved": formulaerrors.NewUnresolvedFieldError(
			[]string{"serviceType.code"},
			map[string]error{"serviceType.code": errors.New("nil pointer")},
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
