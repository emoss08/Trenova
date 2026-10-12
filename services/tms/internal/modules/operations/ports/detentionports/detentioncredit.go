package detentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DetentionWaiver struct {
	OccurrenceID pulid.ID               `json:"occurrenceId"`
	Reason       detention.WaiverReason `json:"reason"`
	Note         string                 `json:"note"`
	RequestedBy  pulid.ID               `json:"requestedBy"`
}

type CreditBilledDetentionRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	ShipmentID pulid.ID
	ChargeID   pulid.ID
	Reason     string
	Waiver     DetentionWaiver
}

type DetentionCreditStatus string

const (
	DetentionCreditIssued          = DetentionCreditStatus("Issued")
	DetentionCreditPendingApproval = DetentionCreditStatus("PendingApproval")
)

type DetentionCreditInvoice struct {
	AdjustmentID     pulid.ID
	InvoiceID        pulid.ID
	InvoiceNumber    string
	CreditMemoNumber string
	Amount           decimal.Decimal
	Status           DetentionCreditStatus
}

type DetentionCredit struct {
	Amount   decimal.Decimal
	Currency string
	Invoices []DetentionCreditInvoice
}

func (c *DetentionCredit) Pending() bool {
	if c == nil {
		return false
	}

	for i := range c.Invoices {
		if c.Invoices[i].Status == DetentionCreditPendingApproval {
			return true
		}
	}

	return false
}

func (c *DetentionCredit) InvoiceNumbers() []string {
	numbers := make([]string, 0, len(c.Invoices))
	for i := range c.Invoices {
		numbers = append(numbers, c.Invoices[i].InvoiceNumber)
	}

	return numbers
}

func (c *DetentionCredit) CreditMemoNumbers() []string {
	numbers := make([]string, 0, len(c.Invoices))
	for i := range c.Invoices {
		if c.Invoices[i].CreditMemoNumber != "" {
			numbers = append(numbers, c.Invoices[i].CreditMemoNumber)
		}
	}

	return numbers
}

type BilledDetentionCreditor interface {
	CreditBilledCharge(
		ctx context.Context,
		req *CreditBilledDetentionRequest,
	) (*DetentionCredit, error)
}
