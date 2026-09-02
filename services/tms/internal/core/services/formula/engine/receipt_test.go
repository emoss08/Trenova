package engine_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// explainingLookup answers lookups and can say which band or key matched,
// the way the matrix lookup does in production.
type explainingLookup struct {
	values map[string]float64
}

func (l *explainingLookup) Lookup(table string, _ any) (float64, error) {
	value, ok := l.values[table]
	if !ok {
		return 0, fmt.Errorf("rate table %q not found", table)
	}
	return value, nil
}

func (l *explainingLookup) Has(table string) bool { _, ok := l.values[table]; return ok }

func (l *explainingLookup) Lookup2(table string, _, _ any) (float64, error) {
	return l.Lookup(table, nil)
}

func (l *explainingLookup) Has2(table string) bool { return l.Has(table) }

func (l *explainingLookup) ExplainLookup(_ string, key any) (formulatypes.LookupMatch, bool) {
	low := decimal.NewFromInt(0)
	high := decimal.NewFromInt(500)
	return formulatypes.LookupMatch{
		MatchedKey: fmt.Sprint(key),
		BandMin:    &low,
		BandMax:    &high,
	}, true
}

func (l *explainingLookup) ExplainLookup2(_ string, row, col any) (formulatypes.LookupMatch, bool) {
	return formulatypes.LookupMatch{MatchedKey: fmt.Sprintf("%v/%v", row, col)}, true
}

func sourceOf(receipt *formulatypes.Receipt, name string) (formulatypes.ValueSource, bool) {
	for _, variable := range receipt.Variables {
		if variable.Name == name {
			return variable.Source, true
		}
	}
	return "", false
}

func TestEvaluate_ReceiptRecordsProvenanceAndLookups(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	template := &formulatemplate.FormulaTemplate{
		SchemaID:   "test",
		Expression: `lookup("miles", weight) * rate + surcharge`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "rate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.0},
			{Name: "surcharge", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 10.0},
		},
		BreakdownDefinitions: []*formulatypes.BreakdownDefinition{
			{Name: "fuel", Label: "Fuel", Expression: `lookup("fsc", weight)`},
		},
	}

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    &TestShipment{Weight: 100},
		Variables: map[string]any{"surcharge": 25.0},
		Overrides: map[string]any{"rate": 3.0},
		Lookup:    &explainingLookup{values: map[string]float64{"miles": 1.5, "fsc": 7}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Receipt)

	receipt := result.Receipt
	for name, want := range map[string]formulatypes.ValueSource{
		"weight":    formulatypes.ValueSourceField,
		"rate":      formulatypes.ValueSourceOverride,
		"surcharge": formulatypes.ValueSourceInput,
	} {
		got, ok := sourceOf(receipt, name)
		require.True(t, ok, name)
		assert.Equal(t, want, got, name)
	}
	for _, variable := range receipt.Variables {
		_, isFunc := variable.Value.(func(string, any) (float64, error))
		assert.False(t, isFunc, "lookup functions are not variables")
		assert.NotEqual(t, "__ctx", variable.Name)
	}

	require.Len(t, receipt.Lookups, 2)
	main := receipt.Lookups[0]
	assert.Equal(t, "miles", main.Table)
	assert.Equal(t, "expression", main.Scope)
	assert.InDelta(t, 1.5, main.Value, 0)
	require.NotNil(t, main.Match)
	assert.Equal(t, "100", main.Match.MatchedKey)
	assert.True(t, main.Match.BandMax.Equal(decimal.NewFromInt(500)))

	line := receipt.Lookups[1]
	assert.Equal(t, "fsc", line.Table)
	assert.Equal(t, "fuel", line.Scope)
}

func TestEvaluate_ReceiptMarksDeclaredDefaults(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	template := &formulatemplate.FormulaTemplate{
		SchemaID:   "test",
		Expression: `weight * rate`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "rate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.0},
		},
	}

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   &TestShipment{Weight: 10},
		Lookup:   &explainingLookup{values: map[string]float64{}},
	})
	require.NoError(t, err)

	source, ok := sourceOf(result.Receipt, "rate")
	require.True(t, ok)
	assert.Equal(t, formulatypes.ValueSourceDefault, source)
}

func TestEvaluateWithEnv_ReceiptTreatsEnvAsSample(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	result, err := e.EvaluateWithEnv(t.Context(), &formulatemplatetypes.EnvEvaluationRequest{
		Expression: "a + b",
		Env:        map[string]any{"a": 1.0, "b": 2.0},
		Lookup:     &explainingLookup{values: map[string]float64{}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Receipt)

	source, ok := sourceOf(result.Receipt, "a")
	require.True(t, ok)
	assert.Equal(t, formulatypes.ValueSourceSample, source)
	_, hasCtx := result.Variables["__ctx"]
	assert.False(t, hasCtx)
	_, hasLookup := result.Variables["lookup"]
	assert.False(t, hasLookup, "function values are stripped at the engine boundary")
}
