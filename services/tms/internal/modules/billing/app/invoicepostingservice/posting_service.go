//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoicepostingservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceediservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepdfservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	AccountingPolicy    *accountingcontrolpolicyservice.Service
	AccountingRepo      tenantports.AccountingControlRepository
	AccountingSync      accountingsyncports.AccountingSyncEnqueuer     `optional:"true"`
	AdjustmentCtrlRepo  tenantports.InvoiceAdjustmentControlRepository `optional:"true"`
	AgreementRepo       factoringports.AgreementRepository             `optional:"true"`
	AuditService        auditports.AuditService
	BillingPolicy       *billingcontrolpolicyservice.Service
	BillingQueueRepo    billingqueueports.BillingQueueRepository
	BillingRepo         tenantports.BillingControlRepository
	CustomerLedgerRepo  customerledgerports.CustomerLedgerProjectionRepository
	CustomerRepo        customerports.CustomerRepository
	DB                  kernelports.Transactor
	EDI                 *invoiceediservice.Service
	FactoringRepo       factoringcompanyports.FactoringCompanyRepository
	FiscalPeriodRepo    fiscalperiodports.FiscalPeriodRepository
	GLAccounts          glaccountports.GLAccountRepository `optional:"true"`
	JournalRepo         journalentryports.JournalPostingRepository
	NotificationService *notificationservice.Service
	OrderDerivation     orderports.OrderDerivationService
	PDF                 *invoicepdfservice.Service
	Permissions         permissionports.PermissionEngine
	Realtime            realtimeports.RealtimeService
	Repo                invoiceports.InvoiceRepository
	SequenceGenerator   seqgen.Generator
	ShipmentRepo        shipmentports.ShipmentRepository
	Stamper             *exchangeratestamp.Stamper
	Validator           *invoicevalidation.Validator
	WorkflowStarter     kernelports.WorkflowStarter
}

// Service posts invoices: the ledger entry, the records posting moves, its preview and the
// auto-post path.
type Service struct {
	accountingPolicy    *accountingcontrolpolicyservice.Service
	accountingRepo      tenantports.AccountingControlRepository
	accountingSync      accountingsyncports.AccountingSyncEnqueuer
	adjustmentCtrlRepo  tenantports.InvoiceAdjustmentControlRepository
	agreementRepo       factoringports.AgreementRepository
	auditService        auditports.AuditService
	billingPolicy       *billingcontrolpolicyservice.Service
	billingQueueRepo    billingqueueports.BillingQueueRepository
	billingRepo         tenantports.BillingControlRepository
	customerLedgerRepo  customerledgerports.CustomerLedgerProjectionRepository
	customerRepo        customerports.CustomerRepository
	db                  kernelports.Transactor
	edi                 *invoiceediservice.Service
	factoringRepo       factoringcompanyports.FactoringCompanyRepository
	fiscalPeriodRepo    fiscalperiodports.FiscalPeriodRepository
	glAccounts          glaccountports.GLAccountRepository
	journalRepo         journalentryports.JournalPostingRepository
	l                   *zap.Logger
	notificationService *notificationservice.Service
	orderDerivation     orderports.OrderDerivationService
	pdf                 *invoicepdfservice.Service
	permissions         permissionports.PermissionEngine
	realtime            realtimeports.RealtimeService
	rec                 *invoiceaction.Recorder
	repo                invoiceports.InvoiceRepository
	sequenceGenerator   seqgen.Generator
	shipmentRepo        shipmentports.ShipmentRepository
	stamper             *exchangeratestamp.Stamper
	validator           *invoicevalidation.Validator
	workflowStarter     kernelports.WorkflowStarter
}

var _ invoiceports.InvoicePostingService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		accountingPolicy:    p.AccountingPolicy,
		accountingRepo:      p.AccountingRepo,
		accountingSync:      p.AccountingSync,
		adjustmentCtrlRepo:  p.AdjustmentCtrlRepo,
		agreementRepo:       p.AgreementRepo,
		auditService:        p.AuditService,
		billingPolicy:       p.BillingPolicy,
		billingQueueRepo:    p.BillingQueueRepo,
		billingRepo:         p.BillingRepo,
		customerLedgerRepo:  p.CustomerLedgerRepo,
		customerRepo:        p.CustomerRepo,
		db:                  p.DB,
		edi:                 p.EDI,
		factoringRepo:       p.FactoringRepo,
		fiscalPeriodRepo:    p.FiscalPeriodRepo,
		glAccounts:          p.GLAccounts,
		journalRepo:         p.JournalRepo,
		l:                   l,
		notificationService: p.NotificationService,
		orderDerivation:     p.OrderDerivation,
		pdf:                 p.PDF,
		permissions:         p.Permissions,
		realtime:            p.Realtime,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       l,
				AuditService: p.AuditService,
				Realtime:     p.Realtime,
			},
		),
		repo:              p.Repo,
		sequenceGenerator: p.SequenceGenerator,
		shipmentRepo:      p.ShipmentRepo,
		stamper:           p.Stamper,
		validator:         p.Validator,
		workflowStarter:   p.WorkflowStarter,
	}
}
