package listtool

import (
	"testing"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type probeRow struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Status string `json:"status"`
}

func rowSpec() *Spec {
	spec := probeSpec(&capturedList{})
	spec.Name = "list_probe_rows"
	spec.Row = probeRow{}

	return spec
}

/*
A script asked shipment_list for fields=["id"] and was refused: the id was
left out of the columns fields may name, because every row keeps it. The id
is now a column fields may name, which changes nothing about the row.
*/
func TestListTool_FieldsTakesTheIDItAlwaysKeeps(t *testing.T) {
	t.Parallel()

	schema := New(rowSpec()).ParamSchema()

	for _, fields := range [][]any{{"id"}, {"id", "code"}, {"status"}} {
		assert.NoError(t, toolschema.Validate(schema, map[string]any{"fields": fields}),
			"fields %v", fields)
	}
	require.Error(t, toolschema.Validate(schema, map[string]any{"fields": []any{"secret"}}))
}

func TestColumnSource_PutsTheIDFirstOnce(t *testing.T) {
	t.Parallel()

	source := ColumnSource("list_probe_columns.column", []string{"code", "id", "status"})

	assert.Equal(t, []string{"id", "code", "status"}, source.Names())
	assert.True(t, OffersFields(source))
	assert.False(t, OffersFields(ColumnSource("list_probe_one.column", []string{"code"})))
}
