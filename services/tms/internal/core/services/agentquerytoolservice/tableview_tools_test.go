package agentquerytoolservice

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The answer to "show me a table of ..." is a view, not rows.

Rows pasted into a conversation are a snapshot nobody can act on: not
sortable, not exportable, not re-checked against permissions, and wrong by the
time anybody reads them. A link opens the real table, live.

The link is built here rather than by the model for the same reason the
filters are compiled rather than trusted: a fabricated path is
indistinguishable from a real one until somebody clicks it.
*/

type stubComposer struct {
	result *tablequeryservice.ComposeResult
	saw    *tablequeryservice.ComposeRequest
}

func (s *stubComposer) Compose(
	_ context.Context,
	req *tablequeryservice.ComposeRequest,
) (*tablequeryservice.ComposeResult, error) {
	s.saw = req

	return s.result, nil
}

func viewCatalog() *filtercatalog.Catalog {
	return filtercatalog.New(filtercatalog.Resource{
		Tool:     "list_shipments",
		Resource: permission.ResourceShipment,
		Entity:   "shipments",
		Fields: []filtercatalog.Field{
			{Name: "status", Kind: filtercatalog.KindEnum, Values: []string{"InTransit"}},
		},
	})
}

func viewTool(result *tablequeryservice.ComposeResult) (*composeTableViewTool, *stubComposer) {
	composer := &stubComposer{result: result}

	return &composeTableViewTool{composer: composer, catalog: viewCatalog()}, composer
}

func composed() *tablequeryservice.ComposeResult {
	return &tablequeryservice.ComposeResult{
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "status", Operator: dbtype.OpEqual, Value: "InTransit"},
		},
		Explanation: "shipments where status equals InTransit",
		Terms:       []string{"status equals InTransit"},
	}
}

func compose(t *testing.T, tool *composeTableViewTool, params map[string]any) tableViewResult {
	t.Helper()
	result, err := tool.Query(t.Context(), testParams(params))
	require.NoError(t, err)
	view, ok := result.(tableViewResult)
	require.True(t, ok)

	return view
}

func TestComposeTableView_GivesBackALinkThatOpensTheTable(t *testing.T) {
	t.Parallel()

	tool, _ := viewTool(composed())
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	assert.Equal(t, "shipments", view.Entity)
	assert.True(
		t,
		strings.HasPrefix(view.Path, "/shipment-management/shipments?"),
		"the link opens the page the router serves shipments on; got %q",
		view.Path,
	)
	assert.Equal(t, 1, view.FilterCount)
	assert.Equal(t, "shipments where status equals InTransit", view.Explanation)
}

// The filters ride in the query parameters the data table already reads, so
// the link lands on a table that is filtered rather than one that is not.
func TestComposeTableView_PutsTheFiltersInTheLink(t *testing.T) {
	t.Parallel()

	tool, _ := viewTool(composed())
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	parsed, err := url.Parse(view.Path)
	require.NoError(t, err)
	filters := parsed.Query().Get("fieldFilters")
	assert.Contains(t, filters, "status")
	assert.Contains(t, filters, "InTransit")
}

// A view that quietly lost a condition looks like an answer, so what the
// table could not express travels with it.
func TestComposeTableView_CarriesWhatTheTableCouldNotExpress(t *testing.T) {
	t.Parallel()

	result := composed()
	result.Unresolved = []tablequeryservice.Unresolved{
		{Phrase: "the profitable ones", Reason: "margin is not a field here"},
	}
	tool, _ := viewTool(result)

	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "in transit and profitable",
	})

	require.Len(t, view.Unresolved, 1)
	assert.Equal(t, "the profitable ones", view.Unresolved[0].Phrase)
}

// An unfiltered view is a bare path, not a path with empty parameters hanging
// off it.
func TestComposeTableView_LeavesTheLinkBareWhenNothingNarrowedIt(t *testing.T) {
	t.Parallel()

	tool, _ := viewTool(&tablequeryservice.ComposeResult{
		Explanation: "Every shipments, unfiltered.",
	})
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "everything",
	})

	assert.Equal(t, "/shipment-management/shipments", view.Path)
	assert.Equal(t, 0, view.FilterCount)
}

func TestComposeTableView_PassesTheDescriptionAndTheActorThrough(t *testing.T) {
	t.Parallel()

	tool, composer := viewTool(composed())
	compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	require.NotNil(t, composer.saw)
	assert.Equal(t, "still in transit", composer.saw.Prompt)
	assert.Equal(t, permission.ResourceShipment, composer.saw.Resource)
	// The composer checks the read against the person, so it needs the person.
	require.NotNil(t, composer.saw.Actor)
}

// A table nothing can narrow is refused by name, with the ones that can
// listed, rather than being composed against an empty vocabulary.
func TestComposeTableView_RefusesATableItHasNoCatalogueFor(t *testing.T) {
	t.Parallel()

	tool, composer := viewTool(composed())

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		"entity":      "unicorns",
		"description": "the sparkly ones",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shipments")
	assert.Nil(t, composer.saw, "the composer ran for a table that does not exist")
}

// The schema names the tables the catalogue actually holds, so a provider
// that enforces it cannot ask for one that is not there.
func TestComposeTableView_OffersOnlyTheTablesItCanBuild(t *testing.T) {
	t.Parallel()

	tool, _ := viewTool(composed())
	schema := tool.ParamSchema()

	properties, _ := schema["properties"].(map[string]any)
	entity, _ := properties["entity"].(map[string]any)
	assert.Equal(t, []string{"shipments"}, entity["enum"])
}

/*
A view is run once through the table's own list, so the card beside the
conversation can say how many rows opening it will show and what the first
few are. The fake list below stands in for list_shipments: what matters here
is that the view's own filters reach the fetch, and that the count stops at
its limit rather than reading the whole table.
*/

type viewRow struct {
	ID        string `json:"id"`
	ProNumber string `json:"proNumber"`
	Status    string `json:"status"`
}

func viewRows(n int) []any {
	rows := make([]any, 0, n)
	for i := range n {
		n := strconv.Itoa(i)
		rows = append(rows, &viewRow{ID: "shp_" + n, ProNumber: "P" + n, Status: "InTransit"})
	}

	return rows
}

func runnableViewTool(rows []any) (*composeTableViewTool, *capturedList) {
	capture := &capturedList{rows: rows}
	list := buildListTool(&listSpec{
		name:         "list_shipments",
		entityPlural: "shipments",
		summary:      "List shipments.",
		resource:     permission.ResourceShipment,
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: []string{"InTransit"}},
		},
		fetch: capture.fetch,
	})
	tool, _ := viewTool(composed())
	tool.lists = map[permission.Resource]*listTool{permission.ResourceShipment: list}

	return tool, capture
}

func TestComposeTableView_CountsAndPreviewsTheViewThroughItsList(t *testing.T) {
	t.Parallel()

	tool, capture := runnableViewTool(viewRows(3))
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	require.NotNil(t, view.Count)
	assert.Equal(t, 3, *view.Count)
	assert.False(t, view.CountCapped)
	assert.Equal(t, []string{"id", "proNumber", "status"}, view.Columns)
	assert.Len(t, view.Items, 3)

	// The list read the view's own filters, and one row past the count limit
	// so it can say whether there are more.
	require.NotNil(t, capture.opts)
	assert.Equal(t, composed().FieldFilters, capture.opts.FieldFilters)
	assert.Equal(t, viewCountLimit+1, capture.opts.Pagination.Limit)
}

// Past the limit the count stops and says so; the preview is only the first
// few rows, never the hundred it counted.
func TestComposeTableView_StopsCountingAtTheLimit(t *testing.T) {
	t.Parallel()

	tool, _ := runnableViewTool(viewRows(viewCountLimit + 1))
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	require.NotNil(t, view.Count)
	assert.Equal(t, viewCountLimit, *view.Count)
	assert.True(t, view.CountCapped)
	assert.Len(t, view.Items, viewPreviewRows)
}

// A table with no list to run the view through still gets its view; it just
// has nothing to count.
func TestComposeTableView_LeavesTheCountOutWithoutAList(t *testing.T) {
	t.Parallel()

	tool, _ := viewTool(composed())
	view := compose(t, tool, map[string]any{
		"entity":      "shipments",
		"description": "still in transit",
	})

	assert.Nil(t, view.Count)
	assert.Nil(t, view.Items)
	assert.NotEmpty(t, view.Path)
}
