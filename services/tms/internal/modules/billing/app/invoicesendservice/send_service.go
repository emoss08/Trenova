package invoicesendservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceediservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	BillingRepo      tenantports.BillingControlRepository
	CustomerRepo     customerports.CustomerRepository
	DocumentService  documentports.InvoiceDocumentService `optional:"true"`
	EDI              *invoiceediservice.Service
	EmailRepo        emailports.EmailRepository
	EmailSenders     emailports.EmailSenderResolver `optional:"true"`
	EmailService     emailports.EmailService
	FactoringRepo    factoringcompanyports.FactoringCompanyRepository
	OrganizationRepo tenantports.OrganizationRepository
	Repo             invoiceports.InvoiceRepository
	ShipmentRepo     shipmentports.ShipmentRepository
	Templates        documenttemplateports.DocumentTemplateResolver
	WorkflowStarter  kernelports.WorkflowStarter
	AuditService     auditports.AuditService
	Realtime         realtimeports.RealtimeService
	AppConfig        *appconfig.Config `optional:"true"`
}

// Service emails invoices and serves their shared documents.
type Service struct {
	billingRepo      tenantports.BillingControlRepository
	customerRepo     customerports.CustomerRepository
	documentService  documentports.InvoiceDocumentService
	edi              *invoiceediservice.Service
	emailRepo        emailports.EmailRepository
	emailSenders     emailports.EmailSenderResolver
	emailService     emailports.EmailService
	factoringRepo    factoringcompanyports.FactoringCompanyRepository
	l                *zap.Logger
	organizationRepo tenantports.OrganizationRepository
	rec              *invoiceaction.Recorder
	repo             invoiceports.InvoiceRepository
	shipmentRepo     shipmentports.ShipmentRepository
	templates        documenttemplateports.DocumentTemplateResolver
	webBaseURL       string
	workflowStarter  kernelports.WorkflowStarter
}

var _ invoiceports.InvoiceSendService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		billingRepo:      p.BillingRepo,
		customerRepo:     p.CustomerRepo,
		documentService:  p.DocumentService,
		edi:              p.EDI,
		emailRepo:        p.EmailRepo,
		emailSenders:     p.EmailSenders,
		emailService:     p.EmailService,
		factoringRepo:    p.FactoringRepo,
		l:                l,
		organizationRepo: p.OrganizationRepo,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       l,
				AuditService: p.AuditService,
				Realtime:     p.Realtime,
			},
		),
		repo:            p.Repo,
		shipmentRepo:    p.ShipmentRepo,
		templates:       p.Templates,
		webBaseURL:      webBaseURL(p.AppConfig),
		workflowStarter: p.WorkflowStarter,
	}
}

func webBaseURL(cfg *appconfig.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.GetWebBaseURL()
}
