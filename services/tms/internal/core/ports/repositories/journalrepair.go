package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListJournalRepairRequest struct {
	OrganizationID pulid.ID `json:"organizationId"`
	AfterID        pulid.ID `json:"afterId"`
	Limit          int      `json:"limit"`
}

type AdjustmentMemoRepair struct {
	Memo            *invoice.Invoice       `json:"memo"`
	Kind            invoiceadjustment.Kind `json:"kind"`
	SourceInvoiceID pulid.ID               `json:"sourceInvoiceId"`
	Journaled       bool                   `json:"journaled"`
	SourceMissing   bool                   `json:"sourceMissing"`
}

type DriverPaymentRepair struct {
	Settlement     *driversettlement.Settlement `json:"settlement"`
	JournalBatchID pulid.ID                     `json:"journalBatchId"`
}

type SetDriverSettlementPaidBatchParams struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	SettlementID pulid.ID              `json:"settlementId"`
	BatchID      pulid.ID              `json:"batchId"`
}

type JournalRepairRepository interface {
	ListUnjournaledAdjustmentMemos(
		ctx context.Context,
		req *ListJournalRepairRequest,
	) ([]*AdjustmentMemoRepair, error)
	ListUnjournaledDriverPayments(
		ctx context.Context,
		req *ListJournalRepairRequest,
	) ([]*DriverPaymentRepair, error)
	SetDriverSettlementPaidBatch(
		ctx context.Context,
		params *SetDriverSettlementPaidBatchParams,
	) error
}
