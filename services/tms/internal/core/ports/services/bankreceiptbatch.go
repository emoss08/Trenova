package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ImportBankReceiptBatchLine struct {
	ReceiptDate     int64  `json:"receiptDate"`
	AmountMinor     int64  `json:"amountMinor"`
	ReferenceNumber string `json:"referenceNumber"`
	Memo            string `json:"memo"`
}

type ImportBankReceiptBatchRequest struct {
	Source     string                        `json:"source"`
	Reference  string                        `json:"reference"`
	Receipts   []*ImportBankReceiptBatchLine `json:"receipts"`
	TenantInfo pagination.TenantInfo         `json:"tenantInfo"`
}

type GetBankReceiptBatchRequest struct {
	BatchID    pulid.ID              `json:"batchId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptBatchResult struct {
	Batch    *bankreceiptbatch.BankReceiptBatch `json:"batch"`
	Receipts []*bankreceipt.BankReceipt         `json:"receipts"`
}

type BankReceiptBatchService interface {
	Get(ctx context.Context, req *GetBankReceiptBatchRequest) (*BankReceiptBatchResult, error)
	List(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceiptbatch.BankReceiptBatch, error)
	Import(
		ctx context.Context,
		req *ImportBankReceiptBatchRequest,
		actor *RequestActor,
	) (*BankReceiptBatchResult, error)
	DistinctSources(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*repositoryports.BankReceiptBatchSourceOption], error)
}
