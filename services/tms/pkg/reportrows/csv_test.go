package reportrows_test

import (
	"bytes"
	"testing"

	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCSVKeepsRawValuesUnderColumnIDs(t *testing.T) {
	t.Parallel()

	envelope, err := reportrows.Decode([]byte(`{
		"meta": {"title": "Margin by lane", "generatedAt": 1790000000},
		"schema": [
			{"id": "lane", "label": "Lane", "type": "string"},
			{"id": "margin", "label": "Margin", "type": "decimal"},
			{"id": "loads", "label": "Loads", "type": "integer"},
			{"id": "deliveredAt", "label": "Delivered", "type": "timestamp"},
			{"id": "hazmat", "label": "Hazmat", "type": "boolean"}
		],
		"rows": [
			["DAL → ATL, \"express\"", "2840.125000", 12, 1789990000, true],
			["HOU → MEM", null, 0, null, false]
		],
		"totals": ["", "2840.125000", 12, null, null],
		"summary": {"rowCount": 2, "truncated": false}
	}`))
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, envelope.WriteCSV(&out))
	assert.Equal(t,
		"lane,margin,loads,deliveredAt,hazmat\n"+
			"\"DAL → ATL, \"\"express\"\"\",2840.125000,12,1789990000,true\n"+
			"HOU → MEM,,0,,false\n",
		out.String(),
	)
}
