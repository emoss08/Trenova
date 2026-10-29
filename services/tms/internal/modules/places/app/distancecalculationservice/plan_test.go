package distancecalculationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentmoveportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlanMoveJurisdictionMiles_NamesTheMoveWithoutRoutingIt(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	routable := testMove(tenant.OrgID, tenant.BuID, true)
	single := testMove(tenant.OrgID, tenant.BuID, true)
	single.Stops = single.Stops[:1]

	moves := shipmentmoveportsmock.NewMockShipmentMoveRepository(t)
	moves.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentmoveports.GetMoveByIDRequest) bool {
			return req.MoveID == routable.ID
		})).
		Return(routable, nil).
		Once()
	moves.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentmoveports.GetMoveByIDRequest) bool {
			return req.MoveID == single.ID
		})).
		Return(single, nil).
		Once()

	svc := &Service{l: zap.NewNop(), shipmentMoveRepo: moves}

	plan, err := svc.PlanMoveJurisdictionMiles(
		t.Context(),
		&shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo:     tenant,
			ShipmentMoveID: routable.ID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, routable.ID, plan.Move.ID)

	_, err = svc.PlanMoveJurisdictionMiles(
		t.Context(),
		&shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo:     tenant,
			ShipmentMoveID: single.ID,
		},
	)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	_, err = svc.PlanMoveJurisdictionMiles(
		t.Context(),
		&shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo: tenant,
		},
	)
	require.ErrorAs(t, err, &business)
}
