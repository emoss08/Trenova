package templateengine

import (
	"slices"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFuncMapDelegatesToSharedHelpers is the regression net for the extractions
// out of invoiceservice and detentionservice. Every formatter here must produce
// byte-identical output to the shared helper it wraps, or a date on an invoice
// PDF stops matching the same date in the email that delivers it.
func TestFuncMapDelegatesToSharedHelpers(t *testing.T) {
	const ts = int64(1784131200)
	const tz = "America/Chicago"

	fm := FuncMap()

	t.Run("money", func(t *testing.T) {
		fn, ok := fm["money"].(func(string, decimal.Decimal) string)
		require.True(t, ok, "money signature changed")

		value := decimal.RequireFromString("1234.5")
		assert.Equal(t, money.FormatDecimal("USD", value), fn("USD", value))
		assert.Equal(t, "USD 1234.50", fn("USD", value))
		// An empty currency must not produce a bare number on a document.
		assert.Equal(t, "USD 1234.50", fn("", value))
		assert.Equal(t, "CAD 1234.50", fn("cad", value))
	})

	t.Run("date", func(t *testing.T) {
		fn, ok := fm["date"].(func(int64, string) string)
		require.True(t, ok, "date signature changed")

		assert.Equal(t, timeutils.FormatUnixDateIn(ts, tz), fn(ts, tz))
		assert.Equal(t, "2026-07-15", fn(ts, tz))
		assert.Empty(t, fn(0, tz), "an unset timestamp renders as blank, not as 1970")
		// A bad timezone must not stop a document from rendering.
		assert.Equal(t, "2026-07-15", fn(ts, "Not/AZone"))
	})

	t.Run("datetime", func(t *testing.T) {
		fn, ok := fm["datetime"].(func(int64, string) string)
		require.True(t, ok)

		assert.Equal(t, timeutils.FormatUnixDateTimeIn(ts, tz), fn(ts, tz))
		assert.Contains(t, fn(ts, tz), "2026")
		assert.Empty(t, fn(0, tz))
	})

	t.Run("stamp", func(t *testing.T) {
		fn, ok := fm["stamp"].(func(int64, string) string)
		require.True(t, ok)

		assert.Equal(t, timeutils.FormatStampIn(ts, tz), fn(ts, tz))
		assert.Equal(t, timeutils.FormatStamp(ts, timeutils.LoadLocation(tz)), fn(ts, tz))
		// The weekday and zone are what make an arrival time hard to dispute.
		assert.Contains(t, fn(ts, tz), "Wed")
		assert.Contains(t, fn(ts, tz), "CDT")
	})

	t.Run("minutes", func(t *testing.T) {
		fn, ok := fm["minutes"].(func(int32) string)
		require.True(t, ok)

		for _, m := range []int32{-5, 0, 1, 45, 59, 60, 61, 90, 120, 200} {
			assert.Equal(t, timeutils.FormatMinutes(m), fn(m))
		}
		assert.Equal(t, "0 min", fn(0))
		assert.Equal(t, "45 min", fn(45))
		assert.Equal(t, "1h", fn(60))
		assert.Equal(t, "1h 30m", fn(90))
	})

	t.Run("duration", func(t *testing.T) {
		fn, ok := fm["duration"].(func(int64) string)
		require.True(t, ok)

		ms := (90 * time.Minute).Milliseconds()
		assert.Equal(t, timeutils.FormatLongDurationMs(ms), fn(ms))
	})

	t.Run("title", func(t *testing.T) {
		fn, ok := fm["title"].(func(string) string)
		require.True(t, ok)
		assert.Equal(t, stringutils.CapitalizeFirst("acme freight"), fn("acme freight"))
	})

	t.Run("bytes", func(t *testing.T) {
		fn, ok := fm["bytes"].(func(int64) string)
		require.True(t, ok)
		assert.NotEmpty(t, fn(1<<20))
	})
}

// TestPipelineArgumentOrder covers the wrappers whose argument order differs from
// the shared helper, because Go passes a piped value as the final argument.
func TestPipelineArgumentOrder(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		assert.Equal(t, "fallback", withDefault("fallback", ""))
		assert.Equal(t, "actual", withDefault("fallback", "actual"))
		// Matches the shared helper with the arguments the other way round.
		assert.Equal(t, stringutils.WithDefault("", "fallback"), withDefault("fallback", ""))
	})

	t.Run("truncate", func(t *testing.T) {
		assert.Equal(t, stringutils.TruncateRunes("abcdefgh", 4), truncate(4, "abcdefgh"))
	})
}

func TestJoinNonEmptyDropsBlanks(t *testing.T) {
	// An address block with no second street line must not leave a dangling comma
	// on a customer's invoice.
	assert.Equal(t, "1 Main St, Austin, TX", joinNonEmpty(", ", []string{"1 Main St", "", "Austin, TX"}))
	assert.Empty(t, joinNonEmpty(", ", []string{"", ""}))
	assert.Empty(t, joinNonEmpty(", ", nil))
}

func TestSeqIsBounded(t *testing.T) {
	assert.Nil(t, seq(0))
	assert.Nil(t, seq(-1))
	assert.Len(t, seq(3), 3)
	assert.Equal(t, []int{0, 1, 2}, seq(3))
	// A template must not be able to ask for a billion iterations.
	assert.Len(t, seq(1_000_000), maxSeqLength)
}

// TestForbiddenFunctionsAreAbsentFromTheMap asserts the escape hatches never
// reappear. If someone adds a "raw" helper to make one template easier to write,
// this fails before it ships.
func TestForbiddenFunctionsAreAbsentFromTheMap(t *testing.T) {
	for _, mapName := range []string{"html", "text"} {
		fm := FuncMap()
		if mapName == "text" {
			fm = TextFuncMap()
		}

		for _, forbidden := range ForbiddenFunctionNames() {
			_, present := fm[forbidden]
			assert.False(t, present,
				"%s FuncMap must not define %q: it would defeat contextual auto-escaping",
				mapName, forbidden)
		}
	}
}

func TestForbiddenFunctionsAreNotAllowed(t *testing.T) {
	allowed := AllowedFunctionNames()

	for _, forbidden := range ForbiddenFunctionNames() {
		assert.False(t, slices.Contains(allowed, forbidden),
			"%q must not be in the allowed set", forbidden)
	}
}

func TestTextFuncMapDropsOnlyNl2br(t *testing.T) {
	htmlMap := FuncMap()
	textMap := TextFuncMap()

	assert.Contains(t, htmlMap, "nl2br")
	assert.NotContains(t, textMap, "nl2br")
	assert.Len(t, textMap, len(htmlMap)-1)
}

func TestAllowedFunctionNamesIsSortedAndComplete(t *testing.T) {
	names := AllowedFunctionNames()

	require.NotEmpty(t, names)
	assert.True(t, slices.IsSorted(names), "the editor shows this list verbatim")

	for name := range FuncMap() {
		assert.Contains(t, names, name)
	}
	for _, name := range builtinTemplateFunctions {
		assert.Contains(t, names, name)
	}
}

func TestNl2brEscapesThenAddsBreaks(t *testing.T) {
	got := string(nl2br("a\n<b>"))

	assert.Equal(t, "a<br>&lt;b&gt;", got)
	assert.Equal(t, stringutils.TextToHTMLBreaks("a\n<b>"), got)
}
