package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslateRichSplitsATemplateAtItsTags(t *testing.T) {
	t.Parallel()

	got := TranslateRich(
		EN,
		"<s>{0}</s> completed <b>{1}</b> of its runs.",
		[]string{"s", "b"},
		"Llama",
		"93%",
	)

	assert.Equal(t, []RichPiece{
		{Tag: "s", Text: "Llama"},
		{Text: " completed "},
		{Tag: "b", Text: "93%"},
		{Text: " of its runs."},
	}, got)
}

func TestTranslateRichFormatsPiecesSoAnArgumentIsNeverReadAsATag(t *testing.T) {
	t.Parallel()

	got := TranslateRich(EN, "<s>{0}</s> recovered", []string{"s"}, "<b>vLLM</b>")

	assert.Equal(t, []RichPiece{
		{Tag: "s", Text: "<b>vLLM</b>"},
		{Text: " recovered"},
	}, got)
}

func TestTranslateRichTreatsUndeclaredAngleBracketsAsText(t *testing.T) {
	t.Parallel()

	got := TranslateRich(EN, "<b>{0}</b> < 2s <i>", []string{"b"}, "Fast")

	assert.Equal(t, []RichPiece{{Tag: "b", Text: "Fast"}, {Text: " < 2s <i>"}}, got)
}

func TestTranslateRichKeepsASelfClosingTag(t *testing.T) {
	t.Parallel()

	got := TranslateRich(EN, "Press <kbd/> to go", []string{"kbd"})

	assert.Equal(t, []RichPiece{{Text: "Press "}, {Tag: "kbd"}, {Text: " to go"}}, got)
}

func TestTranslateRichFallsBackToTheWordsWhenTheTemplateIsBroken(t *testing.T) {
	t.Parallel()

	for _, message := range []string{
		"<b>{0} never closes",
		"stray </b> {0}",
		"<b>nested <b>{0}</b></b>",
	} {
		got := TranslateRich(EN, message, []string{"b"}, "x")

		assert.Len(t, got, 1, message)
		assert.Empty(t, got[0].Tag, message)
		assert.NotContains(t, got[0].Text, "<b>", message)
		assert.NotContains(t, got[0].Text, "</b>", message)
	}
}

func TestTranslateRichReturnsNothingForAnEmptyMessage(t *testing.T) {
	t.Parallel()

	assert.Nil(t, TranslateRich(EN, "", []string{"b"}))
}
