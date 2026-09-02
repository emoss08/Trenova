package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	formulaerrors "github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnguardedNullableFields(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		want       []string
	}{
		{"bare arithmetic on a nullable field", "weight * 2", []string{"weight"}},
		{"comparison on a nullable field", "weight > 100 ? 50 : 0", []string{"weight"}},
		{"coalesce guards it", "coalesce(weight, 0) * 2", nil},
		{"unrelated expression", "1 + 1", nil},
		{"unparseable expression", "1 +* 2", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			warnings, err := e.UnguardedNullableFields(t.Context(), tt.expression, "test", nil)
			require.NoError(t, err)

			fields := make([]string, 0, len(warnings))
			for _, warning := range warnings {
				fields = append(fields, warning.Field)
			}
			assert.Equal(t, tt.want, nilIfEmpty(fields))
		})
	}
}

func nilIfEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func TestUnguardedNullableFields_SuggestsATypedGuard(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	warnings, err := e.UnguardedNullableFields(t.Context(), "weight * 2", "test", nil)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Equal(t, "weight", warnings[0].Field)
	assert.Equal(t, "coalesce(weight, 0)", warnings[0].Suggestion)
}

func TestUnguardedNullableFields_UnknownSchema(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	_, err := e.UnguardedNullableFields(t.Context(), "weight * 2", "nope", nil)
	require.ErrorIs(t, err, engine.ErrSchemaNotFound)
}

func TestEvaluateWithEnv_NilFieldIsExplained(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	_, err := e.EvaluateWithEnv(t.Context(), &formulatemplatetypes.EnvEvaluationRequest{
		Expression: "weight * rate",
		Env:        map[string]any{"weight": nil, "rate": 2.5},
	})

	require.Error(t, err)
	var missing *formulaerrors.MissingFieldError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, []string{"weight"}, missing.Fields)
	assert.Contains(t, err.Error(), "coalesce(weight, 0)")

	result, err := e.EvaluateWithEnv(t.Context(), &formulatemplatetypes.EnvEvaluationRequest{
		Expression: "coalesce(weight, 0) * rate",
		Env:        map[string]any{"weight": nil, "rate": 2.5},
	})
	require.NoError(t, err)
	assert.True(t, result.Value.IsZero())
}
