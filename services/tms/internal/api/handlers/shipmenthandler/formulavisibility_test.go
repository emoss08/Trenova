package shipmenthandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type formulaVisibilityShipment struct {
	FormulaTemplate *struct {
		Source string `json:"source"`
	} `json:"formulaTemplate"`
	RatingDetail struct {
		FormulaTemplateName string         `json:"formulaTemplateName"`
		Trace               map[string]any `json:"trace"`
		Bill                *struct {
			Total string `json:"total"`
		} `json:"bill"`
	} `json:"ratingDetail"`
}

func ratedShipment(id pulid.ID) *shipment.Shipment {
	return &shipment.Shipment{
		ID:              id,
		FormulaTemplate: &formulatemplate.FormulaTemplate{Source: "tariff \"Per pound\"\n"},
		RatingDetail: &shipment.RatingDetail{
			FormulaTemplateName: "Per pound",
			Trace:               rules.RawJSON(`{"statements":["weight * 2"]}`),
			Bill: &tarifftypes.Bill{
				Currency:     "USD",
				FreightTotal: decimal.NewFromInt(42),
				Total:        decimal.NewFromInt(42),
			},
		},
	}
}

func withoutFormulaRead() permissionports.PermissionEngine {
	return &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceFormulaTemplate.String()},
	}
}

func TestShipmentHandler_Get_FormulaContentFollowsPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		permissions permissionports.PermissionEngine
		wantVisible bool
	}{
		{name: "formula reader sees the template and expression", wantVisible: true},
		{
			name:        "shipment reader without formula read gets neither",
			permissions: withoutFormulaRead(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			shipmentID := pulid.MustNew("shp_")
			service := shipmentportsmock.NewMockShipmentService(t)
			service.EXPECT().
				Get(mock.Anything, mock.Anything).
				Return(ratedShipment(shipmentID), nil).
				Once()

			handler := buildShipmentHandler(t, handlerDeps{
				service:     service,
				permissions: tt.permissions,
			})

			ginCtx := testutil.NewGinTestContext().
				WithMethod(http.MethodGet).
				WithPath("/api/v1/shipments/" + shipmentID.String()).
				WithQuery(map[string]string{"expandShipmentDetails": "true"}).
				WithDefaultAuthContext()

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))

			var body formulaVisibilityShipment
			require.NoError(t, ginCtx.ResponseJSON(&body))
			assert.Equal(t, "Per pound", body.RatingDetail.FormulaTemplateName)
			require.NotNil(t, body.RatingDetail.Bill, "the bill is not formula content")
			assert.Equal(t, "42", body.RatingDetail.Bill.Total)
			if tt.wantVisible {
				require.NotNil(t, body.FormulaTemplate)
				assert.Equal(t, "tariff \"Per pound\"\n", body.FormulaTemplate.Source)
				assert.NotEmpty(t, body.RatingDetail.Trace)
				return
			}
			assert.Nil(t, body.FormulaTemplate)
			assert.Empty(t, body.RatingDetail.Trace)
		})
	}
}

func TestShipmentHandler_List_HidesFormulaContentWithoutFormulaRead(t *testing.T) {
	t.Parallel()

	service := shipmentportsmock.NewMockShipmentService(t)
	service.EXPECT().
		List(mock.Anything, mock.Anything).
		Return(&pagination.CursorListResult[*shipment.Shipment]{
			Items: []*shipment.Shipment{
				ratedShipment(pulid.MustNew("shp_")),
				{ID: pulid.MustNew("shp_")},
			},
		}, nil).
		Once()

	handler := buildShipmentHandler(t, handlerDeps{
		service:     service,
		permissions: withoutFormulaRead(),
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/").
		WithQuery(map[string]string{"expandShipmentDetails": "true"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))

	var body struct {
		Items []formulaVisibilityShipment `json:"results"`
	}
	require.NoError(t, ginCtx.ResponseJSON(&body))
	require.Len(t, body.Items, 2)
	assert.Nil(t, body.Items[0].FormulaTemplate)
	assert.Empty(t, body.Items[0].RatingDetail.Trace)
	assert.Equal(t, "Per pound", body.Items[0].RatingDetail.FormulaTemplateName)
}

func TestShipmentHandler_TransferToBilling_HidesFormulaContentWithoutFormulaRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		permissions permissionports.PermissionEngine
	}{
		{name: "formula read denied", permissions: withoutFormulaRead()},
		{
			name: "permission engine failing after the write committed",
			permissions: &mocks.FailingPermissionEngine{
				Failing: []string{permission.ResourceFormulaTemplate.String()},
				Err:     errors.New("engine down"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			shipmentID := pulid.MustNew("shp_")
			service := shipmentportsmock.NewMockShipmentService(t)
			service.EXPECT().
				TransferToBilling(mock.Anything, mock.Anything, mock.Anything).
				Return(&billingqueue.BillingQueueItem{
					ID:       pulid.MustNew("bqi_"),
					Shipment: ratedShipment(shipmentID),
				}, nil).
				Once()

			handler := buildShipmentHandler(t, handlerDeps{
				service:     service,
				permissions: tt.permissions,
			})

			ginCtx := testutil.NewGinTestContext().
				WithMethod(http.MethodPost).
				WithPath("/api/v1/shipments/" + shipmentID.String() + "/transfer-to-billing/").
				WithJSONBody(map[string]any{}).
				WithDefaultAuthContext()

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))

			var body struct {
				Shipment formulaVisibilityShipment `json:"shipment"`
			}
			require.NoError(t, ginCtx.ResponseJSON(&body))
			assert.Nil(t, body.Shipment.FormulaTemplate)
			assert.Empty(t, body.Shipment.RatingDetail.Trace)
			assert.Equal(t, "Per pound", body.Shipment.RatingDetail.FormulaTemplateName)
		})
	}
}
