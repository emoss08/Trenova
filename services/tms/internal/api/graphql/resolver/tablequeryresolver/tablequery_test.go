package tablequeryresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeTableQuery_ChecksReadOnTheNamedTable(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}

	_, err := (&QueryResolver{deps}).ComposeTableQuery(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		gqlmodel.ComposeTableQueryInput{Resource: "shipment", Prompt: "late loads"},
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceShipment, permission.OpRead)
}

func TestCurrentViewFromInput_KeepsWhatTheTableShows(t *testing.T) {
	t.Parallel()

	query := "acme"
	view := currentViewFromInput(&gqlmodel.TableQueryCurrentViewInput{
		Query: &query,
		FieldFilters: []*gqlmodel.FieldFilterInput{
			{Field: "status", Operator: "eq", Value: "New"},
		},
		Sort: []*gqlmodel.SortFieldInput{{Field: "createdAt", Direction: "desc"}},
	})

	assert.Equal(t, "acme", view.Query)
	require.Len(t, view.FieldFilters, 1)
	assert.Equal(t, "status", view.FieldFilters[0].Field)
	assert.Equal(t, dbtype.Operator("eq"), view.FieldFilters[0].Operator)
	require.Len(t, view.Sort, 1)
	assert.Equal(t, "createdAt", view.Sort[0].Field)
	assert.Empty(t, currentViewFromInput(nil).FieldFilters)
}

func TestCatalogueToModel_ListsEachTableAndItsTerms(t *testing.T) {
	t.Parallel()

	catalog := filtercatalog.New(filtercatalog.Resource{
		Resource: permission.ResourceShipment,
		Entity:   "shipments",
		Summary:  "Loads",
		Fields: []filtercatalog.Field{
			{Name: "status", Kind: filtercatalog.KindEnum, Values: []string{"New"}, Sortable: true},
		},
	})

	resources := catalogueToModel(catalog)

	require.Len(t, resources, 1)
	assert.Equal(t, "shipment", resources[0].Resource)
	assert.Equal(t, "shipments", resources[0].Entity)
	require.Len(t, resources[0].Fields, 1)
	assert.Equal(t, "status", resources[0].Fields[0].Name)
	assert.Equal(t, []string{"New"}, resources[0].Fields[0].Values)
	assert.True(t, resources[0].Fields[0].Sortable)
	assert.NotEmpty(t, resources[0].Fields[0].Operators)
}

func TestTableQueryFilterAndSort_RenderTheirOperators(t *testing.T) {
	t.Parallel()

	operator, err := (&TableQueryFieldFilterResolver{&Deps{}}).Operator(
		t.Context(),
		&domaintypes.FieldFilter{Operator: dbtype.Operator("contains")},
	)
	require.NoError(t, err)
	assert.Equal(t, "contains", operator)

	direction, err := (&TableQuerySortFieldResolver{&Deps{}}).Direction(
		t.Context(),
		&domaintypes.SortField{Direction: dbtype.SortDirection("asc")},
	)
	require.NoError(t, err)
	assert.Equal(t, "asc", direction)
}

func TestTableQueryCatalogue_RequiresASignedInCaller(t *testing.T) {
	t.Parallel()

	_, err := (&QueryResolver{&Deps{Core: &resolvercore.Core{}}}).TableQueryCatalogue(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
