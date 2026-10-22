package errors_test

import (
	stderrors "errors"
	"testing"

	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRefusal_PlacesEachErrorOnItsField(t *testing.T) {
	t.Parallel()

	check := &rules.TariffCheck{
		Diagnostics: []rules.TariffDiagnostic{
			{
				Range:    &rules.TariffRange{From: 40, To: 46},
				Severity: rules.SeverityError,
				Code:     "unit-mismatch",
				Message:  rules.Message{Template: "`rate` is money per mile"},
			},
			{
				Range:    &rules.TariffRange{From: 2, To: 3},
				Severity: rules.SeverityWarning,
				Code:     "unused",
				Message:  rules.Message{Template: "`x` is never used"},
			},
			{
				Severity: rules.SeverityError,
				Code:     tarifferrors.CodeInputValue,
				Path:     "agreement.breaks[2].rate",
				Message:  rules.Message{Template: "Expected money per hundredweight"},
			},
			{
				Range:    &rules.TariffRange{From: 0, To: 4},
				Severity: rules.SeverityError,
				Code:     "unknown-name",
				Module:   "org.accessorials",
				Message:  rules.Message{Template: "There is no `{0}`", Args: []any{"fee"}},
			},
		},
	}

	err := tarifferrors.CheckRefusal(check, tarifferrors.CheckFields{
		Source: "candidate.source",
		Value: func(path string) string {
			return "rules[0].inputs." + path
		},
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 3, "warnings never refuse")

	source := multiErr.Errors[0]
	assert.Equal(t, "candidate.source", source.Field)
	require.NotNil(t, source.Range)
	assert.Equal(t, errortypes.SourceRange{From: 40, To: 46}, *source.Range)

	value := multiErr.Errors[1]
	assert.Equal(t, "rules[0].inputs.agreement.breaks[2].rate", value.Field)
	assert.Nil(t, value.Range)

	library := multiErr.Errors[2]
	assert.Equal(t, "candidate.source", library.Field)
	assert.Nil(t, library.Range, "a range in a library's source is not a range in this source")
	assert.Equal(t, "In org.accessorials: There is no `fee`", library.Error())

	var diagnostic *tarifferrors.DiagnosticError
	require.True(t, stderrors.As(source.Internal, &diagnostic))
	assert.Equal(t, "unit-mismatch", diagnostic.Diagnostic.Code)
}

func TestCheckRefusal_CleanCheckRefusesNothing(t *testing.T) {
	t.Parallel()

	assert.NoError(t, tarifferrors.CheckRefusal(&rules.TariffCheck{OK: true}, tarifferrors.CheckFields{}))
	assert.NoError(t, tarifferrors.CheckRefusal(nil, tarifferrors.CheckFields{}))
}

func TestCheckRefusal_NotOKWithoutAnErrorStillRefuses(t *testing.T) {
	t.Parallel()

	err := tarifferrors.CheckRefusal(&rules.TariffCheck{OK: false}, tarifferrors.CheckFields{})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "source", multiErr.Errors[0].Field)
}

func TestEvaluationAndRefusal_CarryTheirRange(t *testing.T) {
	t.Parallel()

	evaluation := tarifferrors.Evaluation("formulaTemplateId", &rules.TariffEvalError{
		Code:    "overflow",
		Message: rules.Message{Template: "The amount passed {0} digits", Args: []any{int64(28)}},
		Range:   &rules.TariffRange{From: 5, To: 9},
	})
	assert.Equal(t, "formulaTemplateId", evaluation.Field)
	assert.Equal(t, errortypes.SourceRange{From: 5, To: 9}, *evaluation.Range)
	var typed *tarifferrors.EvaluationError
	require.ErrorAs(t, evaluation.Internal, &typed)
	assert.Equal(t, "overflow", typed.Code)
}
