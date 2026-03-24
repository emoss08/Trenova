package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetPreviousRates_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	originID := pulid.MustNew("loc_")
	destinationID := pulid.MustNew("loc_")
	shipmentTypeID := pulid.MustNew("sht_")
	serviceTypeID := pulid.MustNew("svc_")

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetPreviousRates(mock.Anything, mock.MatchedBy(func(req *repositories.GetPreviousRatesRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.OriginLocationID == originID &&
				req.DestinationLocationID == destinationID &&
				req.ShipmentTypeID == shipmentTypeID &&
				req.ServiceTypeID == serviceTypeID
		})).
		Return(&pagination.ListResult[*repositories.PreviousRateSummary]{
			Items: []*repositories.PreviousRateSummary{{ShipmentID: pulid.MustNew("shp_")}},
			Total: 1,
		}, nil).
		Once()

	svc := &service{
		l:    zap.NewNop(),
		repo: repo,
	}

	result, err := svc.GetPreviousRates(t.Context(), &repositories.GetPreviousRatesRequest{
		TenantInfo:            pagination.TenantInfo{OrgID: orgID, BuID: buID},
		OriginLocationID:      originID,
		DestinationLocationID: destinationID,
		ShipmentTypeID:        shipmentTypeID,
		ServiceTypeID:         serviceTypeID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
}

func TestServiceGetPreviousRates_ValidatesRequest(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:    zap.NewNop(),
		repo: mocks.NewMockShipmentRepository(t),
	}

	result, err := svc.GetPreviousRates(t.Context(), &repositories.GetPreviousRatesRequest{})

	require.Nil(t, result)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}
