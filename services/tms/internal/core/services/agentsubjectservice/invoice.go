package agentsubjectservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	noteCustomerID = "customerId"
	noteNumber     = "number"
	noteStatus     = "status"
	noteResolution = "resolution"
)

func (s *Service) invoice(
	ctx context.Context,
	tenant pagination.TenantInfo,
	invoiceID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectInvoice,
		ID:    invoiceID.String(),
		Label: "Invoice",
	}
	if s.invoices == nil {
		return subject, nil
	}

	found, err := s.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         invoiceID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load invoice: %w", err)
	}

	subject.Label = "Invoice " + found.Number
	subject.Notes = marshalNotes(invoiceNotes(found))

	return subject, nil
}

func invoiceNotes(found *invoice.Invoice) map[string]any {
	return map[string]any{
		noteNumber:          found.Number,
		noteStatus:          found.Status,
		"settlementStatus":  found.SettlementStatus,
		"sendStatus":        found.SendStatus,
		"disputeStatus":     found.DisputeStatus,
		"billTo":            found.BillToName,
		noteCustomerID:      found.CustomerID,
		"shipmentId":        found.ShipmentID,
		"shipmentProNumber": found.ShipmentProNumber,
		"invoiceDate":       found.InvoiceDate,
		"dueDate":           found.DueDate,
		"postedAt":          found.PostedAt,
		"totalAmount":       found.TotalAmount.String(),
		"appliedAmount":     found.AppliedAmount.String(),
		"openAmount":        found.TotalAmount.Sub(found.AppliedAmount).String(),
	}
}

func (s *Service) invoiceDispute(
	ctx context.Context,
	tenant pagination.TenantInfo,
	disputeID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectInvoiceDispute,
		ID:    disputeID.String(),
		Label: "Invoice dispute",
	}
	if s.disputes == nil {
		return subject, nil
	}

	found, err := s.disputes.GetByID(ctx, repositories.GetInvoiceDisputeByIDRequest{
		ID:         disputeID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load invoice dispute: %w", err)
	}

	notes := map[string]any{
		noteStatus:        found.Status,
		"reasonCode":      found.ReasonCode,
		"disputedAmount":  found.DisputedAmount.String(),
		"notes":           found.Notes,
		"openedAt":        found.OpenedAt,
		"resolvedAt":      found.ResolvedAt,
		noteResolution:    found.Resolution,
		"resolutionNotes": found.ResolutionNotes,
		"invoiceId":       found.InvoiceID,
		noteCustomerID:    found.CustomerID,
	}
	if s.invoices != nil {
		billed, invErr := s.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
			ID:         found.InvoiceID,
			TenantInfo: tenant,
		})
		if invErr != nil {
			return nil, fmt.Errorf("load the disputed invoice: %w", invErr)
		}
		subject.Label = "Dispute on invoice " + billed.Number
		notes["invoice"] = invoiceNotes(billed)
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}
