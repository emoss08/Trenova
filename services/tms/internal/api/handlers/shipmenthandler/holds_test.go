package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentcommentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestShipmentHandler_ListHolds_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	shipmentID := pulid.MustNew("shp_")

	holdService.EXPECT().
		ListByShipmentID(mock.Anything, mock.MatchedBy(func(req *shipmentholdports.ListShipmentHoldsRequest) bool {
			return req.ShipmentID == shipmentID &&
				req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
				req.Filter.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(&pagination.ListResult[*shipmenthold.ShipmentHold]{
			Items: []*shipmenthold.ShipmentHold{
				{ID: pulid.MustNew("shh_"), ShipmentID: shipmentID},
			},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()

	handler := setupShipmentHandlerWithSubresources(t, service, commentService, holdService)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/holds/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_CreateHold_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	shipmentID := pulid.MustNew("shp_")
	holdReasonID := pulid.MustNew("hr_")

	holdService.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(req *shipmentholdports.CreateShipmentHoldRequest) bool {
			return req.ShipmentID == shipmentID &&
				req.HoldReasonID == holdReasonID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		}), mock.AnythingOfType("*permissionports.RequestActor")).
		RunAndReturn(func(_ context.Context, req *shipmentholdports.CreateShipmentHoldRequest, _ *permissionports.RequestActor) (*shipmenthold.ShipmentHold, error) {
			return &shipmenthold.ShipmentHold{
				ID:             pulid.MustNew("shh_"),
				ShipmentID:     req.ShipmentID,
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				HoldReasonID:   &req.HoldReasonID,
				Type:           holdreason.HoldTypeOperational,
				Severity:       holdreason.HoldSeverityBlocking,
				ReasonCode:     "APPT",
				Source:         shipmenthold.HoldSourceUser,
				BlocksDispatch: true,
				StartedAt:      100,
			}, nil
		}).
		Once()

	handler := setupShipmentHandlerWithSubresources(t, service, commentService, holdService)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/holds/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"holdReasonId": holdReasonID.String(),
			"notes":        "dock issue",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
}

func TestShipmentHandler_ReleaseHold_InvalidHoldID(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	shipmentID := pulid.MustNew("shp_")

	handler := setupShipmentHandlerWithSubresources(t, service, commentService, holdService)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/holds/not-a-hold/release/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
