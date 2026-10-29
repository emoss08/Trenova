package extractionartifact

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doclingStoredPages(t *testing.T, name string) []ExtractionPage {
	t.Helper()

	data, err := os.ReadFile(
		filepath.Join("..", "..", "..", "..", "infrastructure", "docling", "testdata", name),
	)
	require.NoError(t, err)

	var converted []kernelports.ConvertedPage
	require.NoError(t, sonic.Unmarshal(data, &converted))

	pages := make([]ExtractionPage, 0, len(converted))
	for _, page := range converted {
		lines := make([]documentcontent.LayoutLine, 0, len(page.Lines))
		for _, line := range page.Lines {
			lines = append(lines, documentcontent.LayoutLine{
				Text: line.Text, X: line.X, Y: line.Y, W: line.W, H: line.H,
			})
		}
		encoded, marshalErr := sonic.Marshal(map[string]any{documentcontent.MetadataLines: lines})
		require.NoError(t, marshalErr)
		var metadata map[string]any
		require.NoError(t, sonic.Unmarshal(encoded, &metadata))

		pages = append(pages, ExtractionPage{
			Number: page.Number,
			Lines:  documentcontent.LinesOf(metadata),
		})
	}

	return pages
}

/*
Docling places one line per text item and per table cell, top-left origin, as
fractions of the page. The extraction card finds a value in those lines the
same way it does in the legacy reader's, so a value read from a table cell is
boxed on that cell and one read from a header is boxed on its part of the line.
*/
func TestExtractionArtifact_BoxesValuesOnDoclingLines(t *testing.T) {
	t.Parallel()

	pages := doclingStoredPages(t, "rate-confirmation.pdf.golden.json")
	draft := map[string]any{
		"documentId":   "doc_1",
		"status":       "Ready",
		"documentKind": "RateConfirmation",
		"confidence":   0.97,
		"fields": map[string]any{
			"rate": map[string]any{
				"label": "Rate", "value": "2062.75", "confidence": 0.98, "pageNumber": 1.0,
			},
			"loadNumber": map[string]any{
				"label": "Load number", "value": "RC-48213", "confidence": 0.95, "pageNumber": 1.0,
			},
		},
	}

	artifact := ExtractionArtifact(
		"call_1",
		draft,
		ExtractionSource{FileName: "rate-confirmation.pdf", Pages: pages},
	)
	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindExtraction, artifact.Kind)
	assert.Equal(t, len(pages), artifact.Payload["pageCount"])

	fields, ok := artifact.Payload["fields"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, fields, 2)

	byKey := make(map[string]map[string]any, len(fields))
	for _, field := range fields {
		byKey[field["key"].(string)] = field
	}

	rate := byKey["rate"]
	require.NotNil(t, rate)
	assert.Equal(t, 1, rate["page"])
	assert.Equal(t, map[string]float64{
		"x": 0.7045, "y": 0.4571, "w": 0.0516, "h": 0.01,
	}, rate["box"], "the total is boxed on its own table cell past the dollar sign, no shorter than the floor")

	load := byKey["loadNumber"]
	require.NotNil(t, load)
	assert.Equal(t, 1, load["page"])
	assert.Equal(t, map[string]float64{
		"x": 0.8486, "y": 0.0327, "w": 0.0632, "h": 0.0105,
	}, load["box"], "the load number is boxed on its part of the header line, after the label")

	assert.Equal(t, "loadNumber", fields[0]["key"], "fields read top to bottom")
}
