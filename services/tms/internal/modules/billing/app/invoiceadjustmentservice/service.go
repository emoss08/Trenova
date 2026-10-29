package invoiceadjustmentservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB                 kernelports.Transactor
	Repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	InvoiceRepo        invoiceports.InvoiceRepository
	AdjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	Validator          *invoiceadjustmentengine.Validator
	WorkflowStarter    kernelports.WorkflowStarter
	Engine             *invoiceadjustmentengine.Engine
}

type Service struct {
	db                 kernelports.Transactor
	repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	invoiceRepo        invoiceports.InvoiceRepository
	adjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	validator          *invoiceadjustmentengine.Validator
	workflowStarter    kernelports.WorkflowStarter
	engine             *invoiceadjustmentengine.Engine
}

var _ invoiceadjustmentports.InvoiceAdjustmentService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) invoiceadjustmentports.InvoiceAdjustmentService {
	return &Service{
		db:                 p.DB,
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		adjustmentCtrlRepo: p.AdjustmentCtrlRepo,
		validator:          p.Validator,
		workflowStarter:    p.WorkflowStarter,
		engine:             p.Engine,
	}
}
