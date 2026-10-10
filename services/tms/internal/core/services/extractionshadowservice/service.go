package extractionshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ extractionshadowports.ExtractionShadowService   = (*Service)(nil)
	_ extractionshadowports.ExtractionShadowSampler   = (*Service)(nil)
	_ extractionshadowports.ExtractionShadowRetention = (*Service)(nil)
)

type contentReader interface {
	GetByDocumentID(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
}

type Params struct {
	fx.In

	Logger    *zap.Logger
	Settings  extractionshadowports.ExtractionShadowSettingsRepository
	Results   extractionshadowports.ExtractionShadowResultRepository
	Scorer    *Scorer
	Contents  documentcontentports.DocumentContentRepository
	Providers agentports.AIProviderRepository
	Retention tenantports.DataRetentionRepository
	Audit     auditports.AuditService
	Predictor extractionshadowports.ExtractionShadowPredictor `optional:"true"`
	Starter   extractionshadowports.ExtractionShadowStarter   `optional:"true"`
}

type Service struct {
	l         *zap.Logger
	settings  extractionshadowports.ExtractionShadowSettingsRepository
	results   extractionshadowports.ExtractionShadowResultRepository
	scorer    *Scorer
	contents  contentReader
	providers agentports.AIProviderRepository
	retention tenantports.DataRetentionRepository
	audit     auditports.AuditService
	predictor extractionshadowports.ExtractionShadowPredictor
	starter   extractionshadowports.ExtractionShadowStarter
	now       func() int64
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	return &Service{
		l:         p.Logger.Named("service.extractionshadow"),
		settings:  p.Settings,
		results:   p.Results,
		scorer:    p.Scorer,
		contents:  p.Contents,
		providers: p.Providers,
		retention: p.Retention,
		audit:     p.Audit,
		predictor: p.Predictor,
		starter:   p.Starter,
		now:       timeutils.NowUnix,
	}
}

func AsService(s *Service) extractionshadowports.ExtractionShadowService { return s }

func AsSampler(s *Service) extractionshadowports.ExtractionShadowSampler { return s }

func AsRetention(s *Service) extractionshadowports.ExtractionShadowRetention { return s }

func tenantOf(result *extractionshadow.ShadowResult) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: result.OrganizationID, BuID: result.BusinessUnitID}
}

func (s *Service) logAction(
	actor *permissionports.RequestActor,
	params *auditports.LogActionParams,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()
	params.UserID = auditActor.UserID
	params.PrincipalType = auditActor.PrincipalType
	params.PrincipalID = auditActor.PrincipalID
	params.APIKeyID = auditActor.APIKeyID

	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log extraction shadow audit", zap.Error(err))
	}
}
