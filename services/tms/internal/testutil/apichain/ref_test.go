package apichain_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scope() *apichain.Scope {
	s := apichain.NewScope()
	s.Fixtures["customer"] = "cus_01"
	s.Run[apichain.RunUnique] = "x7k2"
	s.Steps["create"] = apichain.StepValues{
		Request:  map[string]any{"variables": map[string]any{"rate": "1250.00"}},
		Response: map[string]any{"data": map[string]any{"moves": []any{map[string]any{"id": "smv_9"}}}},
		Exports:  map[string]any{"shipment": "shp_1", "weight": float64(42000)},
	}

	return s
}

func TestResolve_AWholeReferenceKeepsTheValuesType(t *testing.T) {
	t.Parallel()

	got, err := scope().Resolve(map[string]any{
		"weight": "${create.weight}",
		"items":  []any{"${fixture.customer}"},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"weight": float64(42000), "items": []any{"cus_01"}}, got)
}

func TestResolve_AReferenceInsideTextIsInterpolated(t *testing.T) {
	t.Parallel()

	got, err := scope().ResolveString(
		"/api/v1/shipment-moves/${create.response.data.moves.0.id}/assignment/",
	)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/shipment-moves/smv_9/assignment/", got)
}

func TestResolve_ReadsTheRequestAStepSent(t *testing.T) {
	t.Parallel()

	got, err := scope().Resolve("BOL-${run.unique}-${create.request.variables.rate}")

	require.NoError(t, err)
	assert.Equal(t, "BOL-x7k2-1250.00", got)
}

func TestResolve_NamesWhatIsMissing(t *testing.T) {
	t.Parallel()

	_, err := scope().Resolve("${create.invoice}")
	require.ErrorContains(t, err, `exports no "invoice"`)

	_, err = scope().Resolve("${later.id}")
	require.ErrorContains(t, err, `step "later" has not run`)

	_, err = scope().Resolve("${create.response.data.moves.4.id}")
	require.ErrorContains(t, err, "nothing at")
}

func TestReferences_FindsEveryReferenceInNestedValues(t *testing.T) {
	t.Parallel()

	refs := apichain.References(map[string]any{
		"a": []any{"${x.id}", map[string]any{"b": "pre-${fixture.y}-${run.now}"}},
		"c": 7,
	})

	assert.ElementsMatch(t, []string{"x.id", "fixture.y", "run.now"}, refs)
}

func TestGet_WalksMapsAndListIndexes(t *testing.T) {
	t.Parallel()

	root := map[string]any{"data": map[string]any{"items": []any{"a", "b"}}}

	got, ok := apichain.Get(root, "data.items.1")
	assert.True(t, ok)
	assert.Equal(t, "b", got)

	_, ok = apichain.Get(root, "data.items.x")
	assert.False(t, ok)
}
