package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/invoicecontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// InvoiceControlOps is the service for invoice control settings
type InvoiceControlOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewInvoiceControlOps creates a new invoice control service
func NewInvoiceControlOps(ctx context.Context) *InvoiceControlOps {
	return &InvoiceControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetInvoiceControlByOrgID creates a new invoice control settings for an organization
func (r *InvoiceControlOps) GetInvoiceControlByOrgID(orgID uuid.UUID) (*ent.InvoiceControl, error) {
	invoiceControl, err := r.client.InvoiceControl.Query().Where(
		invoicecontrol.HasOrganizationWith(
			organization.ID(orgID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return invoiceControl, nil
}

// UpdateInvoiceControl updates the invoice control settings for an organization
func (r *InvoiceControlOps) UpdateInvoiceControl(ic ent.InvoiceControl) (*ent.InvoiceControl, error) {
	updateIC, err := r.client.InvoiceControl.
		UpdateOneID(ic.ID).
		SetInvoiceNumberPrefix(ic.InvoiceNumberPrefix).
		SetCreditMemoNumberPrefix(ic.CreditMemoNumberPrefix).
		SetInvoiceTerms(*ic.InvoiceTerms).
		SetInvoiceFooter(*ic.InvoiceFooter).
		SetInvoiceLogoURL(*ic.InvoiceLogoURL).
		SetInvoiceDateFormat(ic.InvoiceDateFormat).
		SetInvoiceDueAfterDays(ic.InvoiceDueAfterDays).
		SetInvoiceLogoWidth(ic.InvoiceLogoWidth).
		SetShowAmountDue(ic.ShowAmountDue).
		SetAttachPdf(ic.AttachPdf).
		SetShowInvoiceDueDate(ic.ShowInvoiceDueDate).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateIC, nil
}
