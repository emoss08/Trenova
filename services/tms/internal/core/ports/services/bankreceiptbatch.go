package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
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
	Batch    *bankreceiptbatch.Batch `json:"batch"`
	Receipts []*bankreceipt.Receipt  `json:"receipts"`
}

type BankReceiptBatchService interface {
	Get(ctx context.Context, req *GetBankReceiptBatchRequest) (*BankReceiptBatchResult, error)
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*bankreceiptbatch.Batch, error)
	Import(ctx context.Context, req *ImportBankReceiptBatchRequest, actor *RequestActor) (*BankReceiptBatchResult, error)
	DistinctSources(ctx context.Context, req *pagination.SelectQueryRequest) (*pagination.ListResult[*bankreceiptbatch.SourceOption], error)
}
