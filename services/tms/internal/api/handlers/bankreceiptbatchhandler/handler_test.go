package bankreceiptbatchhandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/bankreceiptbatchhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
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
	service *mocks.MockBankReceiptBatchService,
) *bankreceiptbatchhandler.Handler {
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

	return bankreceiptbatchhandler.New(
		bankreceiptbatchhandler.Params{
			Service:              service,
			ErrorHandler:         errorHandler,
			PermissionMiddleware: pm,
		},
	)
}

func TestHandlerListBatches(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockBankReceiptBatchService(t)
	handler := setupHandler(t, service)
	batch := &bankreceiptbatch.BankReceiptBatch{
		ID:             pulid.MustNew("brib_"),
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Source:         "csv",
		Status:         bankreceiptbatch.StatusCompleted,
	}

	service.EXPECT().
		List(mock.Anything, pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}).
		Return([]*bankreceiptbatch.BankReceiptBatch{batch}, nil).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/bank-receipt-batches/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp []*bankreceiptbatch.BankReceiptBatch
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, batch.ID, resp[0].ID)
}

func TestHandlerGetBatchDetail(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockBankReceiptBatchService(t)
	handler := setupHandler(t, service)
	batchID := pulid.MustNew("brib_")
	batch := &bankreceiptbatch.BankReceiptBatch{
		ID:             batchID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Source:         "csv",
		Status:         bankreceiptbatch.StatusCompleted,
	}
	receipt := &bankreceipt.BankReceipt{
		ID:            pulid.MustNew("brcpt_"),
		ImportBatchID: batchID,
		AmountMinor:   10000,
	}

	service.EXPECT().
		Get(mock.Anything, &serviceports.GetBankReceiptBatchRequest{BatchID: batchID, TenantInfo: pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID, UserID: sharedtestutil.TestUserID}}).
		Return(&serviceports.BankReceiptBatchResult{Batch: batch, Receipts: []*bankreceipt.BankReceipt{receipt}}, nil).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/bank-receipt-batches/" + batchID.String() + "/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp serviceports.BankReceiptBatchResult
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.Batch)
	require.Len(t, resp.Receipts, 1)
	assert.Equal(t, batchID, resp.Batch.ID)
	assert.Equal(t, batchID, resp.Receipts[0].ImportBatchID)
}

func TestHandlerImportBatch(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockBankReceiptBatchService(t)
	handler := setupHandler(t, service)
	batchID := pulid.MustNew("brib_")
	batch := &bankreceiptbatch.BankReceiptBatch{
		ID:             batchID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Source:         "csv",
		Status:         bankreceiptbatch.StatusCompleted,
	}
	receipt := &bankreceipt.BankReceipt{
		ID:            pulid.MustNew("brcpt_"),
		ImportBatchID: batchID,
		AmountMinor:   10000,
	}

	service.EXPECT().
		Import(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *serviceports.ImportBankReceiptBatchRequest, actor *serviceports.RequestActor) (*serviceports.BankReceiptBatchResult, error) {
			require.Equal(t, "csv", req.Source)
			require.Equal(t, sharedtestutil.TestOrgID, req.TenantInfo.OrgID)
			require.Equal(t, sharedtestutil.TestBuID, req.TenantInfo.BuID)
			require.Equal(t, sharedtestutil.TestUserID, actor.UserID)
			return &serviceports.BankReceiptBatchResult{
				Batch:    batch,
				Receipts: []*bankreceipt.BankReceipt{receipt},
			}, nil
		}).
		Once()

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accounting/bank-receipt-batches/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"source": "csv", "reference": "BATCH-1", "receipts": []map[string]any{{"receiptDate": 1, "amountMinor": 10000, "referenceNumber": "A"}}})
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
	var resp serviceports.BankReceiptBatchResult
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.Batch)
	assert.Equal(t, batchID, resp.Batch.ID)
}

func TestHandlerGetBatchInvalidID(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockBankReceiptBatchService(t)
	handler := setupHandler(t, service)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/bank-receipt-batches/not-a-pulid/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestHandlerImportBatchBadJSON(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockBankReceiptBatchService(t)
	handler := setupHandler(t, service)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/accounting/bank-receipt-batches/").
		WithDefaultAuthContext().
		WithBody("{invalid")
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
