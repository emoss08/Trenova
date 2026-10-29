package selectoption

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectOptionConnection_UsesOpaqueEntityCursors(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_")
	createdAt := int64(1780415883)

	result, err := Connection(
		[]Item{
			{
				Option: &gqlmodel.SelectOption{
					ID:    id.String(),
					Label: "John Smith",
				},
				Cursor: pagination.PositionCursor{
					At: createdAt,
					ID: id,
				},
			},
		},
		1,
		0,
	)
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.NotEqual(t, "1", result.Edges[0].Cursor)
	require.NotNil(t, result.PageInfo.EndCursor)
	assert.Equal(t, result.Edges[0].Cursor, *result.PageInfo.EndCursor)

	decoded, err := pagination.DecodePositionCursor(result.Edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, createdAt, decoded.At)
	assert.Equal(t, id, decoded.ID)
}

func TestSelectOptionListConnection_CarriesAnEstimatedTotal(t *testing.T) {
	t.Parallel()

	option := func(label string) Item {
		return Item{
			Option: &gqlmodel.SelectOption{ID: label, Label: label},
			Cursor: pagination.PositionCursor{At: 1780415883, ID: pulid.MustNew("wrk_")},
		}
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
	return ItemFor(
		&gqlmodel.SelectOption{ID: entity.id.String(), Label: entity.label},
		0,
		entity.id,
	)
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
