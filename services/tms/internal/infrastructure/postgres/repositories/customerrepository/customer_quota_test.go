package customerrepository

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateRefusesACustomerPastTheQuota(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newTestRepository(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo.quota = guard
	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "CUST9",
	}

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		Meter:      platformcatalog.MeterCustomersTotal,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterCustomersTotal), 8, 8, "free_demo"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	created, err := repo.Create(t.Context(), entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
