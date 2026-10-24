package recurringshipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
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

func TestEnforceGeneratedShipmentCountsOneShipment(t *testing.T) {
	t.Parallel()

	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := &repository{l: zap.NewNop(), quota: guard}
	series := quotaSeries()
	refusal := errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterShipmentsTotal), 12, 12, "free_demo",
	)

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
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
