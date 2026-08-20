package rateimport_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The template is only worth handing out if a sheet built from it imports
// without a hitch: every header must map, nothing may be reported as unplaced,
// and the examples themselves must read as rules.
func TestTemplateCSV_ImportsCleanly(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadCSV([]byte(rateimport.TemplateCSV()))
	require.NoError(t, err)
	require.Len(t, sheet.Rows, 2, "the template carries two example rows")

	mapping, unplaced := rateimport.GuessMapping(sheet.Headers)
	assert.Empty(t, unplaced, "every template header must be a name the importer recognises")
	require.NoError(t, mapping.Validate())

	templates := func(string) (pulid.ID, bool) { return pulid.MustNew("ft_"), true }

	for index, row := range sheet.Rows {
		rule, parseErr := rateimport.ParseRow(mapping, row, templates)
		require.NoErrorf(t, parseErr, "template example row %d must parse", index+1)
		require.NotNil(t, rule)
		assert.NotEmpty(t, rule.LaneKey)
	}
}
