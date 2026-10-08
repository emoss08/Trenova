package compiler

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func headerLabels(t *testing.T, locale i18n.Locale) []string {
	t.Helper()
	c := newTestCompiler(allowAllEngine())
	compiled, err := c.Compile(i18n.WithLocale(t.Context(), locale), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	}))
	require.NoError(t, err)

	labels := make([]string, 0, len(compiled.Columns))
	for _, column := range compiled.Columns {
		labels = append(labels, column.Label)
	}
	return labels
}

func TestColumnHeadersReadInTheRequestersLanguage(t *testing.T) {
	english := headerLabels(t, i18n.EN)
	assert.Equal(t, []string{"Customer Name", "Total Charge", "Shipments Count"}, english)

	spanish := headerLabels(t, i18n.ES)
	es := func(message string, args ...any) string {
		return i18n.Translate(i18n.ES, message, args...)
	}
	assert.Equal(t, []string{
		es("Customer") + " " + es("Name"),
		es("Total Charge"),
		es("{0} Count", es("Shipments")),
	}, spanish)
	assert.NotEqual(t, english, spanish)
}

func TestAggregationWordsAreSkippedOnlyWhereTheMessagePlacesThem(t *testing.T) {
	l := labeler{locale: i18n.EN}
	total := func(base string) string {
		return l.unlessRepeated(
			base,
			i18n.Translate(i18n.EN, "Total {0}", base),
			i18n.Translate(i18n.EN, "Total {0}", labelSlot),
		)
	}
	assert.Equal(t, "Total Charges", total("Total Charges"))
	assert.Equal(t, "Total Weight", total("Weight"))

	trailing := "{0} total"
	suffixed := func(base string) string {
		return l.unlessRepeated(
			base,
			strings.ReplaceAll(trailing, "{0}", base),
			strings.ReplaceAll(trailing, "{0}", labelSlot),
		)
	}
	assert.Equal(t, "Peso total", suffixed("Peso"))
	assert.Equal(t, "Importe total", suffixed("Importe total"))
	assert.Equal(t, "Total Charge", suffixed("Total Charge"))
	assert.Equal(t, "Cargo total", total("Cargo total"))
	assert.Equal(t, "Total Totality", total("Totality"))

	assert.Equal(t, "X", l.unlessRepeated("Y", "X", "no slot"))
}
