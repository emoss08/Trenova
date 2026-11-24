package bankreceiptports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReceiptsTouchingRequest struct {
	TenantInfo pagination.TenantInfo
	SubjectIDs []pulid.ID
}

type ReplaceCandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	ReceiptIDs []pulid.ID
	Candidates []*bankreceipt.Candidate
}

type ListCandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	ReceiptID  pulid.ID
	Kinds      []bankreceipt.CandidateKind
}

type QueueUnappliedRequest struct {
	Kind string
}

type CandidateView struct {
	Candidate     *bankreceipt.Candidate
	Reference     string
	InvoiceNumber string
	CustomerName  string
}

type CandidateRepository interface {
	ReceiptsTouching(ctx context.Context, req *ReceiptsTouchingRequest) ([]*bankreceipt.BankReceipt, error)
	PaymentEvidence(
		ctx context.Context,
		receipts []*bankreceipt.BankReceipt,
	) (map[pulid.ID][]bankreceipt.PaymentEvidence, error)
	OpenInvoicesByNumber(
		ctx context.Context,
		numbers []string,
	) ([]bankreceipt.InvoiceEvidence, error)
	Replace(ctx context.Context, req *ReplaceCandidatesRequest) error
	ListForReceipt(ctx context.Context, req *ListCandidatesRequest) ([]*CandidateView, error)
	QueueUnapplied(ctx context.Context, req *QueueUnappliedRequest) (int64, error)
}

type CandidateRefresher interface {
	Refresh(ctx context.Context, tenant pagination.TenantInfo, subjectIDs []pulid.ID) (int, error)
	RefreshReceipt(
		ctx context.Context,
		receipt *bankreceipt.BankReceipt,
	) ([]*bankreceipt.Candidate, error)
}

type CandidateWaker interface {
	Wake(ctx context.Context, tenant pagination.TenantInfo)
}
