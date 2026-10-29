package selectoption

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/errortypes"
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

type requiredItemsEntity struct {
	id    pulid.ID
	label string
}

func requiredItemsEntityID(entity *requiredItemsEntity) pulid.ID {
	return entity.id
}

func requiredItemsEntityItem(entity *requiredItemsEntity) Item {
	return ItemFor(
		&gqlmodel.SelectOption{ID: entity.id.String(), Label: entity.label},
		0,
		entity.id,
	)
}

func TestRequiredItems_KeepsTheRequestedOrder(t *testing.T) {
	t.Parallel()

	first := &requiredItemsEntity{id: pulid.MustNew("st_"), label: "first"}
	second := &requiredItemsEntity{id: pulid.MustNew("st_"), label: "second"}

	items, err := RequiredItems(
		[]pulid.ID{second.id, first.id, second.id},
		[]*requiredItemsEntity{first, second},
		requiredItemsEntityID,
		requiredItemsEntityItem,
		"ShipmentType",
	)
	require.NoError(t, err)

	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Option.Label)
	}
	assert.Equal(t, []string{"second", "first", "second"}, labels)
}

func TestRequiredItems_AMissingIDIsNotFound(t *testing.T) {
	t.Parallel()

	found := &requiredItemsEntity{id: pulid.MustNew("st_"), label: "found"}

	items, err := RequiredItems(
		[]pulid.ID{found.id, pulid.MustNew("st_")},
		[]*requiredItemsEntity{found},
		requiredItemsEntityID,
		requiredItemsEntityItem,
		"ShipmentType",
	)
	require.Error(t, err)
	assert.Nil(t, items)
	assert.True(t, errortypes.IsNotFoundError(err))
}
