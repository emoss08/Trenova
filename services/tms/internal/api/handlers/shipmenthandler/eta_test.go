package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentcommentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type etaReaderStub struct {
	etas    map[pulid.ID]*shipmenttrackingports.ShipmentEta
	tenants []pagination.TenantInfo
}

func (s *etaReaderStub) EtasByShipmentIDs(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	_ []pulid.ID,
) (map[pulid.ID]*shipmenttrackingports.ShipmentEta, error) {
	s.tenants = append(s.tenants, tenantInfo)
	return s.etas, nil
}

type etaResponse struct {
	ShipmentID       string `json:"shipmentId"`
	StopID           string `json:"stopId"`
	EstimatedArrival *int64 `json:"estimatedArrival"`
	SlackMinutes     *int64 `json:"slackMinutes"`
	Verdict          string `json:"verdict"`
	Stops            []struct {
		StopID  string `json:"stopId"`
		Verdict string `json:"verdict"`
	} `json:"stops"`
}

func serveEta(t *testing.T, etas *etaReaderStub, shipmentID string) (int, etaResponse) {
	t.Helper()

	handler := buildShipmentHandler(t, handlerDeps{
		service:        shipmentserviceportsmock.NewMockShipmentService(t),
		commentService: shipmentcommentportsmock.NewMockShipmentCommentService(t),
		holdService:    shipmentholdportsmock.NewMockShipmentHoldService(t),
		etas:           etas,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID + "/eta/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	var resp etaResponse
	if ginCtx.ResponseCode() == http.StatusOK {
		require.NoError(t, ginCtx.ResponseJSON(&resp))
	}
	return ginCtx.ResponseCode(), resp
}

func TestShipmentHandler_GetEta_ReturnsTheStoredProjection(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	stopID := pulid.MustNew("stp_")
	arrival := int64(1_800_000_000)
	slack := int64(-20)
	etas := &etaReaderStub{etas: map[pulid.ID]*shipmenttrackingports.ShipmentEta{
		shipmentID: {
			ShipmentID:       shipmentID,
			StopID:           stopID,
			EstimatedArrival: &arrival,
			SlackMinutes:     &slack,
			Verdict:          stopeta.VerdictLate,
			Stops: []*stopeta.StopETA{{
				StopID:  stopID,
				Verdict: stopeta.VerdictLate,
			}},
		},
	}}

	code, resp := serveEta(t, etas, shipmentID.String())

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, shipmentID.String(), resp.ShipmentID)
	assert.Equal(t, stopID.String(), resp.StopID)
	assert.Equal(t, "Late", resp.Verdict)
	require.NotNil(t, resp.SlackMinutes)
	assert.Equal(t, int64(-20), *resp.SlackMinutes)
	require.Len(t, resp.Stops, 1)
	assert.Equal(t, stopID.String(), resp.Stops[0].StopID)
	require.Len(t, etas.tenants, 1)
	assert.Equal(t, testutil.TestOrgID, etas.tenants[0].OrgID)
	assert.Equal(t, testutil.TestBuID, etas.tenants[0].BuID)
}

func TestShipmentHandler_GetEta_UnprojectedShipmentIsUnknown(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	code, resp := serveEta(t, &etaReaderStub{}, shipmentID.String())

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "Unknown", resp.Verdict)
	assert.Nil(t, resp.EstimatedArrival)
	assert.NotNil(t, resp.Stops)
	assert.Empty(t, resp.Stops)
}

func TestShipmentHandler_GetEta_InvalidShipmentID(t *testing.T) {
	t.Parallel()

	code, _ := serveEta(t, &etaReaderStub{}, "not-an-id")

	assert.Equal(t, http.StatusBadRequest, code)
}
