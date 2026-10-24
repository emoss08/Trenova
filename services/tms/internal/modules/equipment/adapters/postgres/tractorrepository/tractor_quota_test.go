package tractorrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
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

func tractorQuota(entity *tractor.Tractor) *platformplanports.QuotaRequest {
	return &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterTractorsTotal,
		Quantity: 1,
	}
}

func TestCreateRefusesATractorPastTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
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
