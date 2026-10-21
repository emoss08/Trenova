package templateengine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFor(t *testing.T, engine *Engine, channel Channel, source string) (*Compiled, Diagnostics) {
	t.Helper()
	return engine.Parse(&ParseRequest{Field: "body", Kind: "test", Channel: channel, Source: source})
}

func newTestEngine(t *testing.T, cfg Config) *Engine {
	t.Helper()
	engine, err := New(cfg)
	require.NoError(t, err)
	return engine
}

func TestParseRefusesARangeOverANumberInTheTemplate(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, Config{})
	for _, source := range []string{
		`{{range 1000000000}}{{end}}`,
		`{{range $i := 5}}{{$i}}{{end}}`,
		`{{range (1000000)}}{{end}}`,
	} {
		compiled, diags := parseFor(t, engine, ChannelEmailText, source)
		assert.Nil(t, compiled, source)
		assert.True(t, hasCode(diags, CodeRangeOverNumber), "%s: %v", source, diags)
	}
}

func TestParseRefusesRangesNestedTooDeep(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, Config{})
	source := strings.Repeat(`{{range seq 2}}`, maxRangeDepth+1) + strings.Repeat(`{{end}}`, maxRangeDepth+1)
	compiled, diags := parseFor(t, engine, ChannelEmailText, source)
	assert.Nil(t, compiled)
	assert.True(t, hasCode(diags, CodeRangeTooDeep), "%v", diags)

	allowed := strings.Repeat(`{{range seq 2}}`, maxRangeDepth) + "x" + strings.Repeat(`{{end}}`, maxRangeDepth)
	compiled, diags = parseFor(t, engine, ChannelEmailText, allowed)
	require.NotNil(t, compiled, "%v", diags)
	out, err := compiled.Render(t.Context(), nil, 1<<20)
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("x", 1<<maxRangeDepth), out)
}

func TestParseRefusesATemplateThatCallsTheRangeGuard(t *testing.T) {
	t.Parallel()

	compiled, diags := parseFor(t, newTestEngine(t, Config{}), ChannelEmailText,
		`{{`+rangeGuardName+` .Items}}`)
	assert.Nil(t, compiled)
	assert.True(t, diags.HasErrors())
}

func TestRenderStopsNestedLoopsThatWriteNothingAtTheIterationBudget(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, Config{})
	for _, channel := range []Channel{ChannelEmailText, ChannelEmailHTML} {
		compiled, diags := parseFor(t, engine, channel,
			`{{range seq 10000}}{{range seq 10000}}{{range seq 10000}}{{end}}{{end}}{{end}}done`)
		require.NotNil(t, compiled, "%v", diags)

		start := time.Now()
		_, err := compiled.Render(t.Context(), nil, 1<<20)
		require.ErrorIs(t, err, ErrRenderBudget, channel)
		assert.Less(t, time.Since(start), 5*time.Second, channel)
	}
}

func TestRenderHonoursTheDeadlineWhenNothingIsWritten(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, Config{MaxRangeIterations: 1 << 62})
	compiled, diags := parseFor(t, engine, ChannelEmailText,
		`{{range seq 10000}}{{range seq 10000}}{{range seq 10000}}{{end}}{{end}}{{end}}done`)
	require.NotNil(t, compiled, "%v", diags)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := compiled.Render(ctx, nil, 1<<20)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestRenderRefusesARangeOverALargeNumberInTheData(t *testing.T) {
	t.Parallel()

	compiled, diags := parseFor(t, newTestEngine(t, Config{}), ChannelEmailText,
		`{{range .Count}}x{{end}}`)
	require.NotNil(t, compiled, "%v", diags)

	_, err := compiled.Render(t.Context(), map[string]any{"Count": 1_000_000_000}, 1<<20)
	require.ErrorIs(t, err, ErrRenderBudget)

	out, err := compiled.Render(t.Context(), map[string]any{"Count": 3}, 1<<20)
	require.NoError(t, err)
	assert.Equal(t, "xxx", out)
}

func TestRenderKeepsRangeVariablesElseAndMapOrder(t *testing.T) {
	t.Parallel()

	compiled, diags := parseFor(t, newTestEngine(t, Config{}), ChannelEmailHTML,
		`{{range $i, $row := .Rows}}{{$i}}={{$row}};{{end}}|`+
			`{{range .Empty}}x{{else}}none{{end}}|`+
			`{{range $k, $v := .Totals}}{{$k}}:{{$v}},{{end}}`)
	require.NotNil(t, compiled, "%v", diags)

	data := map[string]any{
		"Rows":   []string{"a", "b"},
		"Empty":  []string{},
		"Totals": map[string]int{"b": 2, "a": 1},
	}
	for range 3 {
		out, err := compiled.Render(t.Context(), data, 1<<20)
		require.NoError(t, err)
		assert.Equal(t, "0=a;1=b;|none|a:1,b:2,", out)
	}
}
