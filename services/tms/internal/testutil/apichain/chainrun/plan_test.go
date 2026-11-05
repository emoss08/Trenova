package chainrun

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var routes = gin.RoutesInfo{
	{Method: "POST", Path: "/api/v1/shipment-moves/:moveID/assignment/"},
	{Method: "PUT", Path: "/api/v1/shipment-moves/:moveID/assignment/"},
	{Method: "GET", Path: "/api/v1/shipments/:id"},
	{Method: "GET", Path: "/api/v1/shipments/board"},
}

func TestMatchRoute_TakesAReferenceOnlyWhereTheRouteHasAParameter(t *testing.T) {
	t.Parallel()

	got, ok := matchRoute("POST", "/api/v1/shipment-moves/${create.move}/assignment/", routes)
	assert.True(t, ok)
	assert.Equal(t, "/api/v1/shipment-moves/:moveID/assignment/", got)

	_, ok = matchRoute("POST", "/api/v1/shipment-moves/${create.move}/assignment", routes)
	assert.False(t, ok, "the trailing slash is part of the route")

	_, ok = matchRoute("POST", "/api/v1/${x}/smv_1/assignment/", routes)
	assert.False(t, ok, "a reference cannot stand in for a literal segment")
}

func TestMatchRoute_PrefersTheMostLiteralRoute(t *testing.T) {
	t.Parallel()

	got, ok := matchRoute("GET", "/api/v1/shipments/board", routes)

	assert.True(t, ok)
	assert.Equal(t, "/api/v1/shipments/board", got)
}

func TestPrepare_RefusesAnOperationNoClientSends(t *testing.T) {
	t.Parallel()

	chain := &apichain.Chain{Path: "x.yaml", Steps: []apichain.Step{{ID: "a", GraphQL: "Nope"}}}

	_, err := Prepare(chain, &apichain.Manifest{}, routes)

	require.ErrorContains(t, err, "not in the persisted manifest")
}

func TestPrepare_TakesAnInlineDocumentForAnAPIOnlyOperation(t *testing.T) {
	t.Parallel()

	chain := &apichain.Chain{Path: "x.yaml", Steps: []apichain.Step{{
		ID: "a", GraphQL: "Post", Document: "mutation Post($id: ID!) { post(id: $id) { id } }",
	}}}

	plan, err := Prepare(chain, &apichain.Manifest{}, routes)

	require.NoError(t, err)
	assert.False(t, plan.Calls[0].IsRead())
	assert.Equal(t, "graphql Post", plan.Calls[0].Label())
}

func TestPrepare_RefusesAnEffectThatReadsAWrite(t *testing.T) {
	t.Parallel()

	chain := &apichain.Chain{Path: "x.yaml", Steps: []apichain.Step{
		{ID: "w", REST: "PUT /api/v1/shipment-moves/smv/assignment/"},
		{
			ID: "a", REST: "POST /api/v1/shipment-moves/smv/assignment/",
			Effects: []apichain.Effect{{Read: "w", Path: "x", Unchanged: true}},
		},
	}}

	_, err := Prepare(chain, &apichain.Manifest{}, routes)

	require.ErrorContains(t, err, "not a GraphQL query or a GET")
}

func TestCheckEffect(t *testing.T) {
	t.Parallel()

	increase := &apichain.Effect{Increase: "5"}
	require.NoError(t, checkEffect(increase, "5", float64(10), "15.00"))
	require.ErrorContains(t, checkEffect(increase, "5", float64(10), float64(10)), "want 15")

	decrease := &apichain.Effect{Decrease: 2}
	require.NoError(t, checkEffect(decrease, 2, "10", float64(8)))

	unchanged := &apichain.Effect{Unchanged: true}
	require.NoError(t, checkEffect(unchanged, nil, "a", "a"))
	require.Error(t, checkEffect(unchanged, nil, "a", "b"))

	equals := &apichain.Effect{Equals: "1250.00"}
	require.NoError(t, checkEffect(equals, "1250.00", nil, "1250"))
	require.Error(t, checkEffect(equals, "1250.00", nil, "1251"))
}
