package services

import (
	"context"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/invoicecontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// InvoiceControlService is the service for invoice control settings.
type InvoiceControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewInvoiceControlService creates a new invoice control service.
func NewInvoiceControlService(s *api.Server) *InvoiceControlService {
	return &InvoiceControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetInvoiceControl creates a new invoice control settings for an organization.
func (r *InvoiceControlService) GetInvoiceControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.InvoiceControl, error) {
	invoiceControl, err := r.Client.InvoiceControl.Query().Where(
		invoicecontrol.HasOrganizationWith(
			organization.ID(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return invoiceControl, nil
}

// UpdateInvoiceControl updates the invoice control settings for an organization.
func (r *InvoiceControlService) UpdateInvoiceControl(ctx context.Context, ic *ent.InvoiceControl) (*ent.InvoiceControl, error) {
	updatedEntity := new(ent.InvoiceControl)
	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateInvoiceControlEntity(ctx, tx, ic)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *InvoiceControlService) updateInvoiceControlEntity(
	ctx context.Context, tx *ent.Tx, ic *ent.InvoiceControl,
) (*ent.InvoiceControl, error) {
	updateOp := tx.InvoiceControl.UpdateOneID(ic.ID).
		SetInvoiceNumberPrefix(ic.InvoiceNumberPrefix).
		SetCreditMemoNumberPrefix(ic.CreditMemoNumberPrefix).
		SetInvoiceTerms(ic.InvoiceTerms).
		SetInvoiceFooter(ic.InvoiceFooter).
		SetInvoiceLogoURL(ic.InvoiceLogoURL).
		SetInvoiceDateFormat(ic.InvoiceDateFormat).
		SetInvoiceDueAfterDays(ic.InvoiceDueAfterDays).
		SetInvoiceLogoWidth(ic.InvoiceLogoWidth).
		SetShowAmountDue(ic.ShowAmountDue).
		SetAttachPdf(ic.AttachPdf).
		SetShowInvoiceDueDate(ic.ShowInvoiceDueDate)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
