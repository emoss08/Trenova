package docrender

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const brief = `Three loads will be **late tonight**[^1].

## Affected loads

| Load | Delay |
| --- | ---: |
| SEED-SHP-002 | +2 h |

1. Send the drafted notice
2. Re-check ETAs at 8 PM

> Northline receives overnight.`

func TestParseReadsTheBlocksTheDeskWrites(t *testing.T) {
	t.Parallel()

	blocks := Parse(brief)

	require.Len(t, blocks, 5)
	assert.Equal(t, BlockParagraph, blocks[0].Kind)
	assert.Equal(t, BlockHeading, blocks[1].Kind)
	assert.Equal(t, 2, blocks[1].Level)
	assert.Equal(t, BlockTable, blocks[2].Kind)
	assert.Equal(t, []string{"Load", "Delay"}, blocks[2].Header)
	assert.Equal(t, [][]string{{"SEED-SHP-002", "+2 h"}}, blocks[2].Rows)
	assert.Equal(t, BlockList, blocks[3].Kind)
	assert.True(t, blocks[3].Ordered)
	assert.Len(t, blocks[3].Items, 2)
	assert.Equal(t, BlockQuote, blocks[4].Kind)
	assert.Equal(t, "Three loads will be late tonight[1].", PlainText(blocks[0].Text))
}

func TestHTMLEscapesAndMarksCitations(t *testing.T) {
	t.Parallel()

	page := HTML(&Document{
		Title:   "Storm <impact>",
		Kicker:  "Brief · 30 words",
		Body:    brief,
		Sources: []Source{{N: 1, Label: "NWS warning", Tool: "weather_alerts"}},
	})

	assert.Contains(t, page, "<h1>Storm &lt;impact&gt;</h1>")
	assert.Contains(t, page, "<b>late tonight</b><sup>1</sup>")
	assert.Contains(t, page, "<ol><li>Send the drafted notice</li>")
	assert.Contains(t, page, "<h4>Sources</h4>")
}

// A Word file is a zip of the parts Word reads, with the body's text in it.
func TestDOCXIsAWordPackage(t *testing.T) {
	t.Parallel()

	file, err := DOCX(&Document{Title: "Storm & impact", Body: brief})
	require.NoError(t, err)

	archive, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	require.NoError(t, err)
	parts := map[string]string{}
	for _, part := range archive.File {
		reader, openErr := part.Open()
		require.NoError(t, openErr)
		content, readErr := io.ReadAll(reader)
		require.NoError(t, readErr)
		parts[part.Name] = string(content)
	}

	require.Contains(t, parts, "[Content_Types].xml")
	require.Contains(t, parts, "word/styles.xml")
	assert.Contains(t, parts["word/document.xml"], "Storm &amp; impact")
	assert.Contains(t, parts["word/document.xml"], `<w:vertAlign w:val="superscript"/>`)
	assert.Contains(t, parts["word/document.xml"], "<w:tbl>")
}
