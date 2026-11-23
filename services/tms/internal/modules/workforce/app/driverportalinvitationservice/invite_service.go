package driverportalinvitationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/portalconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	Features     platformcatalogports.FeatureGate
	Checklists   workerchecklistports.EvidenceRefresher
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
	features     platformcatalogports.FeatureGate
	checklists   workerchecklistports.EvidenceRefresher
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
		features:     p.Features,
		checklists:   p.Checklists,
	}
}

// refreshChecklists ticks a PortalAccess checklist item once the change to the
// worker's portal access is committed.
func (s *Service) refreshChecklists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) {
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		if err := s.checklists.RefreshEvidence(committed, tenantInfo, workerID); err != nil {
			s.l.Warn("failed to refresh checklists after a portal access change",
				zap.String("workerId", workerID.String()),
				zap.Error(err))
		}
	})
}
