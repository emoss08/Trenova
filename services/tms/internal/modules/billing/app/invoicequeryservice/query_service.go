package invoicequeryservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Repo invoiceports.InvoiceRepository
}

// Service reads invoices and their email attempts.
type Service struct {
	repo invoiceports.InvoiceRepository
}

var _ invoiceports.InvoiceQueryService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		repo: p.Repo,
	}
}
