package engine_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingLookup struct {
	singleCalls [][]any
	twoCalls    [][]any
	values      map[string]float64
	misses      map[string]struct{}
}

func (r *recordingLookup) Lookup(table string, key any) (float64, error) {
	r.singleCalls = append(r.singleCalls, []any{table, key})
	if _, miss := r.misses[table]; miss {
		return 0, fmt.Errorf("%w: rate table %q has no entry for key %v",
			formulatemplatetypes.ErrRateTableMiss, table, key)
	}
	value, ok := r.values[table]
	if !ok {
		return 0, fmt.Errorf("rate table %q not found", table)
	}
	return value, nil
}

func (r *recordingLookup) Has(table string) bool {
	_, ok := r.values[table]
	return ok
}

func (r *recordingLookup) Lookup2(table string, rowKey, colKey any) (float64, error) {
	r.twoCalls = append(r.twoCalls, []any{table, rowKey, colKey})
	if _, miss := r.misses[table]; miss {
		return 0, fmt.Errorf("%w: rate table %q has no cell matching row %v and column %v",
			formulatemplatetypes.ErrRateTableMiss, table, rowKey, colKey)
	}
	value, ok := r.values[table]
	if !ok {
		return 0, fmt.Errorf("two-axis rate table %q not found", table)
	}
	return value, nil
}

func (r *recordingLookup) Has2(table string) bool {
	_, ok := r.values[table]
	return ok
}

func TestExtractLookupTableRefs_SeparatesByArity(t *testing.T) {
	t.Parallel()

	refs, err := engine.ExtractLookupTableRefs(
		`lookup("fuel", x) + lookupOr("lane", y, 0) + ` +
			`lookup2("class_rates", r, c) + lookup2Or("zone_weight", r, c, 0)`,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"fuel", "lane"}, refs.Single)
	assert.Equal(t, []string{"class_rates", "zone_weight"}, refs.Multi)
}

func TestExtractLookupTableRefs_DeduplicatesWithinAnArity(t *testing.T) {
	t.Parallel()

	refs, err := engine.ExtractLookupTableRefs(
		`lookup("fuel", x) + lookup("fuel", y) + lookup2("grid", a, b) + lookup2Or("grid", a, b, 0)`,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"fuel"}, refs.Single)
	assert.Equal(t, []string{"grid"}, refs.Multi)
}

func TestExtractLookupTableRefs_TableUsedByBothFamiliesAppearsInBoth(t *testing.T) {
	t.Parallel()

	refs, err := engine.ExtractLookupTableRefs(
		`lookup("grid", x) + lookup2("grid", a, b)`,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"grid"}, refs.Single)
	assert.Equal(t, []string{"grid"}, refs.Multi)
}

func TestExtractLookupTableRefs_IgnoresNonLiteralTableNames(t *testing.T) {
	t.Parallel()

	refs, err := engine.ExtractLookupTableRefs(`lookup2(tableName, a, b) + lookup(other, x)`)

	require.NoError(t, err)
	assert.Empty(t, refs.Single)
	assert.Empty(t, refs.Multi)
}

func TestExtractLookupTables_UnionsBothFamiliesWithoutDuplicates(t *testing.T) {
	t.Parallel()

	tables, err := engine.ExtractLookupTables(
		`lookup("fuel", x) + lookup2("grid", a, b) + lookupOr("grid", x, 0)`,
	)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"fuel", "grid"}, tables)
}

func TestEngine_EvaluateExpression_Lookup2CallsTheProvider(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{values: map[string]float64{"class_rates": 12.25}}

	result, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookup2("class_rates", "ZONE_5", 8500)`,
			Entity:     struct{}{},
			SchemaID:   "test",
			Lookup:     provider,
		},
	)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(12.25).Equal(result.Value))
	require.Len(t, provider.twoCalls, 1)
	assert.Equal(t, "class_rates", provider.twoCalls[0][0])
	assert.Equal(t, "ZONE_5", provider.twoCalls[0][1])
	assert.EqualValues(t, 8500, provider.twoCalls[0][2])
}

func TestEngine_EvaluateExpression_Lookup2OrFallsBackOnMiss(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{
		values: map[string]float64{},
		misses: map[string]struct{}{"grid": {}},
	}

	result, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookup2Or("grid", "row", 5, 2.5)`,
			Entity:     struct{}{},
			SchemaID:   "test",
			Lookup:     provider,
		},
	)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(2.5).Equal(result.Value))
}

func TestEngine_EvaluateExpression_LookupOrFallsBackOnMiss(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{
		values: map[string]float64{},
		misses: map[string]struct{}{"fsc": {}},
	}

	result, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookupOr("fsc", "GAS", 1.25)`,
			Entity:     struct{}{},
			SchemaID:   "test",
			Lookup:     provider,
		},
	)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(1.25).Equal(result.Value))
}

func TestEngine_EvaluateExpression_LookupOrPropagatesMissingTable(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{values: map[string]float64{}}

	_, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookupOr("missing", "x", 1.25)`,
			Entity:     struct{}{},
			SchemaID:   "test",
			Lookup:     provider,
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `rate table "missing" not found`)
}

func TestEngine_EvaluateExpression_Lookup2OrPropagatesMissingTable(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{values: map[string]float64{}}

	_, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookup2Or("missing", "row", 5, 2.5)`,
			Entity:     struct{}{},
			SchemaID:   "test",
			Lookup:     provider,
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `two-axis rate table "missing" not found`)
}

func TestEngine_EvaluateExpression_NilLookupIsUnavailableNotZero(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	_, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: `lookup("fsc", "DIESEL")`,
			Entity:     struct{}{},
			SchemaID:   "test",
		},
	)

	require.ErrorIs(t, err, formulatemplatetypes.ErrRateTableUnavailable)
}

func TestEngine_EvaluateWithEnv_UsesProvidedLookup(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	provider := &recordingLookup{values: map[string]float64{"fsc": 0.35}}

	result, err := e.EvaluateWithEnv(
		t.Context(),
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: `lookup("fsc", "DIESEL") * miles`,
			Env:        map[string]any{"miles": 100.0},
			Lookup:     provider,
		},
	)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(35).Equal(result.Value))
	require.Len(t, provider.singleCalls, 1)
}

func TestEngine_StubLookup_ResolvesEveryTableToZero(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	result, err := e.EvaluateWithEnv(
		t.Context(),
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: `lookup("anything", 1) + lookup2("grid", "a", 2) + 4`,
			Env:        map[string]any{},
			Lookup:     engine.StubLookup{},
		},
	)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(4).Equal(result.Value))
}

func TestEngine_EvaluateExpression_Lookup2NamesAreReserved(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	for _, name := range []string{"lookup", "lookupOr", "lookup2", "lookup2Or"} {
		_, err := e.EvaluateExpression(
			t.Context(),
			&formulatemplatetypes.ExpressionEvaluationRequest{
				Expression: "1 + 1",
				Entity:     struct{}{},
				SchemaID:   "test",
				Variables:  map[string]any{name: 1.0},
			},
		)
		require.ErrorIs(t, err, engine.ErrReservedVariableName, name)
	}
}
