package helpers

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/pkg/wasmmodule"

	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyTariff_TenantAuthoredFaultsAreValidationProblems(t *testing.T) {
	t.Parallel()

	classifier := NewDefaultClassifier()
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "an evaluation error returned bare",
			err: fmt.Errorf("rate: %w", &tarifferrors.EvaluationError{
				Code:    "overflow",
				Message: wasmmodule.Message{Template: "The amount is too large"},
			}),
		},
		{
			name: "an evaluation error as a validation error",
			err: tarifferrors.Evaluation("source", &rules.TariffEvalError{
				Code:    "divideByZero",
				Message: wasmmodule.Message{Template: "Division by zero"},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, ProblemTypeValidation, classifier.Classify(tt.err))
		})
	}
}

func TestClassifyTariff_ModuleFailuresStayInternal(t *testing.T) {
	t.Parallel()

	classifier := NewDefaultClassifier()
	assert.Equal(t, ProblemTypeInternal, classifier.Classify(fmt.Errorf("check: %w", wasmmodule.ErrTrap)))
	assert.Equal(
		t,
		ProblemTypeInternal,
		classifier.Classify(&wasmmodule.RequestError{Op: rules.OpTariffCheck, Code: "malformed"}),
	)
}

func TestSanitizer_CarriesTheSourceRangeOfACheckError(t *testing.T) {
	t.Parallel()

	err := tarifferrors.CheckRefusal(&rules.TariffCheck{
		Diagnostics: []rules.TariffDiagnostic{{
			Range:    &rules.TariffRange{From: 12, To: 19},
			Severity: rules.SeverityError,
			Code:     "unknown-name",
			Message:  wasmmodule.Message{Template: "There is no `{0}`", Args: []any{"weigth"}},
		}},
	}, tarifferrors.CheckFields{})
	require.Error(t, err)

	extracted := NewSanitizer(false).ExtractErrors(err)
	require.Len(t, extracted, 1)
	assert.Equal(t, "source", extracted[0].Field)
	require.NotNil(t, extracted[0].Range)
	assert.Equal(t, 12, extracted[0].Range.From)
	assert.Equal(t, 19, extracted[0].Range.To)
}
