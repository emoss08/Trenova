package invoiceadjustmentservice

import (
	"context"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 kernelports.DBConnection
	Repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	InvoiceRepo        invoiceports.InvoiceRepository
	CustomerRepo       customerports.CustomerRepository
	BillingQueueRepo   billingqueueports.BillingQueueRepository
	ShipmentRepo       shipmentports.ShipmentRepository
	AccessorialRepo    accessorialchargeports.AccessorialChargeRepository
	ShipmentCtrlRepo   tenantports.ShipmentControlRepository
	BillingCtrlRepo    tenantports.BillingControlRepository
	AdjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	AccountingRepo     tenantports.AccountingControlRepository
	Stamper            *exchangeratestamp.Stamper
	JournalRepo        journalentryports.JournalPostingRepository
	FiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	DocumentRepo       documentports.DocumentRepository
	OrderRepo          orderports.OrderRepository               `optional:"true"`
	ChargeAllocRepo    shipmentports.ChargeAllocationRepository `optional:"true"`
	OrderDerivation    orderports.OrderDerivationService        `optional:"true"`
	DetentionBilling   detentionports.DetentionBillingService
	Validator          *Validator
	AuditService       auditports.AuditService
	WorkflowStarter    kernelports.WorkflowStarter
	Commercial         *shipmentcommercial.Calculator
	Generator          invoiceadjustmentports.InvoiceAdjustGenerator
	SequenceGenerator  seqgen.Generator
	SequenceFormats    seqgen.FormatProvider                      `optional:"true"`
	AccountingSync     accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
	CustomerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	AccountingPolicy   *accountingcontrolpolicyservice.Service
	Realtime           agentports.RealtimeService `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	db                 kernelports.DBConnection
	repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	invoiceRepo        invoiceports.InvoiceRepository
	customerRepo       customerports.CustomerRepository
	billingQueueRepo   billingqueueports.BillingQueueRepository
	shipmentRepo       shipmentports.ShipmentRepository
	accessorialRepo    accessorialchargeports.AccessorialChargeRepository
	shipmentCtrlRepo   tenantports.ShipmentControlRepository
	billingCtrlRepo    tenantports.BillingControlRepository
	adjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	accountingRepo     tenantports.AccountingControlRepository
	stamper            *exchangeratestamp.Stamper
	journalRepo        journalentryports.JournalPostingRepository
	fiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	documentRepo       documentports.DocumentRepository
	orderRepo          orderports.OrderRepository
	chargeAllocRepo    shipmentports.ChargeAllocationRepository
	orderDerivation    orderports.OrderDerivationService
	detentionBilling   detentionports.DetentionBillingService
	validator          *Validator
	auditService       auditports.AuditService
	workflowStarter    kernelports.WorkflowStarter
	commercial         *shipmentcommercial.Calculator
	generator          invoiceadjustmentports.InvoiceAdjustGenerator
	sequenceGenerator  seqgen.Generator
	sequenceFormats    seqgen.FormatProvider
	accountingSync     accountingsyncports.AccountingSyncEnqueuer
	customerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	accountingPolicy   *accountingcontrolpolicyservice.Service
	realtime           agentports.RealtimeService
}

type previewComputation struct {
	invoice           *invoice.Invoice
	correctionGroupID pulid.ID
	control           *tenant.InvoiceAdjustmentControl
	accountingControl *tenant.AccountingControl
	preview           *invoiceadjustmentports.InvoiceAdjustmentPreview
	lines             []*invoiceadjustment.InvoiceAdjustmentLine
	creditLineItems   []*invoice.InvoiceLine
	replacementLines  []*invoice.InvoiceLine
}

type previewLineValuesRequest struct {
	sourceLine *invoice.InvoiceLine
	input      *invoiceadjustmentports.InvoiceAdjustmentLineInput
	kind       invoiceadjustment.Kind
	eligible   decimal.Decimal
}

type previewLineValues struct {
	creditAmount   decimal.Decimal
	creditQuantity decimal.Decimal
	rebillAmount   decimal.Decimal
	rebillQuantity decimal.Decimal
	description    string
	payload        map[string]any
}

//nolint:gocritic // stable API shape
func New(p Params) invoiceadjustmentports.InvoiceAdjustmentService {
	return &Service{
		l:                  p.Logger.Named("service.invoice-adjustment"),
		db:                 p.DB,
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		customerRepo:       p.CustomerRepo,
		billingQueueRepo:   p.BillingQueueRepo,
		shipmentRepo:       p.ShipmentRepo,
		accessorialRepo:    p.AccessorialRepo,
		shipmentCtrlRepo:   p.ShipmentCtrlRepo,
		billingCtrlRepo:    p.BillingCtrlRepo,
		adjustmentCtrlRepo: p.AdjustmentCtrlRepo,
		accountingRepo:     p.AccountingRepo,
		stamper:            p.Stamper,
		journalRepo:        p.JournalRepo,
		fiscalPeriodRepo:   p.FiscalPeriodRepo,
		documentRepo:       p.DocumentRepo,
		orderRepo:          p.OrderRepo,
		chargeAllocRepo:    p.ChargeAllocRepo,
		orderDerivation:    p.OrderDerivation,
		detentionBilling:   p.DetentionBilling,
		validator:          p.Validator,
		auditService:       p.AuditService,
		workflowStarter:    p.WorkflowStarter,
		commercial:         p.Commercial,
		generator:          p.Generator,
		sequenceGenerator:  p.SequenceGenerator,
		sequenceFormats:    p.SequenceFormats,
		accountingSync:     p.AccountingSync,
		customerLedgerRepo: p.CustomerLedgerRepo,
		accountingPolicy:   p.AccountingPolicy,
		realtime:           p.Realtime,
	}
}

func (s *Service) publishAdjustedInvoices(
	ctx context.Context,
	actor *permissionports.RequestActor,
	invoices ...*invoice.Invoice,
) {
	auditActor := actor.AuditActorOrSystem()
	for _, entity := range invoices {
		if entity == nil {
			continue
		}
		realtimeinvalidation.PublishAfterCommit(ctx, s.realtime, s.l, realtimeinvalidation.ChangeOf(
			permission.ResourceInvoice.String(),
			entity,
			invoice.RealtimeActionUpdated,
			auditActor,
			invoice.NewRealtimeSnapshot(entity),
		))
	}
}
