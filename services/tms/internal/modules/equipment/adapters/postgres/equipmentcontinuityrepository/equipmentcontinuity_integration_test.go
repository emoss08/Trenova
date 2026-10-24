//go:build integration

package equipmentcontinuityrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentcontinuityrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEquipmentContinuityRepository(t *testing.T) {
	db := equipmenttest.NewDatabase(t)
	a := db.A
	repo := equipmentcontinuityrepository.New(
		equipmentcontinuityrepository.Params{DB: db.Conn, Logger: zap.NewNop()},
	)

	equipmentID := pulid.MustNew("tr_")
	shipmentID := pulid.MustNew("shp_")
	current := equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
		TenantInfo:    a.Info,
		EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
		EquipmentID:   equipmentID,
	}
	advance := func(t *testing.T, location string, shipment pulid.ID) *equipmentcontinuity.EquipmentContinuity {
		t.Helper()

		row, err := repo.Advance(a.Ctx, equipmentcontinuityports.CreateEquipmentContinuityRequest{
			TenantInfo:        a.Info,
			EquipmentType:     equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:       equipmentID,
			CurrentLocationID: pulid.ID(location),
			SourceType:        equipmentcontinuity.SourceTypeManualLocate,
			SourceShipmentID:  shipment,
		})
		require.NoError(t, err)

		return row
	}

	t.Run("nothing is current before the first advance", func(t *testing.T) {
		got, err := repo.GetCurrent(a.Ctx, current)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	first := advance(t, "loc_first", pulid.Nil)
	second := advance(t, "loc_second", shipmentID)

	t.Run("an advance supersedes the current row", func(t *testing.T) {
		got, err := repo.GetCurrent(a.Ctx, current)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, second.ID, got.ID)
		assert.Equal(t, first.ID, got.PreviousContinuityID)

		previous, err := repo.GetByID(a.Ctx, first.ID)
		require.NoError(t, err)
		assert.False(t, previous.IsCurrent)
		require.NotNil(t, previous.SupersededAt)
		assert.Equal(t, int64(1), previous.Version)
	})

	t.Run("a manual locate is effective", func(t *testing.T) {
		got, err := repo.GetEffectiveCurrent(a.Ctx, current)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, second.ID, got.ID)
	})

	t.Run("rolling back a shipment restores the row before it", func(t *testing.T) {
		require.NoError(t, repo.RollbackCurrentByShipment(a.Ctx,
			equipmentcontinuityports.RollbackEquipmentContinuityByShipmentRequest{
				TenantInfo: a.Info,
				ShipmentID: shipmentID,
			}))

		got, err := repo.GetCurrent(a.Ctx, current)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, first.ID, got.ID)
		assert.Nil(t, got.SupersededAt)
		assert.Equal(t, int64(2), got.Version)
	})

	t.Run("another tenant sees none of it", func(t *testing.T) {
		got, err := repo.GetCurrent(db.B.Ctx, current)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
