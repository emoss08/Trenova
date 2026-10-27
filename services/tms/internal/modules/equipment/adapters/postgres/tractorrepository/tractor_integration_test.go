//go:build integration

package tractorrepository_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTractorRepository(t *testing.T) {
	db := equipmenttest.NewDevelopmentDatabase(t)
	a := db.A
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := tractorrepository.New(
		tractorrepository.Params{
			DB:        db.Conn,
			Logger:    zap.NewNop(),
			Quota:     guard,
			Locations: crossref.LocationColumns(),
			Moves:     crossref.MoveColumns(),
		},
	)

	total := int(equipmenttest.Value[int64](t, db,
		"SELECT count(*) FROM tractors WHERE organization_id = $1 AND business_unit_id = $2",
		a.Info.OrgID.String(), a.Info.BuID.String()))
	require.Positive(t, total)

	var first *tractor.Tractor

	t.Run("a connection reads every tractor of the tenant with its workers", func(t *testing.T) {
		cursor, err := pagination.NewCursorInfo(100, "")
		require.NoError(t, err)

		result, err := repo.List(a.Ctx, &tractorports.ListTractorsRequest{
			Filter: &pagination.QueryOptions{Sort: []domaintypes.SortField{
				{Field: "code", Direction: dbtype.SortDirectionAsc},
			}},
			Cursor: cursor,
			TractorRelationIncludes: tractorports.TractorRelationIncludes{
				IncludeWorkerDetails:        true,
				IncludePrimaryWorkerManager: true,
				IncludeEquipmentDetails:     true,
			},
		})
		require.NoError(t, err)

		require.NotNil(t, result.TotalCount)
		assert.Equal(t, total, *result.TotalCount)
		require.Len(t, result.Items, total)
		for i, item := range result.Items {
			require.NotNil(t, item.PrimaryWorker, item.Code)
			assert.Equal(t, item.PrimaryWorkerID, item.PrimaryWorker.ID)
			require.NotNil(t, item.EquipmentType)
			if i > 0 {
				assert.LessOrEqual(t, result.Items[i-1].Code, item.Code)
			}
		}
		first = result.Items[0]
	})

	t.Run("the status filter and search narrow the connection", func(t *testing.T) {
		cursor, err := pagination.NewCursorInfo(100, "")
		require.NoError(t, err)

		result, err := repo.List(a.Ctx, &tractorports.ListTractorsRequest{
			Filter: &pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{
				{Field: "code", Operator: dbtype.OpEqual, Value: first.Code},
			}},
			Cursor: cursor,
			Status: string(first.Status),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, first.ID, result.Items[0].ID)

		result, err = repo.List(a.Ctx, &tractorports.ListTractorsRequest{
			Filter: &pagination.QueryOptions{},
			Cursor: cursor,
			Status: string(domaintypes.EquipmentStatusSold),
		})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("get by codes matches codes and plates ignoring case", func(t *testing.T) {
		got, err := repo.GetByCodes(a.Ctx, tractorports.GetTractorsByCodesRequest{
			Codes: []string{" " + strings.ToLower(first.Code) + " ", "NO-SUCH-TRACTOR"},
		})
		require.NoError(t, err)
		require.Contains(t, got, strings.ToUpper(first.Code))
		assert.Equal(t, first.ID, got[strings.ToUpper(first.Code)].ID)
		assert.Len(t, got, 1)

		none, err := repo.GetByCodes(a.Ctx, tractorports.GetTractorsByCodesRequest{})
		require.NoError(t, err)
		assert.Empty(t, none)
	})

	t.Run("get by id reads the requested details", func(t *testing.T) {
		got, err := repo.GetByID(a.Ctx, tractorports.GetTractorByIDRequest{
			ID:                      first.ID,
			TractorRelationIncludes: tractorports.FullTractorRelationIncludes(),
		})
		require.NoError(t, err)
		require.NotNil(t, got.PrimaryWorker)
		require.NotNil(t, got.Organization)
		require.NotNil(t, got.Organization.State)
		require.NotNil(t, got.EquipmentManufacturer)
	})

	t.Run("a create runs inside the quota check", func(t *testing.T) {
		guard.EXPECT().Enforce(mock.Anything, mock.Anything).Return(nil).Once()

		created, err := repo.Create(a.Ctx, &tractor.Tractor{
			PrimaryWorkerID: pulid.ID(equipmenttest.Value[string](t, db, `SELECT w.id FROM workers w
				WHERE w.organization_id = $1 AND w.business_unit_id = $2
				AND NOT EXISTS (SELECT 1 FROM tractors t WHERE t.primary_worker_id = w.id) ORDER BY w.id LIMIT 1`,
				a.Info.OrgID.String(), a.Info.BuID.String())),
			EquipmentTypeID:         first.EquipmentTypeID,
			EquipmentManufacturerID: first.EquipmentManufacturerID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "KEEL-1",
			FuelType:                first.FuelType,
			OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
		})
		require.NoError(t, err)
		assert.True(t, created.ID.IsNotNil())

		got, err := repo.GetByIDs(a.Ctx, tractorports.GetTractorsByIDsRequest{
			TractorIDs: []pulid.ID{created.ID, first.ID},
		})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}
