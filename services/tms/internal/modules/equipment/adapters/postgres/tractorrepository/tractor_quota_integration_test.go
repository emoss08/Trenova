//go:build integration

package tractorrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateRefusesATractorPastTheQuota(t *testing.T) {
	db := equipmenttest.NewDatabase(t)
	a := db.A
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := tractorrepository.New(tractorrepository.Params{
		DB:        db.Conn,
		Logger:    zap.NewNop(),
		Quota:     guard,
		Locations: crossref.LocationColumns(),
		Moves:     crossref.MoveColumns(),
	})
	entity := &tractor.Tractor{
		OrganizationID: a.Info.OrgID,
		BusinessUnitID: a.Info.BuID,
		Code:           "QUOTA-1",
	}

	guard.EXPECT().
		Enforce(mock.MatchedBy(func(ctx context.Context) bool {
			return keel.InTx(ctx, db.Conn.Keel(ctx))
		}), &platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{OrgID: a.Info.OrgID, BuID: a.Info.BuID},
			Meter:      platformcatalog.MeterTractorsTotal,
			Quantity:   1,
		}).
		Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterTractorsTotal), 3, 3, "free_demo")).
		Once()

	created, err := repo.Create(a.Ctx, entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.Zero(t, equipmenttest.Value[int64](t, db,
		"SELECT count(*) FROM tractors WHERE organization_id = $1 AND business_unit_id = $2 AND code = $3",
		a.Info.OrgID.String(), a.Info.BuID.String(), entity.Code))
}
