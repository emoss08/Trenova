package datatransformer

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/stretchr/testify/assert"
)

func TestCleanText(t *testing.T) {
	t.Parallel()

	t.Run("empty string returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", cleanText(""))
	})

	t.Run("already clean string unchanged", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", cleanText("hello world"))
	})

	t.Run("leading and trailing whitespace trimmed", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello", cleanText("  hello  "))
	})

	t.Run("multiple internal spaces collapsed to single space", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", cleanText("hello    world"))
	})

	t.Run("tabs and newlines collapsed", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", cleanText("hello\t\n\t world"))
	})
}

func TestApplyCase(t *testing.T) {
	t.Parallel()

	t.Run("CaseFormatUpper converts to uppercase", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "HELLO WORLD", applyCase("hello world", dataentrycontrol.CaseFormatUpper))
	})

	t.Run("CaseFormatLower converts to lowercase", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", applyCase("HELLO WORLD", dataentrycontrol.CaseFormatLower))
	})

	t.Run("CaseFormatTitleCase converts to title case", func(t *testing.T) {
		t.Parallel()
		assert.Equal(
			t,
			"Hello World",
			applyCase("hello world", dataentrycontrol.CaseFormatTitleCase),
		)
	})

	t.Run("CaseFormatAsEntered returns unchanged", func(t *testing.T) {
		t.Parallel()
		assert.Equal(
			t,
			"hElLo WoRlD",
			applyCase("hElLo WoRlD", dataentrycontrol.CaseFormatAsEntered),
		)
	})

	t.Run("empty string returns empty for upper", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", applyCase("", dataentrycontrol.CaseFormatUpper))
	})

	t.Run("empty string returns empty for lower", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", applyCase("", dataentrycontrol.CaseFormatLower))
	})

	t.Run("empty string returns empty for title case", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", applyCase("", dataentrycontrol.CaseFormatTitleCase))
	})

	t.Run("empty string returns empty for as entered", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", applyCase("", dataentrycontrol.CaseFormatAsEntered))
	})
}
