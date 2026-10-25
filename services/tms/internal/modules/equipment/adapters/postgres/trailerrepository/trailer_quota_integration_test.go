//go:build integration

package trailerrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/trailerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
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

func TestCreateRefusesATrailerPastTheQuota(t *testing.T) {
	db := equipmenttest.NewDatabase(t)
	a := db.A
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := trailerrepository.New(trailerrepository.Params{
		DB:        db.Conn,
		Logger:    zap.NewNop(),
		Quota:     guard,
		Locations: crossref.LocationColumns(),
	})
	entity := &trailer.Trailer{
		OrganizationID: a.Info.OrgID,
		BusinessUnitID: a.Info.BuID,
		Code:           "QUOTA-1",
	}

	guard.EXPECT().
		Enforce(mock.MatchedBy(func(ctx context.Context) bool {
			return keel.InTx(ctx, db.Conn.Keel(ctx))
		}), &platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{OrgID: a.Info.OrgID, BuID: a.Info.BuID},
			Meter:      platformcatalog.MeterTrailersTotal,
			Quantity:   1,
		}).
		Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterTrailersTotal), 3, 3, "free_demo")).
		Once()

	created, err := repo.Create(a.Ctx, entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.Zero(t, equipmenttest.Value[int64](t, db,
		"SELECT count(*) FROM trailers WHERE organization_id = $1 AND business_unit_id = $2 AND code = $3",
		a.Info.OrgID.String(), a.Info.BuID.String(), entity.Code))
}
