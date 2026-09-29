package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ services.ExtractionRolloutService   = (*Service)(nil)
	_ services.ExtractionRolloutRouter    = (*Service)(nil)
	_ services.ExtractionRolloutGuard     = (*Service)(nil)
	_ services.ExtractionRolloutRetention = (*Service)(nil)
)

type haltNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Rollouts      repositories.ExtractionRolloutRepository
	Assignments   repositories.RolloutAssignmentRepository
	Corrections   repositories.AICorrectionRepository
	Providers     repositories.AIProviderRepository
	Retention     repositories.DataRetentionRepository
	Audit         services.AuditService
	Notifications *notificationservice.Service `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	rollouts    repositories.ExtractionRolloutRepository
	assignments repositories.RolloutAssignmentRepository
	corrections repositories.AICorrectionRepository
	providers   repositories.AIProviderRepository
	retention   repositories.DataRetentionRepository
	audit       services.AuditService
	notifier    haltNotifier
	now         func() int64
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	var notifier haltNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}

	return &Service{
		l:           p.Logger.Named("service.extractionrollout"),
		rollouts:    p.Rollouts,
		assignments: p.Assignments,
		corrections: p.Corrections,
		providers:   p.Providers,
		retention:   p.Retention,
		audit:       p.Audit,
		notifier:    notifier,
		now:         timeutils.NowUnix,
	}
}

func AsService(s *Service) services.ExtractionRolloutService { return s }

func AsRouter(s *Service) services.ExtractionRolloutRouter { return s }

func AsGuard(s *Service) services.ExtractionRolloutGuard { return s }

func AsRetention(s *Service) services.ExtractionRolloutRetention { return s }

func (s *Service) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionrollout.ExtractionRollout, error) {
	rollout, err := s.rollouts.Get(ctx, tenant)
	switch {
	case err == nil:
		return rollout, nil
	case errortypes.IsNotFoundError(err):
		return extractionrollout.Default(tenant.OrgID, tenant.BuID), nil
	default:
		return nil, err
	}
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
		s.l.Error("failed to log extraction rollout audit", zap.Error(err))
	}
}

func auditable(rollout *extractionrollout.ExtractionRollout) map[string]any {
	return map[string]any{
		"id":                         rollout.ID,
		"enabled":                    rollout.Enabled,
		"providerId":                 rollout.ProviderID,
		"percent":                    rollout.Percent,
		"maxAccuracyDropPoints":      rollout.MaxAccuracyDropPoints,
		"maxRejectionIncreasePoints": rollout.MaxRejectionIncreasePoints,
		"startedAt":                  rollout.StartedAt,
		"haltedAt":                   rollout.HaltedAt,
		"haltReason":                 rollout.HaltReason,
		"haltCandidateRate":          rollout.HaltCandidateRate,
		"haltBaselineRate":           rollout.HaltBaselineRate,
	}
}
