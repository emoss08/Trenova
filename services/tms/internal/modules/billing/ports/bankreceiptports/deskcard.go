package bankreceiptports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDeskCardRequest struct {
	TenantInfo      pagination.TenantInfo
	ReceiptID       pulid.ID
	IncludePayment  bool
	IncludeInvoices bool
	IncludeWorkItem bool
}

type DeskCard struct {
	Receipt            *bankreceipt.BankReceipt `json:"receipt"`
	Payment            *DeskCardPayment         `json:"payment"`
	PaymentHidden      bool                     `json:"paymentHidden"`
	WorkItem           *DeskCardWorkItem        `json:"workItem"`
	WorkItemShown      bool                     `json:"workItemShown"`
	Candidates         []DeskCardCandidate      `json:"candidates"`
	CandidatesWithheld []string                 `json:"candidatesWithheld"`
}

type DeskCardCandidate struct {
	Rank             int      `json:"rank"`
	Kind             string   `json:"kind"`
	InvoiceID        pulid.ID `json:"invoiceId"`
	InvoiceNumber    string   `json:"invoiceNumber"`
	PaymentID        pulid.ID `json:"paymentId"`
	PaymentReference string   `json:"paymentReference"`
	CustomerName     string   `json:"customerName"`
	Standing         string   `json:"standing"`
	Evidence         []string `json:"evidence"`
	OpenMinor        int64    `json:"openMinor"`
	DifferenceMinor  int64    `json:"differenceMinor"`
}

type DeskCardPayment struct {
	ID                   pulid.ID              `json:"id"`
	CustomerID           pulid.ID              `json:"customerId"`
	CustomerName         string                `json:"customerName"`
	PaymentMethod        string                `json:"paymentMethod"`
	ReferenceNumber      string                `json:"referenceNumber"`
	PaymentDate          int64                 `json:"paymentDate"`
	Status               string                `json:"status"`
	CurrencyCode         string                `json:"currencyCode"`
	AmountMinor          int64                 `json:"amountMinor"`
	AppliedAmountMinor   int64                 `json:"appliedAmountMinor"`
	UnappliedAmountMinor int64                 `json:"unappliedAmountMinor"`
	Applications         []DeskCardApplication `json:"applications"`
}

type DeskCardApplication struct {
	InvoiceID           pulid.ID `json:"invoiceId"`
	InvoiceNumber       string   `json:"invoiceNumber"`
	AppliedAmountMinor  int64    `json:"appliedAmountMinor"`
	ShortPayAmountMinor int64    `json:"shortPayAmountMinor"`
}

type DeskCardWorkItem struct {
	ID             pulid.ID `json:"id"`
	Status         string   `json:"status"`
	AssignedTo     string   `json:"assignedTo"`
	AssignedAt     *int64   `json:"assignedAt"`
	ResolutionType string   `json:"resolutionType"`
	ResolutionNote string   `json:"resolutionNote"`
}

type DeskCardReader interface {
	GetDeskCard(ctx context.Context, req *GetDeskCardRequest) (*DeskCard, error)
}
