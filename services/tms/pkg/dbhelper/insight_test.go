package dbhelper

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testInsightColumns = InsightColumns{
	"status": {Column: buncolgen.NewColumn("status", "sp"), Facetable: true},
	"weight": {Column: buncolgen.NewColumn("weight", "sp"), Summable: true},
}

func TestFacet_RefusesAFieldOutsideTheTablesList(t *testing.T) {
	for _, field := range []string{"secret", "weight", "status; drop table x"} {
		_, err := Facet(t.Context(), nil, testInsightColumns, &repositories.TableFacetRequest{
			Field: field,
		})

		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation, field)
		assert.Equal(t, "field", validation.Field)
	}
}

func TestAggregate_RefusesAFieldThatCannotBeTotalled(t *testing.T) {
	_, err := Aggregate(t.Context(), nil, testInsightColumns, &repositories.TableAggregateRequest{
		Fields: []string{"weight", "status"},
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "fields[1]", validation.Field)
}

func TestAggregate_RefusesTooManyFieldsAtOnce(t *testing.T) {
	fields := make([]string, MaxAggregateFields+1)
	for i := range fields {
		fields[i] = "weight"
	}

	_, err := Aggregate(t.Context(), nil, testInsightColumns, &repositories.TableAggregateRequest{
		Fields: fields,
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "fields", validation.Field)
}

func TestParseInsightDecimal(t *testing.T) {
	value, err := parseInsightDecimal("1234.50")
	require.NoError(t, err)
	assert.Equal(t, "1234.5", value.String())

	value, err = parseInsightDecimal([]byte("7"))
	require.NoError(t, err)
	assert.Equal(t, "7", value.String())

	value, err = parseInsightDecimal(nil)
	require.NoError(t, err)
	assert.Nil(t, value)

	_, err = parseInsightDecimal("not a number")
	require.Error(t, err)
}

func TestInsightColumnsFields_ListsEachFieldOnceInNameOrder(t *testing.T) {
	assert.Equal(t, []repositories.TableInsightField{
		{Name: "status", Facetable: true},
		{Name: "weight", Summable: true},
	}, testInsightColumns.Fields())
}
