package customerpaymenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/customerpaymenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupHandler(
	t *testing.T,
	service *mocks.MockCustomerPaymentService,
) *customerpaymenthandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	pm := middleware.NewPermissionMiddleware(
		middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
			ErrorHandler:     errorHandler,
		},
	)

	return customerpaymenthandler.New(
		customerpaymenthandler.Params{
			Service:              service,
			ErrorHandler:         errorHandler,
			PermissionMiddleware: pm,
		},
	)
}

func TestHandlerListCustomerPayments(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)
	payment := &customerpayment.Payment{
		ID:             pulid.MustNew("cpay_"),
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Status:         customerpayment.StatusPosted,
		AmountMinor:    10000,
	}

	service.EXPECT().
		List(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.ListCustomerPaymentsRequest) (*pagination.ListResult[*customerpayment.Payment], error) {
			assert.Equal(t, sharedtestutil.TestOrgID, req.Filter.TenantInfo.OrgID)
			assert.Equal(t, sharedtestutil.TestBuID, req.Filter.TenantInfo.BuID)
			assert.Equal(t, sharedtestutil.TestUserID, req.Filter.TenantInfo.UserID)
			return &pagination.ListResult[*customerpayment.Payment]{
				Items: []*customerpayment.Payment{payment},
				Total: 1,
			}, nil
		}).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/customer-payments/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp pagination.Response[[]*customerpayment.Payment]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, payment.ID, resp.Results[0].ID)
}

func TestHandlerGetCustomerPayment(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)
	paymentID := pulid.MustNew("cpay_")
	payment := &customerpayment.Payment{
		ID:             paymentID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Status:         customerpayment.StatusPosted,
		AmountMinor:    10000,
	}

	service.EXPECT().
		Get(mock.Anything, &serviceports.GetCustomerPaymentRequest{PaymentID: paymentID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}}).
		Return(payment, nil).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/customer-payments/" + paymentID.String() + "/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp customerpayment.Payment
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, paymentID, resp.ID)
}

func TestHandlerPostCustomerPayment(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)
	payment := &customerpayment.Payment{
		ID:             pulid.MustNew("cpay_"),
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Status:         customerpayment.StatusPosted,
		AmountMinor:    10000,
	}

	service.EXPECT().
		PostAndApply(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *serviceports.PostCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
			assert.Equal(t, sharedtestutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, sharedtestutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, sharedtestutil.TestUserID, actor.UserID)
			return payment, nil
		}).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accounting/customer-payments/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"customerId": pulid.MustNew("cus_").String(), "paymentDate": 1, "accountingDate": 1, "amountMinor": 10000, "paymentMethod": customerpayment.MethodACH, "referenceNumber": "PAY-1", "currencyCode": "USD", "applications": []map[string]any{}})
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
}

func TestHandlerApplyUnapplied(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)
	paymentID := pulid.MustNew("cpay_")
	payment := &customerpayment.Payment{
		ID:             paymentID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Status:         customerpayment.StatusPosted,
		AmountMinor:    10000,
	}

	service.EXPECT().
		ApplyUnapplied(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *serviceports.ApplyCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
			assert.Equal(t, paymentID, req.PaymentID)
			assert.Equal(t, sharedtestutil.TestUserID, actor.UserID)
			return payment, nil
		}).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accounting/customer-payments/" + paymentID.String() + "/apply/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"accountingDate": 1, "applications": []map[string]any{{"invoiceId": pulid.MustNew("inv_").String(), "appliedAmountMinor": 10000}}})
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestHandlerReverseCustomerPayment(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)
	paymentID := pulid.MustNew("cpay_")
	payment := &customerpayment.Payment{
		ID:             paymentID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Status:         customerpayment.StatusReversed,
		AmountMinor:    10000,
	}

	service.EXPECT().
		Reverse(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *serviceports.ReverseCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
			assert.Equal(t, paymentID, req.PaymentID)
			assert.Equal(t, sharedtestutil.TestUserID, actor.UserID)
			return payment, nil
		}).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accounting/customer-payments/" + paymentID.String() + "/reverse/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"accountingDate": 1, "reason": "chargeback"})
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestHandlerGetCustomerPaymentInvalidID(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockCustomerPaymentService(t)
	handler := setupHandler(t, service)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/customer-payments/not-a-pulid/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
