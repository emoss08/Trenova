package selectoption

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSealCursors_GivesEachEdgeASealedOffsetCursor(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_")
	result, err := Connection(
		[]Item{
			ItemFor(&gqlmodel.SelectOption{ID: id.String(), Label: "John Smith"}),
			ItemFor(&gqlmodel.SelectOption{ID: pulid.MustNew("wrk_").String(), Label: "Jane Doe"}),
		},
		12,
		10,
	)
	require.NoError(t, err)

	tenant := pagination.TenantInfo{
		OrgID: "org_0000000000000000000000000",
		BuID:  "bu_00000000000000000000000000",
	}
	binding := pagination.BindCursor(tenant, "select_options:WORKER", nil)
	SealCursors(result, binding, 10)

	require.Len(t, result.Edges, 2)
	require.NotNil(t, result.PageInfo.EndCursor)
	assert.Equal(t, result.Edges[1].Cursor, *result.PageInfo.EndCursor)

	offset, err := pagination.DecodeOffsetCursor(binding, result.Edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, 11, offset)

	_, err = pagination.DecodeOffsetCursor(
		pagination.BindCursor(tenant, "select_options:TRACTOR", nil),
		result.Edges[0].Cursor,
	)
	require.ErrorIs(t, err, pagination.ErrCursorInvalid)
}

func TestSelectOptionListConnection_CarriesAnEstimatedTotal(t *testing.T) {
	t.Parallel()

	option := func(label string) Item {
		return ItemFor(&gqlmodel.SelectOption{ID: label, Label: label})
	}

	cases := []struct {
		name     string
		result   *pagination.ListResult[string]
		wantNext bool
	}{
		{
			name:     "a full page past the estimate",
			result:   &pagination.ListResult[string]{Items: []string{"a", "b"}, Total: 10_001},
			wantNext: true,
		},
		{
			name:     "a short page under the estimate",
			result:   &pagination.ListResult[string]{Items: []string{"a"}, Total: 1_250_000},
			wantNext: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			connection, err := ListConnection(
				tc.result,
				pagination.Info{Limit: 2, Offset: 20_000},
				option,
			)
			require.NoError(t, err)

			require.NotNil(t, connection.TotalCount)
			assert.Equal(t, tc.result.Total, *connection.TotalCount)
			assert.False(t, connection.TotalCountExact)
			assert.Equal(t, tc.wantNext, connection.PageInfo.HasNextPage)
		})
	}
}

func TestSelectOptionConnection_ACountedTotalIsExact(t *testing.T) {
	t.Parallel()

	connection, err := Connection(nil, 0, 0)
	require.NoError(t, err)

	require.NotNil(t, connection.TotalCount)
	assert.Equal(t, 0, *connection.TotalCount)
	assert.True(t, connection.TotalCountExact)
	assert.False(t, connection.PageInfo.HasNextPage)
}

func TestSelectOptionInt16Filter(t *testing.T) {
	t.Parallel()

	// GraphQL JSON scalars decode numbers as float64, so that is the shape the
	// planYear filter actually arrives in.
	assert.Equal(
		t,
		int16(2026),
		Int16Filter(map[string]any{"planYear": 2026.0}, "planYear"),
	)
	assert.Equal(
		t,
		int16(2026),
		Int16Filter(map[string]any{"planYear": "2026"}, "planYear"),
	)
	assert.Equal(t, int16(0), Int16Filter(map[string]any{"planYear": true}, "planYear"))
	assert.Equal(t, int16(0), Int16Filter(map[string]any{}, "planYear"))
}

type orderedItemsEntity struct {
	id    pulid.ID
	label string
}

func orderedItemsEntityID(entity *orderedItemsEntity) pulid.ID {
	return entity.id
}

func orderedItemsEntityItem(entity *orderedItemsEntity) Item {
	return ItemFor(&gqlmodel.SelectOption{ID: entity.id.String(), Label: entity.label})
}

func orderedItemLabels(items []Item) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Option.Label)
	}

	return labels
}

func TestOrderedItems_KeepsTheRequestedOrderAndDuplicates(t *testing.T) {
	t.Parallel()

	first := &orderedItemsEntity{id: pulid.MustNew("st_"), label: "first"}
	second := &orderedItemsEntity{id: pulid.MustNew("st_"), label: "second"}

	items := OrderedItems(
		[]pulid.ID{second.id, first.id, second.id},
		[]*orderedItemsEntity{first, second},
		orderedItemsEntityID,
		orderedItemsEntityItem,
	)

	assert.Equal(t, []string{"second", "first", "second"}, orderedItemLabels(items))
}

func TestOrderedItems_DropsAMissingID(t *testing.T) {
	t.Parallel()

	first := &orderedItemsEntity{id: pulid.MustNew("st_"), label: "first"}
	second := &orderedItemsEntity{id: pulid.MustNew("st_"), label: "second"}

	items := OrderedItems(
		[]pulid.ID{second.id, pulid.MustNew("st_"), first.id},
		[]*orderedItemsEntity{first, second},
		orderedItemsEntityID,
		orderedItemsEntityItem,
	)

	assert.Equal(t, []string{"second", "first"}, orderedItemLabels(items))
}

func TestOrderedItems_NothingFoundIsEmpty(t *testing.T) {
	t.Parallel()

	items := OrderedItems(
		[]pulid.ID{pulid.MustNew("st_")},
		nil,
		orderedItemsEntityID,
		orderedItemsEntityItem,
	)

	assert.Empty(t, items)
}
