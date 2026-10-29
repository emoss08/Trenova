package invoiceadjustmentengine

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
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
	Repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	InvoiceRepo        invoiceports.InvoiceRepository
	CustomerRepo       customerports.CustomerRepository
	BillingQueueRepo   billingqueueports.BillingQueueRepository
	ShipmentRepo       shipmentports.ShipmentRepository
	AccessorialRepo    accessorialchargeports.AccessorialChargeRepository
	ShipmentCtrlRepo   tenantports.ShipmentControlRepository
	AdjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	AccountingRepo     tenantports.AccountingControlRepository
	Stamper            *exchangeratestamp.Stamper
	JournalRepo        journalentryports.JournalPostingRepository
	FiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	DocumentRepo       documentports.DocumentRepository
	OrderRepo          orderports.OrderRepository                       `optional:"true"`
	ChargeAllocRepo    chargeallocationports.ChargeAllocationRepository `optional:"true"`
	OrderDerivation    orderports.OrderDerivationService                `optional:"true"`
	DetentionBilling   detentionports.DetentionBillingService
	Validator          *Validator
	AuditService       auditports.AuditService
	Commercial         *shipmentcommercial.Calculator
	Generator          invoiceadjustmentports.InvoiceAdjustGenerator
	SequenceGenerator  seqgen.Generator
	SequenceFormats    seqgen.FormatProvider                       `optional:"true"`
	CreditApplier      customerpaymentports.CustomerPaymentService `optional:"true"`
	AccountingSync     accountingsyncports.AccountingSyncEnqueuer  `optional:"true"`
	CustomerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	AccountingPolicy   *accountingcontrolpolicyservice.Service
	Realtime           realtimeports.RealtimeService `optional:"true"`
}

// Engine computes what an invoice adjustment would do and carries out an
// approved one; submitting, deciding and drafting adjustments all run
// through it.
type Engine struct {
	l                  *zap.Logger
	repo               invoiceadjustmentports.InvoiceAdjustmentRepository
	invoiceRepo        invoiceports.InvoiceRepository
	customerRepo       customerports.CustomerRepository
	billingQueueRepo   billingqueueports.BillingQueueRepository
	shipmentRepo       shipmentports.ShipmentRepository
	accessorialRepo    accessorialchargeports.AccessorialChargeRepository
	shipmentCtrlRepo   tenantports.ShipmentControlRepository
	adjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	accountingRepo     tenantports.AccountingControlRepository
	stamper            *exchangeratestamp.Stamper
	journalRepo        journalentryports.JournalPostingRepository
	fiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	documentRepo       documentports.DocumentRepository
	orderRepo          orderports.OrderRepository
	chargeAllocRepo    chargeallocationports.ChargeAllocationRepository
	orderDerivation    orderports.OrderDerivationService
	detentionBilling   detentionports.DetentionBillingService
	validator          *Validator
	auditService       auditports.AuditService
	commercial         *shipmentcommercial.Calculator
	generator          invoiceadjustmentports.InvoiceAdjustGenerator
	sequenceGenerator  seqgen.Generator
	sequenceFormats    seqgen.FormatProvider
	credits            creditApplier
	accountingSync     accountingsyncports.AccountingSyncEnqueuer
	customerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	accountingPolicy   *accountingcontrolpolicyservice.Service
	realtime           realtimeports.RealtimeService
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Engine {
	engine := &Engine{
		l:                  p.Logger.Named("service.invoice-adjustment"),
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		customerRepo:       p.CustomerRepo,
		billingQueueRepo:   p.BillingQueueRepo,
		shipmentRepo:       p.ShipmentRepo,
		accessorialRepo:    p.AccessorialRepo,
		shipmentCtrlRepo:   p.ShipmentCtrlRepo,
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
		commercial:         p.Commercial,
		generator:          p.Generator,
		sequenceGenerator:  p.SequenceGenerator,
		sequenceFormats:    p.SequenceFormats,
		accountingSync:     p.AccountingSync,
		customerLedgerRepo: p.CustomerLedgerRepo,
		accountingPolicy:   p.AccountingPolicy,
		realtime:           p.Realtime,
	}
	if p.CreditApplier != nil {
		engine.credits = p.CreditApplier
	}

	return engine
}

type PreviewComputation struct {
	Invoice           *invoice.Invoice
	correctionGroupID pulid.ID
	control           *invoiceadjustmentcontrol.InvoiceAdjustmentControl
	accountingControl *accountingcontrol.AccountingControl
	Preview           *invoiceadjustmentports.InvoiceAdjustmentPreview
	Lines             []*invoiceadjustment.InvoiceAdjustmentLine
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

func (s *Engine) PublishAdjustedInvoices(
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
			invoicerealtime.RealtimeActionUpdated,
			auditActor,
			invoicerealtime.NewRealtimeSnapshot(entity),
		))
	}
}

func ReplacementReviewStatus(required bool) invoiceadjustment.ReplacementReviewStatus {
	if required {
		return invoiceadjustment.ReplacementReviewStatusRequired
	}
	return invoiceadjustment.ReplacementReviewStatusNotRequired
}
