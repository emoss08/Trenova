package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentcommentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackingPreviewStub struct {
	customertrackingports.CustomerTrackingService
	view      *customertrackingports.PublicTrackingView
	err       error
	tenants   []pagination.TenantInfo
	shipments []pulid.ID
}

func (s *trackingPreviewStub) PreviewForShipment(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.PublicTrackingView, error) {
	s.tenants = append(s.tenants, tenantInfo)
	s.shipments = append(s.shipments, shipmentID)
	return s.view, s.err
}

type previewResponse struct {
	CompanyName       string   `json:"companyName"`
	ShipmentProNumber string   `json:"shipmentProNumber"`
	Fields            []string `json:"fields"`
	Status            string   `json:"status"`
	Stops             []struct {
		Sequence int    `json:"sequence"`
		City     string `json:"city"`
	} `json:"stops"`
}

func servePreview(
	t *testing.T,
	tracking *trackingPreviewStub,
	shipmentID string,
) (int, previewResponse, http.Header) {
	t.Helper()

	handler := buildShipmentHandler(t, handlerDeps{
		service:        newShipmentServiceMocks(t),
		commentService: shipmentcommentportsmock.NewMockShipmentCommentService(t),
		holdService:    shipmentholdportsmock.NewMockShipmentHoldService(t),
		tracking:       tracking,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID + "/customer-tracking/preview/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	var resp previewResponse
	if ginCtx.ResponseCode() == http.StatusOK {
		require.NoError(t, ginCtx.ResponseJSON(&resp))
	}
	return ginCtx.ResponseCode(), resp, ginCtx.Recorder.Header()
}

func TestShipmentHandler_PreviewCustomerTracking_ReturnsThePublicView(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	tracking := &trackingPreviewStub{view: &customertrackingports.PublicTrackingView{
		CompanyName:       "Trenova Logistics",
		ShipmentProNumber: "S2610-0398",
		Fields:            []customertracking.Field{customertracking.FieldStatus},
		Status:            customertrackingports.PublicStatusInTransit,
		Stops: []customertrackingports.PublicStop{
			{Sequence: 1, City: "Omaha"},
			{Sequence: 2, City: "Minneapolis"},
		},
	}}

	code, resp, headers := servePreview(t, tracking, shipmentID.String())

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "no-store", headers.Get("Cache-Control"))
	assert.Equal(t, "Trenova Logistics", resp.CompanyName)
	assert.Equal(t, "S2610-0398", resp.ShipmentProNumber)
	assert.Equal(t, []string{"Status"}, resp.Fields)
	assert.Equal(t, "InTransit", resp.Status)
	require.Len(t, resp.Stops, 2)
	assert.Equal(t, "Minneapolis", resp.Stops[1].City)
	require.Len(t, tracking.tenants, 1)
	assert.Equal(t, testutil.TestOrgID, tracking.tenants[0].OrgID)
	assert.Equal(t, testutil.TestBuID, tracking.tenants[0].BuID)
	assert.Equal(t, []pulid.ID{shipmentID}, tracking.shipments)
}

func TestShipmentHandler_PreviewCustomerTracking_UnknownShipmentIsNotFound(t *testing.T) {
	t.Parallel()

	tracking := &trackingPreviewStub{err: errortypes.NewNotFoundError("Shipment not found")}
	code, _, _ := servePreview(t, tracking, pulid.MustNew("shp_").String())

	assert.Equal(t, http.StatusNotFound, code)
}

func TestShipmentHandler_PreviewCustomerTracking_InvalidShipmentID(t *testing.T) {
	t.Parallel()

	tracking := &trackingPreviewStub{}
	code, _, _ := servePreview(t, tracking, "not-an-id")

	assert.Equal(t, http.StatusBadRequest, code)
	assert.Empty(t, tracking.shipments)
}
