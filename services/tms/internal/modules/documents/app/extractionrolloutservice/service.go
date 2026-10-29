package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ extractionrolloutports.ExtractionRolloutService   = (*Service)(nil)
	_ extractionrolloutports.ExtractionRolloutRouter    = (*Service)(nil)
	_ extractionrolloutports.ExtractionRolloutGuard     = (*Service)(nil)
	_ extractionrolloutports.ExtractionRolloutRetention = (*Service)(nil)
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
	Rollouts      extractionrolloutports.ExtractionRolloutRepository
	Assignments   extractionrolloutports.RolloutAssignmentRepository
	Corrections   aicorrectionports.AICorrectionRepository
	Providers     aiproviderports.AIProviderRepository
	Retention     tenantports.DataRetentionRepository
	Audit         auditports.AuditService
	Notifications *notificationservice.Service `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	rollouts    extractionrolloutports.ExtractionRolloutRepository
	assignments extractionrolloutports.RolloutAssignmentRepository
	corrections aicorrectionports.AICorrectionRepository
	providers   aiproviderports.AIProviderRepository
	retention   tenantports.DataRetentionRepository
	audit       auditports.AuditService
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

func AsService(s *Service) extractionrolloutports.ExtractionRolloutService { return s }

func AsRouter(s *Service) extractionrolloutports.ExtractionRolloutRouter { return s }

func AsGuard(s *Service) extractionrolloutports.ExtractionRolloutGuard { return s }

func AsRetention(s *Service) extractionrolloutports.ExtractionRolloutRetention { return s }

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
