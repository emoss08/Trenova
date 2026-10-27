package render

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const browserContract = "../../../../../crates/rules/testdata/report.json"

type contractFile struct {
	Op    string         `json:"op"`
	Cases []contractCase `json:"cases"`
}

type contractCase struct {
	Name  string `json:"name"`
	Op    string `json:"op"`
	Input struct {
		Columns []contractColumn `json:"columns"`
	} `json:"input"`
	Output struct {
		Columns []struct {
			Values []map[string]string `json:"values"`
		} `json:"columns"`
	} `json:"output"`
}

type contractColumn struct {
	Resolve *struct {
		Type string       `json:"type"`
		Hint format.Kind  `json:"hint"`
		Spec *format.Spec `json:"spec"`
		Band *format.Band `json:"band"`
	} `json:"resolve"`
	Display   *format.Resolved `json:"display"`
	ValueType string           `json:"valueType"`
	Values    []any            `json:"values"`
}

func contractType(name string) (value.Type, bool) {
	switch name {
	case "decimal", "":
		return value.Decimal, true
	case "int":
		return value.Int, true
	case "float":
		return value.Float, true
	case "bool":
		return value.Bool, true
	case "string", "enum", "ref":
		return value.String, true
	default:
		return 0, false
	}
}

type contractNumber interface {
	Float64() (float64, error)
	String() string
}

func contractValue(v any) any {
	n, ok := v.(contractNumber)
	if !ok {
		return v
	}
	f, err := n.Float64()
	if err != nil || math.IsInf(f, 0) {
		return n.String()
	}
	return f
}

func typedValue(t value.Type, v any) bool {
	switch v.(type) {
	case nil:
		return true
	case bool:
		return t == value.Bool
	case string:
		return t == value.String
	case float64:
		return t.Numeric()
	default:
		return false
	}
}

type contractCell struct {
	name   string
	column int
	value  int
}

var knownDivergences = map[contractCell]string{
	{"a negative that rounds to zero keeps its sign, as in Go", 0, 0}: "keel drops the sign of a negative that rounds to zero",
	{"a negative that rounds to zero keeps its sign, as in Go", 1, 0}: "keel drops the sign of a negative that rounds to zero",
	{"currency symbols, prefixes and suffixes", 0, 0}:                 "keel separates a currency code with a no-break space",
	{"currency symbols, prefixes and suffixes", 3, 0}:                 "keel writes CAD as CA$, the browser as C$",
	{"a number too large to read exactly is shown as written", 0, 0}:  "keel expands a number it cannot read exactly",
}

func TestServerFormattingMatchesTheBrowsersContract(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(browserContract)
	require.NoError(t, err)
	var contract contractFile
	require.NoError(t, sonic.Config{UseNumber: true}.Froze().Unmarshal(raw, &contract))

	checked := 0
	diverged := make(map[contractCell]string)
	for _, tc := range contract.Cases {
		if op := tc.Op; op != "" && op != "report.format" {
			continue
		}
		for col, column := range tc.Input.Columns {
			if col >= len(tc.Output.Columns) {
				break
			}
			kind, display, ok := contractDisplay(column)
			if !ok {
				continue
			}
			formatter := format.New(kind, display, format.Options{})
			for i, raw := range column.Values {
				want := tc.Output.Columns[col].Values[i]
				cell := contractValue(raw)
				_, number := raw.(contractNumber)
				if _, isDate := want["date"]; isDate || (!number && !typedValue(kind, cell)) {
					continue
				}
				checked++
				text, tone := formatter.Format(cell)
				at := contractCell{name: tc.Name, column: col, value: i}
				if wantText, ok := want["text"]; ok && wantText != text {
					diverged[at] = fmt.Sprintf("text %q, browser %q", text, wantText)
				}
				wantTone, ok := want["tone"]
				if wantTone == "$absent" {
					wantTone = ""
				}
				if ok && wantTone != string(tone) {
					diverged[at] = fmt.Sprintf("tone %q, browser %q", tone, wantTone)
				}
			}
		}
	}
	assert.Positive(t, checked, "no case of the browser's contract was checked")

	for at, got := range diverged {
		_, known := knownDivergences[at]
		assert.Truef(
			t,
			known,
			"%s: column %d value %d: server %s",
			at.name,
			at.column,
			at.value,
			got,
		)
	}
	for at, reason := range knownDivergences {
		_, still := diverged[at]
		assert.Truef(
			t,
			still,
			"%s: column %d value %d now agrees (%s); remove it from knownDivergences",
			at.name,
			at.column,
			at.value,
			reason,
		)
	}
}

func contractDisplay(column contractColumn) (value.Type, format.Resolved, bool) {
	typeName := column.ValueType
	if column.Resolve != nil && typeName == "" {
		typeName = column.Resolve.Type
	}
	kind, ok := contractType(typeName)
	switch {
	case !ok:
		return 0, format.Resolved{}, false
	case column.Resolve != nil:
		hint := format.Hint{Kind: column.Resolve.Hint, Band: column.Resolve.Band}
		return kind, format.Resolve(kind, hint, column.Resolve.Spec), true
	case column.Display != nil:
		return kind, *column.Display, true
	default:
		return 0, format.Resolved{}, false
	}
}

func revenueTable(t *testing.T) *table.Table {
	t.Helper()
	columns := []table.Column{
		{
			ID:      "customer",
			Label:   "Customer",
			Type:    value.String,
			Display: format.Resolve(value.String, format.Hint{}, nil),
		},
		{
			ID:      "revenue",
			Label:   "Revenue",
			Type:    value.Decimal,
			Display: format.Resolve(value.Decimal, format.Hint{Kind: format.Money}, nil),
		},
	}
	builder := table.NewBuilder(columns)
	require.NoError(t, builder.Append("=HYPERLINK(\"x\")", "1200.5"))
	require.NoError(t, builder.Append("Globex", "-80"))
	require.NoError(t, builder.AppendLevel(table.Total, nil, "1120.5"))
	return builder.Table()
}

func renderRequest(t *testing.T, sink *bytes.Buffer) *reportports.ReportRenderRequest {
	t.Helper()
	return &reportports.ReportRenderRequest{
		Dataset: revenueTable(t).Reader(),
		Sink:    sink,
		Meta: reportports.ReportRunMeta{
			Title:           "Revenue",
			GeneratedAtUnix: 1784131200,
			Timezone:        "America/Chicago",
			Locale:          "en-US",
		},
	}
}

func TestCSVWritesFormattedCellsAndDefusesFormulas(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	stats, err := NewCSVRenderer(false, 0).Render(t.Context(), renderRequest(t, &out))
	require.NoError(t, err)

	text := out.String()
	assert.Equal(t, int64(2), stats.Rows)
	assert.False(t, stats.Truncated)
	assert.Contains(t, text, "Customer,Revenue")
	assert.Contains(t, text, "$1,200.50")
	assert.Contains(t, text, "($80.00)")
	assert.NotContains(t, text, "\n=HYPERLINK", "a cell that starts a formula is escaped")
	assert.Contains(t, text, "Total")
}

func TestCSVTranslatesTheTotalLabel(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ctx := i18n.WithLocale(t.Context(), i18n.ES)
	_, err := NewCSVRenderer(false, 0).Render(ctx, renderRequest(t, &out))
	require.NoError(t, err)

	assert.Contains(t, out.String(), i18n.Translate(i18n.ES, "Total"))
}

func TestCSVStopsAtTheByteLimitAndSaysSo(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	stats, err := NewCSVRenderer(false, 40).Render(t.Context(), renderRequest(t, &out))
	require.NoError(t, err)

	assert.True(t, stats.Truncated)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 3, "the header, the row that crossed the limit, then the notice")
	assert.Equal(t, Text(t.Context()).BytesTruncated+",", lines[2])
	assert.LessOrEqual(t, len(lines[0])+1, 40, "no row starts past the byte limit")
	assert.NotContains(t, out.String(), "Globex")
}

func TestJSONIsTheEnvelopeTheRunsRowsAreReadBackFrom(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	stats, err := (&JSONRenderer{}).Render(t.Context(), renderRequest(t, &out))
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Rows)

	envelope, err := export.ReadJSON(&out)
	require.NoError(t, err)
	assert.Equal(t, "Revenue", envelope.Meta.Title)
	assert.Equal(t, "America/Chicago", envelope.Location.String())
	assert.Equal(t, int64(2), envelope.RowCount)
}

func TestXLSXWritesAWorkbook(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	stats, err := (&XLSXRenderer{}).Render(t.Context(), renderRequest(t, &out))
	require.NoError(t, err)

	assert.Equal(t, int64(2), stats.Rows)
	assert.True(t, strings.HasPrefix(out.String(), "PK"), "an XLSX file is a zip archive")
}
