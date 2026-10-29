package invoiceadjustmentdraftservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB          kernelports.Transactor
	Repo        invoiceadjustmentports.InvoiceAdjustmentRepository
	InvoiceRepo invoiceports.InvoiceRepository
	Validator   *invoiceadjustmentengine.Validator
	Engine      *invoiceadjustmentengine.Engine
}

type Service struct {
	db          kernelports.Transactor
	repo        invoiceadjustmentports.InvoiceAdjustmentRepository
	invoiceRepo invoiceports.InvoiceRepository
	validator   *invoiceadjustmentengine.Validator
	engine      *invoiceadjustmentengine.Engine
}

var _ invoiceadjustmentports.InvoiceAdjustmentDraftService = (*Service)(nil)

func New(p Params) invoiceadjustmentports.InvoiceAdjustmentDraftService {
	return &Service{
		db:          p.DB,
		repo:        p.Repo,
		invoiceRepo: p.InvoiceRepo,
		validator:   p.Validator,
		engine:      p.Engine,
	}
}
