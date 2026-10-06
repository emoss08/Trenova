package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

func refusedDecision(quantity int64) *services.QuotaDecision {
	return &services.QuotaDecision{
		Meter:     platformcatalog.MeterShipmentsTotal,
		Plan:      platformplan.PlanKeyFreeDemo,
		Allowed:   false,
		Limit:     12,
		Used:      11,
		Requested: quantity,
	}
}

func TestCreateRefusesBeforeMintingAProNumber(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	entity := &shipment.Shipment{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	guard.EXPECT().Check(mock.Anything, &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		Meter:      platformcatalog.MeterShipmentsTotal,
		Quantity:   1,
	}).Return(refusedDecision(1), nil)

	created, err := repo.Create(t.Context(), entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.Empty(t, entity.ProNumber)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestBulkDuplicateRefusesEveryCopyThePlanCannotHold(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	guard.EXPECT().Check(mock.Anything, &services.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterShipmentsTotal,
		Quantity:   3,
	}).Return(refusedDecision(3), nil)

	duplicated, err := repo.BulkDuplicate(t.Context(), &repositories.BulkDuplicateShipmentRequest{
		TenantInfo: tenant,
		ShipmentID: pulid.MustNew("shp_"),
		Count:      3,
	})

	require.Nil(t, duplicated)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
