package rules

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeValue_WritesEveryKindExactly(t *testing.T) {
	t.Parallel()

	weight := int64(1200)
	var noWeight *int64
	chicago := time.FixedZone("CST", -6*3600)

	encoded, err := EncodeEnv(map[string]any{
		"rate":      decimal.RequireFromString("0.1000000000000000000000000001"),
		"weight":    &weight,
		"missing":   noWeight,
		"fuel":      decimal.NullDecimal{},
		"stops":     int16(2),
		"rush":      true,
		"lane":      "TX \"OK\"\n\u0001",
		"pickup":    time.Date(2026, 1, 2, 8, 30, 0, 500, chicago),
		"wait":      90 * time.Minute,
		"customer":  map[string]any{"name": "Acme", "code": nil},
		"tags":      []string{"a", "b"},
		"lines":     []any{map[string]any{"weight": decimal.NewFromInt(5)}},
		"cached":    RawJSON(`[{"name":"fsc"}]`),
		"bigUnsign": uint64(18446744073709551615),
	})
	require.NoError(t, err)

	assert.Equal(t, `{`+
		`"bigUnsign":18446744073709551615,`+
		`"cached":[{"name":"fsc"}],`+
		`"customer":{"code":null,"name":"Acme"},`+
		`"fuel":null,`+
		`"lane":"TX \"OK\"\n\u0001",`+
		`"lines":[{"weight":5}],`+
		`"missing":null,`+
		`"pickup":{"$date":"2026-01-02T08:30:00.0000005-06:00"},`+
		`"rate":0.1000000000000000000000000001,`+
		`"rush":true,`+
		`"stops":2,`+
		`"tags":["a","b"],`+
		`"wait":{"$duration":"5400000000000"},`+
		`"weight":1200}`, string(encoded))
}

func TestEncodeValue_RefusesAFloatLoudlyNamingWhereItIs(t *testing.T) {
	t.Parallel()

	_, err := EncodeEnv(map[string]any{
		"stops": []any{map[string]any{"weight": 4000.0}},
	})
	require.ErrorIs(t, err, ErrInexactNumber)
	assert.Contains(t, err.Error(), "stops[0].weight")

	_, err = EncodeValue(float32(1.5))
	require.ErrorIs(t, err, ErrInexactNumber)

	_, err = EncodeValue(map[string][]float64{"x": {1}})
	require.ErrorIs(t, err, ErrInexactNumber)
}

func TestEncodeValue_RefusesReservedKeysAndUnknownKinds(t *testing.T) {
	t.Parallel()

	_, err := EncodeEnv(map[string]any{"$date": "2026-01-01"})
	require.ErrorIs(t, err, ErrReservedKey)

	_, err = EncodeEnv(map[string]any{"record": struct{ Name string }{"x"}})
	require.ErrorIs(t, err, ErrUnsupportedValue)
}

func TestEvaluateFormulas_OneCallPricesEveryProgramAgainstOneEnvironment(t *testing.T) {
	t.Parallel()

	resp, err := EvaluateFormulas(t.Context(), &EvaluateRequest{
		Programs: []FormulaProgram{
			{ID: "expression", Source: `lookup("fsc", fuelType) * miles`, Expect: ExpectAmount},
			{ID: "rush", Source: "isRush and miles > 100", Expect: ExpectPredicate},
			{ID: "count", Source: "miles - 120", Expect: ExpectPredicate},
			{ID: "missing", Source: "let rate = 2;\nrate * weight", Expect: ExpectAmount},
		},
		Env: map[string]any{
			"miles":    decimal.RequireFromString("120.5"),
			"fuelType": "DIESEL",
			"isRush":   true,
			"weight":   nil,
		},
		Tables: RawJSON(`[{"name":"fsc","axes":[{"match":"Exact"}],` +
			`"cells":[{"at":[{"key":"DIESEL"}],"value":"0.35"}]}]`),
		TraceLookups:    true,
		TraceStatements: true,
	})
	require.NoError(t, err)
	require.Len(t, resp.Results, 4)

	amount, err := resp.Result("expression").Amount()
	require.NoError(t, err)
	assert.Equal(t, "42.175", amount.String())
	lookups := resp.Result("expression").Lookups
	require.Len(t, lookups, 1)
	assert.Equal(t, "lookup", lookups[0].Function)
	assert.Equal(t, "fsc", lookups[0].Table)
	assert.Equal(t, "0.35", lookups[0].Value.String())
	assert.Equal(t, []RawJSON{RawJSON(`"DIESEL"`)}, lookups[0].Keys)
	require.NotNil(t, lookups[0].Match)
	assert.Equal(t, "DIESEL", lookups[0].Match.MatchedKey)

	holds, err := resp.Result("rush").Holds()
	require.NoError(t, err)
	assert.True(t, holds)
	holds, err = resp.Result("count").Holds()
	require.NoError(t, err)
	assert.True(t, holds, "a non-zero number holds")

	failed := resp.Result("missing")
	_, err = failed.Amount()
	require.ErrorIs(t, err, failed.Error)
	assert.Equal(t, ErrorMissingField, failed.Error.Code)
	assert.Equal(t, []string{"weight"}, failed.Error.Fields)
	assert.Equal(t, []string{"coalesce(weight, 0)"}, failed.Error.Guards)
	require.Len(t, failed.Statements, 2)
	assert.Equal(t, "let", failed.Statements[0].Kind)
	assert.Equal(t, "rate", failed.Statements[0].Name)
	assert.Equal(t, RawJSON(`"2"`), failed.Statements[0].Value)
	require.NotNil(t, failed.Statements[1].Error)
	assert.Equal(t, ErrorMissingField, failed.Statements[1].Error.Code)
}

func TestCheckFormula_DecodesWhatTheModuleSaysAndWritesItBackUnchanged(t *testing.T) {
	t.Parallel()

	input := &CheckRequest{
		Source: "hasHazmat * 2 + weight + foo",
		Expect: ExpectAmount,
		Schema: &FormulaSchema{Variables: []FormulaVariable{
			{Name: "hasHazmat", Type: "boolean"},
			{Name: "weight", Type: "number", Nullable: true},
		}},
		Tables: []FormulaTable{},
	}
	check, err := CheckFormula(t.Context(), input)
	require.NoError(t, err)
	require.False(t, check.Accepted())
	assert.Equal(t, "type", check.FirstError().Code)
	assert.Equal(t, []string{"weight"}, check.Unguarded())
	assert.True(t, check.ReadsField("hasHazmat"))

	encoded, err := sonic.Marshal(input)
	require.NoError(t, err)
	raw, err := CallRaw(t.Context(), OpFormulaCheck, encoded)
	require.NoError(t, err)
	var module struct {
		Diagnostics []sonic.NoCopyRawMessage `json:"diagnostics"`
	}
	require.NoError(t, sonic.Unmarshal(raw, &module))
	require.Len(t, module.Diagnostics, len(check.Diagnostics))
	for index := range check.Diagnostics {
		written, writeErr := check.Diagnostics[index].MarshalJSON()
		require.NoError(t, writeErr)
		assert.JSONEq(t, string(module.Diagnostics[index]), string(withArgs(t, written)))
	}
}

func withArgs(t *testing.T, written []byte) []byte {
	t.Helper()

	var value map[string]any
	require.NoError(t, sonic.Unmarshal(written, &value))
	if args, ok := value["args"].([]any); ok && len(args) == 0 {
		delete(value, "args")
	}
	encoded, err := sonic.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func TestFormulaMessages_TranslateWithNestedArguments(t *testing.T) {
	t.Parallel()

	check, err := CheckFormula(t.Context(), &CheckRequest{
		Source: `"85" + 1`,
		Schema: &FormulaSchema{},
	})
	require.NoError(t, err)
	refusal := check.FirstError()
	require.NotNil(t, refusal)

	english := refusal.Message.Localize(i18n.Default)
	assert.NotContains(t, english, "{0}")
	assert.NotEmpty(t, english)
	for _, arg := range refusal.Message.Args {
		if nested, ok := arg.(Message); ok {
			assert.Contains(t, english, nested.Localize(i18n.Default),
				"a nested phrase is translated and substituted")
		}
	}
}

func TestCatalog_ListsTheLanguageAndAnswersReservedNames(t *testing.T) {
	t.Parallel()

	catalog, err := Catalog(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, catalog.Functions)
	assert.NotEmpty(t, catalog.Builtins)
	assert.True(t, catalog.IsReserved("round"))
	assert.True(t, catalog.IsReserved("filter"))
	assert.False(t, catalog.IsReserved("baseRate"))

	names := make([]string, 0, len(catalog.Disabled))
	for _, disabled := range catalog.Disabled {
		names = append(names, disabled.Name)
		assert.NotEmpty(t, disabled.Message.Template)
	}
	assert.Contains(t, names, "now")

	again, err := Catalog(t.Context())
	require.NoError(t, err)
	assert.Same(t, catalog, again, "the catalog is read from the module once")
}
