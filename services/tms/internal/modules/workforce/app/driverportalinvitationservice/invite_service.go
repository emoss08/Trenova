package driverportalinvitationservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/portalconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	PortalConfig *portalconfig.Config
	PortalRepo   portalaccessports.PortalAccessRepository
	OrgRepo      tenantports.OrganizationRepository
	EmailService emailports.EmailService
	Templates    documenttemplateports.DocumentTemplateResolver
	AuditService auditports.AuditService
	Sessions     sessionports.SessionRepository
}

type Service struct {
	l            *zap.Logger
	cfg          *portalconfig.Config
	portalRepo   portalaccessports.PortalAccessRepository
	orgRepo      tenantports.OrganizationRepository
	emailService emailports.EmailService
	templates    documenttemplateports.DocumentTemplateResolver
	auditService auditports.AuditService
	sessions     sessionports.SessionRepository
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.driver-portal"),
		cfg:          p.PortalConfig,
		portalRepo:   p.PortalRepo,
		orgRepo:      p.OrgRepo,
		emailService: p.EmailService,
		templates:    p.Templates,
		auditService: p.AuditService,
		sessions:     p.Sessions,
	}
}
