package invoiceadjustmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 ports.DBConnection
	Repo               repositories.InvoiceAdjustmentRepository
	InvoiceRepo        repositories.InvoiceRepository
	CustomerRepo       repositories.CustomerRepository
	BillingQueueRepo   repositories.BillingQueueRepository
	ShipmentRepo       repositories.ShipmentRepository
	AccessorialRepo    repositories.AccessorialChargeRepository
	ShipmentCtrlRepo   repositories.ShipmentControlRepository
	BillingCtrlRepo    repositories.BillingControlRepository
	AdjustmentCtrlRepo repositories.InvoiceAdjustmentControlRepository
	AccountingRepo     repositories.AccountingControlRepository
	Stamper            *exchangeratestamp.Stamper
	JournalRepo        repositories.JournalPostingRepository
	FiscalPeriodRepo   repositories.FiscalPeriodRepository
	DocumentRepo       repositories.DocumentRepository
	OrderRepo          repositories.OrderRepository            `optional:"true"`
	ChargeAllocRepo    repositories.ChargeAllocationRepository `optional:"true"`
	OrderDerivation    servicesports.OrderDerivationService    `optional:"true"`
	DetentionBilling   servicesports.DetentionBillingService
	Validator          *Validator
	AuditService       servicesports.AuditService
	WorkflowStarter    servicesports.WorkflowStarter
	Commercial         *shipmentcommercial.Calculator
	Generator          servicesports.InvoiceAdjustGenerator
	SequenceGenerator  seqgen.Generator
	AccountingSync     servicesports.AccountingSyncEnqueuer `optional:"true"`
	CustomerLedgerRepo repositories.CustomerLedgerProjectionRepository
	AccountingPolicy   *accountingcontrolpolicyservice.Service
}

type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	repo               repositories.InvoiceAdjustmentRepository
	invoiceRepo        repositories.InvoiceRepository
	customerRepo       repositories.CustomerRepository
	billingQueueRepo   repositories.BillingQueueRepository
	shipmentRepo       repositories.ShipmentRepository
	accessorialRepo    repositories.AccessorialChargeRepository
	shipmentCtrlRepo   repositories.ShipmentControlRepository
	billingCtrlRepo    repositories.BillingControlRepository
	adjustmentCtrlRepo repositories.InvoiceAdjustmentControlRepository
	accountingRepo     repositories.AccountingControlRepository
	stamper            *exchangeratestamp.Stamper
	journalRepo        repositories.JournalPostingRepository
	fiscalPeriodRepo   repositories.FiscalPeriodRepository
	documentRepo       repositories.DocumentRepository
	orderRepo          repositories.OrderRepository
	chargeAllocRepo    repositories.ChargeAllocationRepository
	orderDerivation    servicesports.OrderDerivationService
	detentionBilling   servicesports.DetentionBillingService
	validator          *Validator
	auditService       servicesports.AuditService
	workflowStarter    servicesports.WorkflowStarter
	commercial         *shipmentcommercial.Calculator
	generator          servicesports.InvoiceAdjustGenerator
	sequenceGenerator  seqgen.Generator
	accountingSync     servicesports.AccountingSyncEnqueuer
	customerLedgerRepo repositories.CustomerLedgerProjectionRepository
	accountingPolicy   *accountingcontrolpolicyservice.Service
}

type previewComputation struct {
	invoice           *invoice.Invoice
	correctionGroupID pulid.ID
	control           *tenant.InvoiceAdjustmentControl
	accountingControl *tenant.AccountingControl
	preview           *servicesports.InvoiceAdjustmentPreview
	lines             []*invoiceadjustment.InvoiceAdjustmentLine
	creditLineItems   []*invoice.InvoiceLine
	replacementLines  []*invoice.InvoiceLine
}

type previewLineValuesRequest struct {
	sourceLine *invoice.InvoiceLine
	input      *servicesports.InvoiceAdjustmentLineInput
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

func New(p Params) servicesports.InvoiceAdjustmentService { //nolint:gocritic // stable API shape
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
		accountingSync:     p.AccountingSync,
		customerLedgerRepo: p.CustomerLedgerRepo,
		accountingPolicy:   p.AccountingPolicy,
	}
}
