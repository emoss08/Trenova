package recurringshipmentrepository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func quotaSeries() *recurringshipment.RecurringShipment {
	return &recurringshipment.RecurringShipment{
		ID:             pulid.MustNew("rsh_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func seriesTenant(series *recurringshipment.RecurringShipment) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: series.OrganizationID, BuID: series.BusinessUnitID}
}

func TestInsertSeriesRefusesASeriesPastTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	series := quotaSeries()

	guard.EXPECT().Enforce(mock.Anything, &services.QuotaRequest{
		TenantInfo: seriesTenant(series),
		Meter:      platformcatalog.MeterRecurringShipmentSeries,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterRecurringShipmentSeries), 1, 1, "free_demo",
	))
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	err := repo.insertSeries(t.Context(), series)

	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestInsertSeriesWritesUnderTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	series := quotaSeries()

	guard.EXPECT().Enforce(mock.Anything, mock.Anything).Return(nil)
	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "recurring_shipments"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(series.ID.String()))
	sqlMock.ExpectCommit()

	require.NoError(t, repo.insertSeries(t.Context(), series))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestEnforceGeneratedShipmentCountsOneShipment(t *testing.T) {
	t.Parallel()

	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{l: zap.NewNop(), quota: guard}
	series := quotaSeries()
	refusal := errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterShipmentsTotal), 12, 12, "free_demo",
	)

	guard.EXPECT().Enforce(mock.Anything, &services.QuotaRequest{
		TenantInfo: seriesTenant(series),
		Meter:      platformcatalog.MeterShipmentsTotal,
		Quantity:   1,
	}).Return(refusal)

	require.ErrorIs(t, repo.enforceGeneratedShipment(t.Context(), series), refusal)
}

func TestEnforceGeneratedShipmentIsANoOpWithoutAGuard(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}
	require.NoError(t, repo.enforceGeneratedShipment(t.Context(), quotaSeries()))
}
