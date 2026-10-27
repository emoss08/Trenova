//go:build integration

package equipmenttyperepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEquipmentTypeRepository(t *testing.T) {
	db := equipmenttest.NewDatabase(t)
	a := db.A
	repo := equipmenttyperepository.New(
		equipmenttyperepository.Params{DB: db.Conn, Logger: zap.NewNop()},
	)

	created := make(map[string]*equipmenttype.EquipmentType)
	for code, class := range map[string]equipmenttype.Class{
		"DRY":  equipmenttype.ClassTrailer,
		"REEF": equipmenttype.ClassTrailer,
		"DAY":  equipmenttype.ClassTractor,
	} {
		entity, err := repo.Create(a.Ctx, &equipmenttype.EquipmentType{
			Status: domaintypes.StatusActive,
			Code:   code,
			Class:  class,
		})
		require.NoError(t, err)
		created[code] = entity
	}
	_, err := repo.Create(db.B.Ctx, &equipmenttype.EquipmentType{
		Status: domaintypes.StatusActive,
		Code:   "DRY",
		Class:  equipmenttype.ClassTrailer,
	})
	require.NoError(t, err)

	t.Run("a duplicate code in the tenant is a validation error", func(t *testing.T) {
		_, err := repo.Create(a.Ctx, &equipmenttype.EquipmentType{
			Status: domaintypes.StatusActive,
			Code:   "dry",
			Class:  equipmenttype.ClassTrailer,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "code", validation.Field)
		assert.Equal(t, errortypes.ErrDuplicate, validation.Code)
	})

	t.Run("the class filter narrows the connection and its count", func(t *testing.T) {
		cursor, err := pagination.NewCursorInfo(10, "")
		require.NoError(t, err)

		result, err := repo.List(a.Ctx, &equipmenttypeports.ListEquipmentTypesRequest{
			Filter:  &pagination.QueryOptions{},
			Cursor:  cursor,
			Classes: []string{string(equipmenttype.ClassTrailer), ""},
		})
		require.NoError(t, err)
		require.NotNil(t, result.TotalCount)
		assert.Equal(t, 2, *result.TotalCount)
		assert.Len(t, result.Items, 2)
	})

	t.Run("select options read the active types of the classes asked for", func(t *testing.T) {
		result, err := repo.SelectOptions(
			a.Ctx,
			&equipmenttypeports.EquipmentTypeSelectOptionsRequest{
				SelectQueryRequest: &pagination.SelectQueryRequest{
					Query:      "re",
					Pagination: pagination.Info{Limit: 10},
				},
				Classes: []string{string(equipmenttype.ClassTrailer)},
			},
		)
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "REEF", result.Items[0].Code)
	})

	t.Run("bulk status updates return the rows and bump their version", func(t *testing.T) {
		ids := []pulid.ID{created["DRY"].ID, created["DAY"].ID}
		updated, err := repo.BulkUpdateStatus(
			a.Ctx,
			&equipmenttypeports.BulkUpdateEquipmentTypeStatusRequest{
				EquipmentTypeIDs: ids,
				Status:           domaintypes.StatusInactive,
			},
		)
		require.NoError(t, err)
		require.Len(t, updated, 2)
		for _, item := range updated {
			assert.Equal(t, domaintypes.StatusInactive, item.Status)
			assert.Equal(t, int64(1), item.Version)
		}

		got, err := repo.GetByIDs(
			a.Ctx,
			equipmenttypeports.GetEquipmentTypesByIDsRequest{EquipmentTypeIDs: ids},
		)
		require.NoError(t, err)
		assert.Len(t, got, 2)

		stale := *created["DRY"]
		stale.Description = "stale"
		_, err = repo.Update(a.Ctx, &stale)
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, errortypes.ErrVersionMismatch, validation.Code)
	})

	t.Run("get by id does not cross tenants", func(t *testing.T) {
		_, err := repo.GetByID(
			db.B.Ctx,
			equipmenttypeports.GetEquipmentTypeByIDRequest{ID: created["REEF"].ID},
		)
		assert.True(t, errortypes.IsNotFoundError(err))
	})
}
