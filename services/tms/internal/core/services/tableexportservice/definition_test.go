package tableexportservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shipmentView() *View {
	return &View{
		Resource: permission.ResourceShipment,
		Columns: []Column{
			{Field: "proNumber", Label: "PRO"},
			{Field: "status", Label: "Status"},
			{Field: "computedEta", Label: "ETA"},
		},
	}
}

func TestBuild_ShowsTheColumnsTheCatalogKnowsAndNamesTheRest(t *testing.T) {
	built, err := Build(&reportcatalog.Default, shipmentView())

	require.NoError(t, err)
	assert.Equal(t, "shipment", built.Definition.Entity)
	require.Len(t, built.Definition.Columns, 2)
	assert.Equal(t, "proNumber", built.Definition.Columns[0].Ref.Field)
	assert.Equal(t, report.ColumnKindDimension, built.Definition.Columns[0].Kind)
	assert.Equal(t, "PRO", built.Definition.Columns[0].Label)
	assert.Equal(t, []string{"ETA"}, built.Skipped)
}

func TestBuild_CarriesFiltersAsAndAndEachGroupAsOr(t *testing.T) {
	view := shipmentView()
	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpIn, Value: []string{"New", "InTransit"}},
	}
	view.FilterGroups = []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
		{Field: "proNumber", Operator: dbtype.OpContains, Value: "S1"},
		{Field: "proNumber", Operator: dbtype.OpContains, Value: "S2"},
	}}}

	built, err := Build(&reportcatalog.Default, view)

	require.NoError(t, err)
	filters := built.Definition.Filters
	require.NotNil(t, filters)
	assert.Equal(t, report.BoolOpAnd, filters.Op)
	require.Len(t, filters.Filters, 1)
	assert.Equal(t, "status", filters.Filters[0].Ref.Field)
	require.Len(t, filters.Groups, 1)
	assert.Equal(t, report.BoolOpOr, filters.Groups[0].Op)
	assert.Len(t, filters.Groups[0].Filters, 2)
}

func TestBuild_RefusesAFilterTheReportCannotApplyRatherThanDroppingIt(t *testing.T) {
	view := shipmentView()
	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "computedEta", Operator: dbtype.OpEqual, Value: "x"},
	}

	_, err := Build(&reportcatalog.Default, view)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "fieldFilters[0]", validation.Field)
}

func TestBuild_RefusesASearchItCannotExpress(t *testing.T) {
	view := shipmentView()
	view.Query = "acme"

	_, err := Build(&reportcatalog.Default, view)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "query", validation.Field)
}

func TestBuild_SortsOnlyByColumnsItShows(t *testing.T) {
	view := shipmentView()
	view.Sort = []domaintypes.SortField{
		{Field: "status", Direction: dbtype.SortDirectionDesc},
		{Field: "createdAt", Direction: dbtype.SortDirectionAsc},
	}

	built, err := Build(&reportcatalog.Default, view)

	require.NoError(t, err)
	require.Len(t, built.Definition.Sort, 1)
	assert.Equal(t, "c1", built.Definition.Sort[0].ColumnID)
}

func TestBuild_RefusesATableTheCatalogDoesNotKnow(t *testing.T) {
	view := shipmentView()
	view.Resource = permission.Resource("nonexistent")

	_, err := Build(&reportcatalog.Default, view)

	require.Error(t, err)
}
