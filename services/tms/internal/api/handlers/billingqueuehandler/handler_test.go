package billingqueuehandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/billingqueuehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type visibleShipment struct {
	FormulaTemplate *struct {
		Source string `json:"source"`
	} `json:"formulaTemplate"`
	RatingDetail struct {
		FormulaTemplateName string         `json:"formulaTemplateName"`
		Trace               map[string]any `json:"trace"`
	} `json:"ratingDetail"`
}

type visibleItem struct {
	ID       pulid.ID        `json:"id"`
	Shipment visibleShipment `json:"shipment"`
}

func ratedItem() *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		ID: pulid.MustNew("bqi_"),
		Shipment: &shipment.Shipment{
			ID:              pulid.MustNew("shp_"),
			FormulaTemplate: &formulatemplate.FormulaTemplate{Source: "tariff \"Per mile\"\n"},
			RatingDetail: &shipment.RatingDetail{
				FormulaTemplateName: "Per mile",
				Trace: &shipment.RatingTrace{
					Statements: rules.RawJSON(`["miles * rate"]`),
				},
			},
		},
	}
}

func buildHandler(
	t *testing.T,
	service billingqueueports.BillingQueueService,
	permissions permissionports.PermissionEngine,
) *billingqueuehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &appconfig.Config{Debug: true},
	})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: permissions,
		ErrorHandler:     errorHandler,
	})

	return billingqueuehandler.New(billingqueuehandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
		Logger:               logger,
	})
}

func withoutFormulaRead() permissionports.PermissionEngine {
	return &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceFormulaTemplate.String()},
	}
}

func TestBillingQueueHandler_Get_FormulaContentFollowsPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		permissions permissionports.PermissionEngine
		wantVisible bool
	}{
		{
			name:        "formula reader sees the shipment's template and expression",
			permissions: &mocks.AllowAllPermissionEngine{},
			wantVisible: true,
		},
		{
			name:        "biller without formula read gets neither",
			permissions: withoutFormulaRead(),
		},
		{
			name: "a failing permission engine hides the content and keeps the answer",
			permissions: &mocks.FailingPermissionEngine{
				Failing: []string{permission.ResourceFormulaTemplate.String()},
				Err:     errors.New("engine down"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := ratedItem()
			service := billingqueueportsmock.NewMockBillingQueueService(t)
			service.EXPECT().
				GetByID(mock.Anything, mock.MatchedBy(func(req *billingqueueports.GetBillingQueueItemByIDRequest) bool {
					return req.ItemID == item.ID && req.ExpandShipmentDetails
				})).
				Return(item, nil).
				Once()

			handler := buildHandler(t, service, tt.permissions)

			ginCtx := testutil.NewGinTestContext().
				WithMethod(http.MethodGet).
				WithPath("/api/v1/billing-queue/" + item.ID.String() + "/").
				WithQuery(map[string]string{"expandShipmentDetails": "true"}).
				WithDefaultAuthContext()

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))

			var body visibleItem
			require.NoError(t, ginCtx.ResponseJSON(&body))
			assert.Equal(t, item.ID, body.ID)
			assert.Equal(t, "Per mile", body.Shipment.RatingDetail.FormulaTemplateName)
			if tt.wantVisible {
				require.NotNil(t, body.Shipment.FormulaTemplate)
				assert.Equal(t, "tariff \"Per mile\"\n", body.Shipment.FormulaTemplate.Source)
				assert.NotEmpty(t, body.Shipment.RatingDetail.Trace)
				return
			}
			assert.Nil(t, body.Shipment.FormulaTemplate)
			assert.Empty(t, body.Shipment.RatingDetail.Trace)
			assert.NotEmpty(
				t,
				item.Shipment.RatingDetail.Trace,
				"the stored shipment keeps its trace",
			)
		})
	}
}

func TestBillingQueueHandler_List_HidesFormulaContentWithoutFormulaRead(t *testing.T) {
	t.Parallel()

	service := billingqueueportsmock.NewMockBillingQueueService(t)
	service.EXPECT().
		List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*billingqueue.BillingQueueItem]{
			Items:      []*billingqueue.BillingQueueItem{ratedItem(), {ID: pulid.MustNew("bqi_")}},
			Total:      2,
			TotalExact: true,
		}, nil).
		Once()

	handler := buildHandler(t, service, withoutFormulaRead())

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/billing-queue/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))

	var body struct {
		Items []visibleItem `json:"results"`
	}
	require.NoError(t, ginCtx.ResponseJSON(&body))
	require.Len(t, body.Items, 2)
	assert.Nil(t, body.Items[0].Shipment.FormulaTemplate)
	assert.Empty(t, body.Items[0].Shipment.RatingDetail.Trace)
	assert.Equal(t, "Per mile", body.Items[0].Shipment.RatingDetail.FormulaTemplateName)
}
