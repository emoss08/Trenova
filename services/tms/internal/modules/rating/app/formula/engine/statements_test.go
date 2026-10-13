package engine_test

import (
	"testing"
	"unicode/utf16"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func utf16Slice(source string, start, end int) string {
	units := utf16.Encode([]rune(source))
	return string(utf16.Decode(units[start:end]))
}

func TestSplitStatements_LetsAndResult(t *testing.T) {
	t.Parallel()

	expression := "let base = rate * miles;\nlet fuel = base * 0.1;\nbase + fuel"
	statements, err := engine.SplitStatements(expression)
	require.NoError(t, err)
	require.Len(t, statements, 3)

	assert.Equal(t, formulatemplatetypes.StatementLet, statements[0].Kind)
	assert.Equal(t, "base", statements[0].Name)
	assert.Equal(t, "let base = rate * miles", statements[0].Code)
	assert.Equal(t, 0, statements[0].Index)

	assert.Equal(t, "fuel", statements[1].Name)
	assert.Equal(t, "let fuel = base * 0.1", statements[1].Code)

	assert.Equal(t, formulatemplatetypes.StatementResult, statements[2].Kind)
	assert.Empty(t, statements[2].Name)
	assert.Equal(t, "base + fuel", statements[2].Code)
	assert.Equal(t, 2, statements[2].Index)

	for _, statement := range statements {
		assert.Equal(t, statement.Code, utf16Slice(expression, statement.Start, statement.End))
	}
}

func TestSplitStatements_SemicolonsInsideStringsAndParens(t *testing.T) {
	t.Parallel()

	expression := `let label = "a;b"; let inner = (let x = 2; x * 3); label == "a;b" ? inner : 0`
	statements, err := engine.SplitStatements(expression)
	require.NoError(t, err)
	require.Len(t, statements, 3)

	assert.Equal(t, `let label = "a;b"`, statements[0].Code)
	assert.Equal(t, "let inner = (let x = 2; x * 3)", statements[1].Code)
	assert.Equal(t, `label == "a;b" ? inner : 0`, statements[2].Code)
}

func TestSplitStatements_UTF16Offsets(t *testing.T) {
	t.Parallel()

	expression := "let tag = \"🚚 é\"; // ✓ comment\nlet miles = 10; miles * 2;"
	statements, err := engine.SplitStatements(expression)
	require.NoError(t, err)
	require.Len(t, statements, 3)

	assert.Equal(t, 0, statements[0].Start)
	assert.Equal(
		t,
		`let tag = "🚚 é"`,
		utf16Slice(expression, statements[0].Start, statements[0].End),
	)
	assert.Equal(t, "let miles = 10", statements[1].Code)
	assert.Equal(
		t,
		"let miles = 10",
		utf16Slice(expression, statements[1].Start, statements[1].End),
	)
	assert.Equal(t, "miles * 2", statements[2].Code)
	assert.Equal(t, "miles * 2", utf16Slice(expression, statements[2].Start, statements[2].End))
}

func TestSplitStatements_SingleExpression(t *testing.T) {
	t.Parallel()

	statements, err := engine.SplitStatements("  totalDistance * 2  ")
	require.NoError(t, err)
	require.Len(t, statements, 1)
	assert.Equal(t, formulatemplatetypes.StatementResult, statements[0].Kind)
	assert.Equal(t, "totalDistance * 2", statements[0].Code)
	assert.Equal(t, 2, statements[0].Start)
	assert.Equal(t, 19, statements[0].End)
}

func TestSplitStatements_RefusesWhatDoesNotParse(t *testing.T) {
	t.Parallel()

	_, err := engine.SplitStatements("let a = ; a")
	require.Error(t, err)
}

func TestTraceStatements_EvaluatesEveryLet(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	statements, err := e.TraceStatements(t.Context(), &engine.TraceRequest{
		Expression: "let base = rate * miles; let label = \"x;y\"; let fast = miles > 100; base * 1.5",
		Env:        map[string]any{"rate": 2.5, "miles": 200.0},
	})
	require.NoError(t, err)
	require.Len(t, statements, 4)

	assert.Equal(t, "500", statements[0].Value)
	assert.Equal(t, "x;y", statements[1].Value)
	assert.Equal(t, true, statements[2].Value)
	assert.Equal(t, "750", statements[3].Value)
	for _, statement := range statements {
		assert.Empty(t, statement.Error)
	}
}

func TestTraceStatements_ALetThatFails(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	statements, err := e.TraceStatements(t.Context(), &engine.TraceRequest{
		Expression: "let zero = 0.0; let ratio = 1 / zero; let fine = 4; fine * 2",
		Env:        map[string]any{},
	})
	require.NoError(t, err)
	require.Len(t, statements, 4)

	assert.Equal(t, "0", statements[0].Value)
	assert.NotEmpty(t, statements[1].Error, "a non-finite value is reported, not returned")
	assert.Nil(t, statements[1].Value)
	assert.Equal(t, "4", statements[2].Value)
	assert.Equal(t, "8", statements[3].Value)
}

func TestTraceStatements_ResultErrorIsReported(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	statements, err := e.TraceStatements(t.Context(), &engine.TraceRequest{
		Expression: "let ok = 3; ok > 2",
		Env:        map[string]any{},
	})
	require.NoError(t, err)
	require.Len(t, statements, 2)

	assert.Equal(t, "3", statements[0].Value)
	assert.Contains(t, statements[1].Error, "true/false")
}
