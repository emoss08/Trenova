package customertrackingservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB            kernelports.DBConnection
	Logger        *zap.Logger
	Config        *config.Config
	Profiles      customertrackingports.ProfileRepository
	Links         customertrackingports.LinkRepository
	Notifications customertrackingports.NotificationRepository
	Facts         customertrackingports.FactRepository
	ShipmentRepo  shipmentports.ShipmentRepository
	StopEtas      stopetaports.Repository
	CustomerRepo  customerports.CustomerRepository
	OrgRepo       tenantports.OrganizationRepository             `optional:"true"`
	Templates     documenttemplateports.DocumentTemplateResolver `optional:"true"`
	EmailService  emailports.EmailService                        `optional:"true"`
	Documents     documentports.InvoiceDocumentService           `optional:"true"`
	AuditService  auditports.AuditService                        `optional:"true"`
}

type Service struct {
	db            kernelports.DBConnection
	l             *zap.Logger
	cfg           *config.Config
	profiles      customertrackingports.ProfileRepository
	links         customertrackingports.LinkRepository
	notifications customertrackingports.NotificationRepository
	facts         customertrackingports.FactRepository
	shipmentRepo  shipmentports.ShipmentRepository
	stopEtas      stopetaports.Repository
	customerRepo  customerports.CustomerRepository
	orgRepo       tenantports.OrganizationRepository
	templates     documenttemplateports.DocumentTemplateResolver
	emailService  emailports.EmailService
	documents     documentports.InvoiceDocumentService
	auditService  auditports.AuditService
	now           func() int64
}

func New(p Params) *Service { //nolint:gocritic // dependency injection
	return &Service{
		db:            p.DB,
		l:             p.Logger.Named("service.customer-tracking"),
		cfg:           p.Config,
		profiles:      p.Profiles,
		links:         p.Links,
		notifications: p.Notifications,
		facts:         p.Facts,
		shipmentRepo:  p.ShipmentRepo,
		stopEtas:      p.StopEtas,
		customerRepo:  p.CustomerRepo,
		orgRepo:       p.OrgRepo,
		templates:     p.Templates,
		emailService:  p.EmailService,
		documents:     p.Documents,
		auditService:  p.AuditService,
		now:           timeutils.NowUnix,
	}
}

var (
	_ customertrackingports.CustomerTrackingService = (*Service)(nil)
	_ customertrackingports.StatusNotifier          = (*Service)(nil)
)

func (s *Service) canEmail() bool {
	return s.templates != nil && s.emailService != nil
}
