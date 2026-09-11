package helpers

import (
	"testing"

	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
)

func TestProblemBuilder_TranslatesTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		locale i18n.Locale
		want   string
	}{
		{i18n.EN, "Validation Failed"},
		{i18n.ES, "Error de validación"},
		{i18n.ZhTW, "驗證失敗"},
		{i18n.ZhCN, "验证失败"},
	}

	for _, tt := range tests {
		t.Run(string(tt.locale), func(t *testing.T) {
			t.Parallel()

			problem := NewProblemBuilder("https://example.test/problems/").
				WithLocale(tt.locale).
				WithType(ProblemTypeValidation).
				Build()

			assert.Equal(t, tt.want, problem.Title)
		})
	}
}

func TestProblemBuilder_TranslatesValidationMessages(t *testing.T) {
	t.Parallel()

	problem := NewProblemBuilder("https://example.test/problems/").
		WithLocale(i18n.ES).
		WithType(ProblemTypeValidation).
		WithErrors([]ValidationError{
			{Field: "name", Message: "Name is required", Code: "required"},
			{Field: "status", Message: "Status is invalid", Code: "invalid"},
		}).
		Build()

	assert.Equal(t, "El nombre es obligatorio", problem.Errors[0].Message)
	assert.Equal(t, "El estado no es válido", problem.Errors[1].Message)
	assert.Equal(t, "name", problem.Errors[0].Field,
		"translating the message must not disturb the field path the form binds to")
	assert.Equal(t, "required", problem.Errors[0].Code)
}

func TestProblemBuilder_UntranslatedMessageStaysEnglish(t *testing.T) {
	t.Parallel()

	problem := NewProblemBuilder("https://example.test/problems/").
		WithLocale(i18n.ZhCN).
		WithType(ProblemTypeValidation).
		WithErrors([]ValidationError{
			{Field: "vin", Message: "A message nobody has translated yet"},
		}).
		Build()

	assert.Equal(t, "A message nobody has translated yet", problem.Errors[0].Message,
		"a missing translation must fall back to English, never to an empty string")
}

func TestProblemBuilder_ZeroLocaleBehavesAsEnglish(t *testing.T) {
	t.Parallel()

	problem := NewProblemBuilder("https://example.test/problems/").
		WithType(ProblemTypeNotFound).
		Build()

	assert.Equal(t, "Resource Not Found", problem.Title,
		"a builder used without WithLocale must still render, not blank out")
}

func TestProblemBuilder_DoesNotMutateCallerErrors(t *testing.T) {
	t.Parallel()

	errs := []ValidationError{{Field: "name", Message: "Name is required"}}

	NewProblemBuilder("https://example.test/problems/").
		WithLocale(i18n.ES).
		WithType(ProblemTypeValidation).
		WithErrors(errs).
		Build()

	assert.Equal(t, "Name is required", errs[0].Message,
		"the caller's slice is shared with the logging path and must not be translated in place")
}
