//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoicepdfservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	BillingRepo      tenantports.BillingControlRepository
	ContextBuilder   *invoicedocument.ContextBuilder
	CustomerRepo     customerports.CustomerRepository
	FactoringRepo    factoringcompanyports.FactoringCompanyRepository
	OrganizationRepo tenantports.OrganizationRepository
	Repo             invoiceports.InvoiceRepository
	ShipmentRepo     shipmentports.ShipmentRepository
	Templates        documenttemplateports.DocumentTemplateResolver
	WorkflowStarter  kernelports.WorkflowStarter
	AuditService     auditports.AuditService
	Realtime         realtimeports.RealtimeService
}

// Service renders an invoice's preview and generates and attaches its PDF.
type Service struct {
	billingRepo      tenantports.BillingControlRepository
	contextBuilder   *invoicedocument.ContextBuilder
	customerRepo     customerports.CustomerRepository
	factoringRepo    factoringcompanyports.FactoringCompanyRepository
	organizationRepo tenantports.OrganizationRepository
	rec              *invoiceaction.Recorder
	repo             invoiceports.InvoiceRepository
	shipmentRepo     shipmentports.ShipmentRepository
	templates        documenttemplateports.DocumentTemplateResolver
	workflowStarter  kernelports.WorkflowStarter
	l                *zap.Logger
}

var _ invoiceports.InvoicePDFService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		billingRepo:      p.BillingRepo,
		contextBuilder:   p.ContextBuilder,
		customerRepo:     p.CustomerRepo,
		factoringRepo:    p.FactoringRepo,
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
		workflowStarter: p.WorkflowStarter,
		l:               l,
	}
}
