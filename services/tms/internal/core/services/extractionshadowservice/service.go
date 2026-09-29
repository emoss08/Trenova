package extractionshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ services.ExtractionShadowService   = (*Service)(nil)
	_ services.ExtractionShadowSampler   = (*Service)(nil)
	_ services.ExtractionShadowRetention = (*Service)(nil)
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
	Settings  repositories.ExtractionShadowSettingsRepository
	Results   repositories.ExtractionShadowResultRepository
	Scorer    *Scorer
	Contents  repositories.DocumentContentRepository
	Providers repositories.AIProviderRepository
	Retention repositories.DataRetentionRepository
	Audit     services.AuditService
	Predictor services.ExtractionShadowPredictor `optional:"true"`
	Starter   services.ExtractionShadowStarter   `optional:"true"`
}

type Service struct {
	l         *zap.Logger
	settings  repositories.ExtractionShadowSettingsRepository
	results   repositories.ExtractionShadowResultRepository
	scorer    *Scorer
	contents  contentReader
	providers repositories.AIProviderRepository
	retention repositories.DataRetentionRepository
	audit     services.AuditService
	predictor services.ExtractionShadowPredictor
	starter   services.ExtractionShadowStarter
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

func AsService(s *Service) services.ExtractionShadowService { return s }

func AsSampler(s *Service) services.ExtractionShadowSampler { return s }

func AsRetention(s *Service) services.ExtractionShadowRetention { return s }

func tenantOf(result *extractionshadow.ShadowResult) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: result.OrganizationID, BuID: result.BusinessUnitID}
}

func (s *Service) logAction(
	actor *services.RequestActor,
	params *services.LogActionParams,
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
