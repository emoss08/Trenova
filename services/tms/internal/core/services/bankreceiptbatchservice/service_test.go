package bankreceiptbatchservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/core/ports"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestImportBatchAggregatesReceiptOutcomes(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockBankReceiptBatchRepository(t)
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	db := fakeBatchDB{}
	auditSvc := mocks.NewMockAuditService(t)
	bankSvc := mocks.NewMockBankReceiptService(t)
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *bankreceiptbatch.BankReceiptBatch,
		) (*bankreceiptbatch.BankReceiptBatch, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brib_")
			}
			return &copy, nil
		}).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *bankreceiptbatch.BankReceiptBatch,
		) (*bankreceiptbatch.BankReceiptBatch, error) {
			copy := *entity
			return &copy, nil
		}).
		Once()
	svc := New(
		Params{
			Logger:             zap.NewNop(),
			DB:                 db,
			Repo:               repo,
			ReceiptRepo:        receiptRepo,
			BankReceiptService: bankSvc,
			AuditService:       auditSvc,
		},
	)
	actor := testutil.NewSessionActor(userID, orgID, buID)

	returnedReceipts := []*bankreceipt.BankReceipt{
		{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusMatched, AmountMinor: 10000},
		{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusException, AmountMinor: 5000},
	}
	bankSvc.EXPECT().
		Import(mock.Anything, mock.Anything, actor).
		RunAndReturn(func(_ context.Context, req *serviceports.ImportBankReceiptRequest, _ *serviceports.RequestActor) (*bankreceipt.BankReceipt, error) {
			require.False(t, req.BatchID.IsNil())
			require.True(t, req.SkipAudit)
			receipt := *returnedReceipts[0]
			returnedReceipts = returnedReceipts[1:]
			receipt.ImportBatchID = req.BatchID
			return &receipt, nil
		}).
		Twice()
	auditSvc.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptBatchRequest{
			Source:    "csv",
			Reference: "BATCH-1",
			Receipts: []*serviceports.ImportBankReceiptBatchLine{
				{ReceiptDate: 1, AmountMinor: 10000, ReferenceNumber: "A"},
				{ReceiptDate: 2, AmountMinor: 5000, ReferenceNumber: "B"},
			},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		actor,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, bankreceiptbatch.StatusCompleted, result.Batch.Status)
	assert.Equal(t, int64(2), result.Batch.ImportedCount)
	assert.Equal(t, int64(1), result.Batch.MatchedCount)
	assert.Equal(t, int64(1), result.Batch.ExceptionCount)
	assert.Equal(t, int64(15000), result.Batch.ImportedAmountMinor)
	assert.Equal(t, int64(10000), result.Batch.MatchedAmountMinor)
	assert.Equal(t, int64(5000), result.Batch.ExceptionAmountMinor)
	require.Len(t, result.Receipts, 2)
	for _, receipt := range result.Receipts {
		require.Equal(t, result.Batch.ID, receipt.ImportBatchID)
	}
}

func TestImportBatchReturnsErrorWhenReceiptImportFails(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockBankReceiptBatchRepository(t)
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	db := fakeBatchDB{}
	auditSvc := mocks.NewMockAuditService(t)
	bankSvc := mocks.NewMockBankReceiptService(t)
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *bankreceiptbatch.BankReceiptBatch,
		) (*bankreceiptbatch.BankReceiptBatch, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brib_")
			}
			return &copy, nil
		}).
		Once()
	svc := New(
		Params{
			Logger:             zap.NewNop(),
			DB:                 db,
			Repo:               repo,
			ReceiptRepo:        receiptRepo,
			BankReceiptService: bankSvc,
			AuditService:       auditSvc,
		},
	)
	actor := testutil.NewSessionActor(userID, orgID, buID)

	bankSvc.EXPECT().Import(mock.Anything, mock.Anything, actor).Return(nil, assert.AnError).Once()

	result, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptBatchRequest{
			Source:    "csv",
			Reference: "BATCH-ERR",
			Receipts: []*serviceports.ImportBankReceiptBatchLine{
				{ReceiptDate: 1, AmountMinor: 10000, ReferenceNumber: "A"},
			},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		actor,
	)

	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, result)
	auditSvc.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

type fakeBatchDB struct{}

func (fakeBatchDB) DB() *bun.DB { return nil }

func (fakeBatchDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeBatchDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func (fakeBatchDB) HealthCheck(context.Context) error { return nil }

func (fakeBatchDB) IsHealthy(context.Context) bool { return true }

func (fakeBatchDB) Close() error { return nil }
