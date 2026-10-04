package tractorrepository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
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

func newTractorForQuota() *tractor.Tractor {
	return &tractor.Tractor{
		ID:             pulid.MustNew("tr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "TRK1",
	}
}

func tractorQuota(entity *tractor.Tractor) *services.QuotaRequest {
	return &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		Meter:      platformcatalog.MeterTractorsTotal,
		Quantity:   1,
	}
}

func TestCreateRefusesATractorPastTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	entity := newTractorForQuota()

	guard.EXPECT().Enforce(mock.Anything, tractorQuota(entity)).
		Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterTractorsTotal), 3, 3, "free_demo"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	created, err := repo.Create(t.Context(), entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestCreateInsertsATractorUnderTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	entity := newTractorForQuota()

	guard.EXPECT().Enforce(mock.Anything, tractorQuota(entity)).Return(nil)
	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tractors"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(entity.ID.String()))
	sqlMock.ExpectCommit()

	created, err := repo.Create(t.Context(), entity)

	require.NoError(t, err)
	require.NotNil(t, created)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
