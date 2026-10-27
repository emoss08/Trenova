//go:build integration

package trailerrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/trailerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTrailerRepository(t *testing.T) {
	db := equipmenttest.NewDevelopmentDatabase(t)
	a := db.A
	repo := trailerrepository.New(trailerrepository.Params{
		DB:        db.Conn,
		Logger:    zap.NewNop(),
		Locations: crossref.LocationColumns(),
	})

	trailerID := pulid.ID(equipmenttest.Value[string](
		t,
		db,
		"SELECT id FROM trailers WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
		a.Info.OrgID.String(),
		a.Info.BuID.String(),
	))

	t.Run("get by id reads the owner operator and where the trailer was left", func(t *testing.T) {
		workerID := equipmenttest.Value[string](
			t,
			db,
			"SELECT id FROM workers WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
			a.Info.OrgID.String(),
			a.Info.BuID.String(),
		)
		locationID := equipmenttest.Value[string](
			t,
			db,
			"SELECT id FROM locations WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
			a.Info.OrgID.String(),
			a.Info.BuID.String(),
		)

		db.Exec(t, "UPDATE trailers SET ownership_type = $1, owner_worker_id = $2 WHERE id = $3",
			string(domaintypes.OwnershipTypeOwnerOperator), workerID, trailerID.String())
		db.Exec(
			t,
			"UPDATE locations SET latitude = 35.4676, longitude = -97.5164 WHERE id = $1",
			locationID,
		)
		db.Exec(
			t,
			`INSERT INTO equipment_continuity
			(id, organization_id, business_unit_id, equipment_type, equipment_id, current_location_id, source_type, is_current)
			VALUES ($1, $2, $3, 'Trailer', $4, $5, 'ManualLocate', TRUE)`,
			pulid.MustNew("eqc_").
				String(),
			a.Info.OrgID.String(),
			a.Info.BuID.String(),
			trailerID.String(),
			locationID,
		)

		got, err := repo.GetByID(a.Ctx, trailerports.GetTrailerByIDRequest{
			ID: trailerID,
			TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
				IncludeOwnerWorker:       true,
				IncludeLastKnownLocation: true,
				IncludeTenantDetails:     true,
			},
		})
		require.NoError(t, err)

		require.NotNil(t, got.OwnerWorker)
		assert.Equal(t, pulid.ID(workerID), got.OwnerWorker.ID)
		assert.Equal(t, pulid.ID(locationID), got.LastKnownLocationID)
		assert.NotEmpty(t, got.LastKnownLocationName)
		require.NotNil(t, got.LastKnownLocationLatitude)
		assert.InDelta(t, 35.4676, *got.LastKnownLocationLatitude, 0.0001)
		require.NotNil(t, got.LastKnownLocationLongitude)
		assert.InDelta(t, -97.5164, *got.LastKnownLocationLongitude, 0.0001)
		require.NotNil(t, got.Organization)
		require.NotNil(t, got.Organization.BusinessUnit)
		require.NotNil(t, got.BusinessUnit)
		assert.Equal(t, a.Info.BuID, got.BusinessUnit.ID)
	})

	t.Run("a trailer that was never located reads no location", func(t *testing.T) {
		other := pulid.ID(equipmenttest.Value[string](
			t,
			db,
			"SELECT id FROM trailers WHERE organization_id = $1 AND business_unit_id = $2 AND id <> $3 ORDER BY id LIMIT 1",
			a.Info.OrgID.String(),
			a.Info.BuID.String(),
			trailerID.String(),
		))

		got, err := repo.GetByID(a.Ctx, trailerports.GetTrailerByIDRequest{
			ID: other,
			TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
				IncludeLastKnownLocation: true,
			},
		})
		require.NoError(t, err)
		assert.True(t, got.LastKnownLocationID.IsNil())
		assert.Empty(t, got.LastKnownLocationName)
		assert.Nil(t, got.LastKnownLocationLatitude)
	})

	t.Run("a connection filters through a queryable relation", func(t *testing.T) {
		code := equipmenttest.Value[string](t, db,
			`SELECT et.code FROM trailers tr JOIN equipment_types et ON et.id = tr.equipment_type_id
			 WHERE tr.id = $1`, trailerID.String())
		want := equipmenttest.Value[int64](
			t,
			db,
			`SELECT count(*) FROM trailers tr JOIN equipment_types et ON et.id = tr.equipment_type_id
			 WHERE tr.organization_id = $1 AND tr.business_unit_id = $2 AND et.code = $3`,
			a.Info.OrgID.String(),
			a.Info.BuID.String(),
			code,
		)

		cursor, err := pagination.NewCursorInfo(100, "")
		require.NoError(t, err)
		result, err := repo.List(a.Ctx, &trailerports.ListTrailersRequest{
			Filter: &pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{
				{Field: "equipmentType.code", Operator: dbtype.OpEqual, Value: code},
			}},
			Cursor: cursor,
			TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
				IncludeEquipmentType: true,
			},
		})
		require.NoError(t, err)

		require.NotNil(t, result.TotalCount)
		assert.Equal(t, int(want), *result.TotalCount)
		require.Len(t, result.Items, int(want))
		for _, item := range result.Items {
			require.NotNil(t, item.EquipmentType)
			assert.Equal(t, code, item.EquipmentType.Code)
		}
	})

	t.Run("get by ids and bulk status updates read the rows they touch", func(t *testing.T) {
		ids := make([]pulid.ID, 0, 2)
		for _, id := range []string{
			trailerID.String(),
			equipmenttest.Value[string](t, db,
				"SELECT id FROM trailers WHERE organization_id = $1 AND id <> $2 ORDER BY id DESC LIMIT 1",
				a.Info.OrgID.String(), trailerID.String()),
		} {
			ids = append(ids, pulid.ID(id))
		}

		got, err := repo.GetByIDs(a.Ctx, trailerports.GetTrailersByIDsRequest{TrailerIDs: ids})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		updated, err := repo.BulkUpdateStatus(a.Ctx, &trailerports.BulkUpdateTrailerStatusRequest{
			TrailerIDs: ids,
			Status:     domaintypes.EquipmentStatusOOS,
		})
		require.NoError(t, err)
		require.Len(t, updated, 2)
		for _, item := range updated {
			assert.Equal(t, domaintypes.EquipmentStatusOOS, item.Status)
		}

		_, err = repo.BulkUpdateStatus(a.Ctx, &trailerports.BulkUpdateTrailerStatusRequest{
			TrailerIDs: []pulid.ID{pulid.MustNew("tr_")},
			Status:     domaintypes.EquipmentStatusAvailable,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, errortypes.ErrVersionMismatch, validation.Code)
	})

	t.Run("update saves and reads the trailer back", func(t *testing.T) {
		current, err := repo.GetByID(a.Ctx, trailerports.GetTrailerByIDRequest{ID: trailerID})
		require.NoError(t, err)

		current.Model = "Updated model"
		updated, err := repo.Update(a.Ctx, current)
		require.NoError(t, err)
		assert.Equal(t, "Updated model", updated.Model)
		assert.Equal(t, current.Version, updated.Version)
	})

	t.Run("a duplicate code is a validation error on the code", func(t *testing.T) {
		existing, err := repo.GetByID(a.Ctx, trailerports.GetTrailerByIDRequest{ID: trailerID})
		require.NoError(t, err)

		duplicate := &trailer.Trailer{
			EquipmentTypeID:         existing.EquipmentTypeID,
			EquipmentManufacturerID: existing.EquipmentManufacturerID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    existing.Code,
			OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
		}
		_, err = repo.Create(a.Ctx, duplicate)
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "code", validation.Field)
	})
}
