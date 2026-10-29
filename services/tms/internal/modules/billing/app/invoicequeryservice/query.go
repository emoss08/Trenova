package invoicequeryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListEmailAttempts(
	ctx context.Context,
	req invoiceports.ListInvoiceEmailAttemptsRequest,
) (*pagination.ListResult[*invoice.EmailAttempt], error) {
	return s.repo.ListEmailAttempts(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *invoiceports.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *invoiceports.ListInvoiceConnectionRequest,
) (*pagination.CursorListResult[*invoice.Invoice], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req invoiceports.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	return s.repo.GetByIDs(ctx, req)
}
