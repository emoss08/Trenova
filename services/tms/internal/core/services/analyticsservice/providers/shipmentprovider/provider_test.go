package shipmentprovider

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestProvider(t *testing.T) (*Provider, sqlmock.Sqlmock, *mocks.MockShipmentControlRepository) {
	t.Helper()

	db, mockDB, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mockDB.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	controlRepo := mocks.NewMockShipmentControlRepository(t)

	return &Provider{
		l:           zap.NewNop(),
		db:          postgres.NewTestConnection(bunDB),
		controlRepo: controlRepo,
	}, mockDB, controlRepo
}

func TestGetDetentionAlerts_ReturnsZeroWhenTrackingDisabled(t *testing.T) {
	t.Parallel()

	provider, mockDB, controlRepo := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().
		Get(t.Context(), repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{TrackDetentionTime: false}, nil).
		Once()

	card, err := provider.getDetentionAlerts(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, 0, card.Count)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetDetentionAlerts_UsesShipmentControlThreshold(t *testing.T) {
	t.Parallel()

	provider, mockDB, controlRepo := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().
		Get(t.Context(), repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			TrackDetentionTime: true,
			DetentionThreshold: ptrInt16(45),
		}, nil).
		Once()

	mockDB.ExpectQuery(`SELECT count\(\*\) FROM stops stp.*\(stp\.actual_departure - stp\.actual_arrival\) > .*2700.*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	card, err := provider.getDetentionAlerts(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, 3, card.Count)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

//go:fix inline
func ptrInt16(v int16) *int16 {
	return new(v)
}
