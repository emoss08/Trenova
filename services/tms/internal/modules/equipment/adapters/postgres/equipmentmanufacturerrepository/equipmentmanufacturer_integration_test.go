//go:build integration

package equipmentmanufacturerrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEquipmentManufacturerRepository(t *testing.T) {
	db := equipmenttest.NewDatabase(t)
	a := db.A
	repo := equipmentmanufacturerrepository.New(
		equipmentmanufacturerrepository.Params{DB: db.Conn, Logger: zap.NewNop()},
	)

	ids := make([]pulid.ID, 0, 3)
	for _, name := range []string{"Freightliner", "Kenworth", "Peterbilt"} {
		entity, err := repo.Create(a.Ctx, &equipmentmanufacturer.EquipmentManufacturer{
			Status: domaintypes.StatusActive,
			Name:   name,
		})
		require.NoError(t, err)
		ids = append(ids, entity.ID)
	}

	t.Run("a duplicate name is a validation error on the name", func(t *testing.T) {
		_, err := repo.Create(a.Ctx, &equipmentmanufacturer.EquipmentManufacturer{
			Status: domaintypes.StatusActive,
			Name:   "KENWORTH",
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "name", validation.Field)
	})

	t.Run("an offset list sorts, pages and counts", func(t *testing.T) {
		result, err := repo.List(
			a.Ctx,
			&equipmentmanufacturerports.ListEquipmentManufacturersRequest{
				Filter: &pagination.QueryOptions{
					Pagination: pagination.Info{Limit: 2},
					Sort: []domaintypes.SortField{
						{Field: "name", Direction: dbtype.SortDirectionDesc},
					},
				},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Total)
		require.Len(t, result.Items, 2)
		assert.Equal(t, "Peterbilt", result.Items[0].Name)
		assert.Equal(t, "Kenworth", result.Items[1].Name)
	})

	t.Run("an offset list ranks search matches", func(t *testing.T) {
		result, err := repo.List(
			a.Ctx,
			&equipmentmanufacturerports.ListEquipmentManufacturersRequest{
				Filter: &pagination.QueryOptions{
					Query:      "pete",
					Pagination: pagination.Info{Limit: 10},
				},
			},
		)
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "Peterbilt", result.Items[0].Name)
	})

	t.Run("a connection pages by name", func(t *testing.T) {
		cursor, err := pagination.NewCursorInfo(2, "")
		require.NoError(t, err)
		filter := &pagination.QueryOptions{Sort: []domaintypes.SortField{
			{Field: "name", Direction: dbtype.SortDirectionAsc},
		}}

		first, err := repo.ListConnection(
			a.Ctx,
			&equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest{
				Filter: filter,
				Cursor: cursor,
			},
		)
		require.NoError(t, err)
		require.Len(t, first.Items, 2)
		assert.Equal(t, "Freightliner", first.Items[0].Name)

		next, err := first.NextCursor()
		require.NoError(t, err)
		cursor, err = pagination.NewCursorInfo(2, next)
		require.NoError(t, err)
		second, err := repo.ListConnection(
			a.Ctx,
			&equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest{
				Filter: filter,
				Cursor: cursor,
			},
		)
		require.NoError(t, err)
		require.Len(t, second.Items, 1)
		assert.Equal(t, "Peterbilt", second.Items[0].Name)
		assert.False(t, second.HasNextPage)
	})

	t.Run("bulk status updates and get by ids", func(t *testing.T) {
		updated, err := repo.BulkUpdateStatus(
			a.Ctx,
			&equipmentmanufacturerports.BulkUpdateEquipmentManufacturerStatusRequest{
				EquipmentManufacturerIDs: ids[:2],
				Status:                   domaintypes.StatusInactive,
			},
		)
		require.NoError(t, err)
		assert.Len(t, updated, 2)

		options, err := repo.SelectOptions(
			a.Ctx,
			&pagination.SelectQueryRequest{Pagination: pagination.Info{Limit: 10}},
		)
		require.NoError(t, err)
		assert.Equal(t, 1, options.Total)

		got, err := repo.GetByIDs(
			db.B.Ctx,
			equipmentmanufacturerports.GetEquipmentManufacturersByIDsRequest{
				EquipmentManufacturerIDs: ids,
			},
		)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
