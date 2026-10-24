package tableexportservice

import (
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func kinds(t *testing.T) *reportaccess.Resolver {
	t.Helper()
	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	resolver, err := reportaccess.New(reportaccess.Params{Catalog: catalog})
	require.NoError(t, err)
	return resolver
}

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
	built, err := Build(kinds(t), shipmentView())

	require.NoError(t, err)
	assert.Equal(t, "shipment", built.Definition.Entity)
	assert.Equal(t, ir.Version, built.Definition.Version)
	require.Len(t, built.Definition.Columns, 2)
	require.NotNil(t, built.Definition.Columns[0].Dimension)
	assert.Equal(t, "proNumber", built.Definition.Columns[0].Dimension.Field)
	assert.Equal(t, "PRO", built.Definition.Columns[0].Label)
	assert.Equal(t, []string{"ETA"}, built.Skipped)
}

func TestBuild_CarriesFiltersAsAndAndEachGroupAsOr(t *testing.T) {
	view := shipmentView()
	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpIn, Value: []string{"New", "InTransit"}},
	}
	view.FilterGroups = []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
		{Field: "proNumber", Operator: dbtype.OpStartsWith, Value: "S1"},
		{Field: "proNumber", Operator: dbtype.OpContains, Value: "S2"},
	}}}

	built, err := Build(kinds(t), view)

	require.NoError(t, err)
	filters := built.Definition.Filters
	require.NotNil(t, filters)
	require.Len(t, filters.And, 2)
	assert.Equal(t, "status", filters.And[0].Field)
	assert.Equal(t, ir.OpIn, filters.And[0].Op)
	require.Len(t, filters.And[1].Or, 2)
	assert.Equal(t, ir.OpStartsWith, filters.And[1].Or[0].Op)
}

func TestBuild_RefusesAFilterTheReportCannotApplyRatherThanDroppingIt(t *testing.T) {
	view := shipmentView()
	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "computedEta", Operator: dbtype.OpEqual, Value: "x"},
	}

	_, err := Build(kinds(t), view)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "fieldFilters[0]", validation.Field)
}

func TestBuild_RefusesASearchItCannotExpress(t *testing.T) {
	view := shipmentView()
	view.Query = "acme"

	_, err := Build(kinds(t), view)

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

	built, err := Build(kinds(t), view)

	require.NoError(t, err)
	require.Len(t, built.Definition.Sort, 1)
	assert.Equal(t, ir.Sort{Column: "c1", Desc: true}, built.Definition.Sort[0])
}

func TestBuild_RefusesATableTheCatalogDoesNotKnow(t *testing.T) {
	view := shipmentView()
	view.Resource = permission.Resource("nonexistent")

	_, err := Build(kinds(t), view)

	require.Error(t, err)
}

func TestBuild_ResolvesNestedColumnsAndReferences(t *testing.T) {
	built, err := Build(kinds(t), &View{
		Resource: permission.ResourceShipment,
		Columns: []Column{
			{Field: "proNumber", Label: "PRO"},
			{Field: "customer.name", Label: "Customer"},
			{Field: "customerId", Label: "Customer ID"},
			{Field: "customer.proNumber", Label: "Customer PRO"},
			{Field: "moves.status", Label: "Move status"},
		},
	})

	require.NoError(t, err)
	require.Len(t, built.Definition.Columns, 3)
	assert.Equal(t, "proNumber", built.Definition.Columns[0].Dimension.Field)
	assert.Equal(t, "customer.name", built.Definition.Columns[1].Dimension.Field)
	assert.Equal(t, "customer.id", built.Definition.Columns[2].Dimension.Field)
	assert.Equal(t, []string{"Customer PRO", "Move status"}, built.Skipped)
}

func TestBuild_ResolvesANestedFilterOnTheEntityItsPathReaches(t *testing.T) {
	view := shipmentView()
	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "customer.name", Operator: dbtype.OpContains, Value: "Acme"},
	}

	built, err := Build(kinds(t), view)
	require.NoError(t, err)
	require.NotNil(t, built.Definition.Filters)
	require.Len(t, built.Definition.Filters.And, 1)
	assert.Equal(t, "customer.name", built.Definition.Filters.And[0].Field)

	view.FieldFilters = []domaintypes.FieldFilter{
		{Field: "customer.proNumber", Operator: dbtype.OpEqual, Value: "P-1"},
	}
	_, err = Build(kinds(t), view)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
}
