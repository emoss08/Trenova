package documentcontent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagedText_MarksWhereEachPageBegins(t *testing.T) {
	t.Parallel()

	content := &Content{
		ContentText: "Acme guide\nMDN: synchronous",
		Pages: []*Page{
			{PageNumber: 2, ExtractedText: " MDN: synchronous "},
			{PageNumber: 1, ExtractedText: "Acme guide"},
			nil,
		},
	}

	assert.Equal(t, "--- page 1 ---\nAcme guide\n\n--- page 2 ---\nMDN: synchronous",
		content.PagedText())
}

func TestPagedText_KeepsTheTextOfAOnePageOrUnpagedDocument(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Rate confirmation", (&Content{
		ContentText: " Rate confirmation ",
		Pages:       []*Page{{PageNumber: 1, ExtractedText: "Rate confirmation"}},
	}).PagedText())
	assert.Equal(t, "Scanned text", (&Content{
		ContentText: "Scanned text",
		Pages:       []*Page{{PageNumber: 1}, {PageNumber: 2}},
	}).PagedText(), "pages with no text of their own leave the whole text as it was read")
}
