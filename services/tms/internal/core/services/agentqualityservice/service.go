package agentqualityservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fingerprinter interface {
	Fingerprint(
		definition *agentdefinition.Definition,
		providerID pulid.ID,
		model string,
	) *agent.Fingerprint
}

type regressionNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            ports.DBConnection
	SuiteRuns     repositories.AgentSuiteRunRepository
	Controls      repositories.AgentQualityControlRepository
	Definitions   repositories.AgentDefinitionRepository
	Cases         repositories.AgentEvalCaseRepository
	Evaluations   repositories.AgentEvaluationRepository
	Usage         repositories.AIUsageRepository
	Feedback      repositories.AIFeedbackRepository
	Providers     repositories.AIProviderRepository
	Organizations repositories.OrganizationCacheRepository
	Runtime       *agentruntime.Service
	Audit         services.AuditService
	Completion    services.CompletionService     `optional:"true"`
	Watchtower    services.WatchtowerProjector   `optional:"true"`
	Notifications *notificationservice.Service   `optional:"true"`
	Scheduler     services.AgentQualityScheduler `optional:"true"`
	Starter       services.AgentSuiteStarter     `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            ports.DBConnection
	suiteRuns     repositories.AgentSuiteRunRepository
	controls      repositories.AgentQualityControlRepository
	definitions   repositories.AgentDefinitionRepository
	cases         repositories.AgentEvalCaseRepository
	evaluations   repositories.AgentEvaluationRepository
	usage         repositories.AIUsageRepository
	feedback      repositories.AIFeedbackRepository
	providers     repositories.AIProviderRepository
	organizations repositories.OrganizationCacheRepository
	runtime       fingerprinter
	audit         services.AuditService
	completion    services.CompletionService
	watchtower    services.WatchtowerProjector
	notifier      regressionNotifier
	scheduler     services.AgentQualityScheduler
	starter       services.AgentSuiteStarter
	now           func() int64
}

func New(p Params) *Service {
	var notifier regressionNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}
	var runtime fingerprinter
	if p.Runtime != nil {
		runtime = p.Runtime
	}

	return &Service{
		l:             p.Logger.Named("service.agentquality"),
		db:            p.DB,
		suiteRuns:     p.SuiteRuns,
		controls:      p.Controls,
		definitions:   p.Definitions,
		cases:         p.Cases,
		evaluations:   p.Evaluations,
		usage:         p.Usage,
		feedback:      p.Feedback,
		providers:     p.Providers,
		organizations: p.Organizations,
		runtime:       runtime,
		audit:         p.Audit,
		completion:    p.Completion,
		watchtower:    p.Watchtower,
		notifier:      notifier,
		scheduler:     p.Scheduler,
		starter:       p.Starter,
		now:           timeutils.NowUnix,
	}
}

func AsService(s *Service) services.AgentQualityService { return s }

var _ services.AgentQualityService = (*Service)(nil)

func (s *Service) control(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentquality.Control, error) {
	control, err := s.controls.Get(ctx, tenant)
	if err == nil {
		return control, nil
	}
	if !dberror.IsNotFoundError(err) && !errortypes.IsNotFoundError(err) {
		return nil, fmt.Errorf("read quality controls: %w", err)
	}

	return agentquality.DefaultControl(tenant.OrgID, tenant.BuID), nil
}

func (s *Service) GetControl(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentquality.Control, error) {
	return s.control(ctx, tenant)
}

func (s *Service) UpdateControl(
	ctx context.Context,
	entity *agentquality.Control,
	actor *services.RequestActor,
) (*agentquality.Control, error) {
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	previous, err := s.control(ctx, tenant)
	if err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.controls.Upsert(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logControl(saved, previous, actor)
	if s.scheduler != nil {
		s.scheduler.Sync(ctx, tenant)
	}

	return saved, nil
}

func (s *Service) logControl(
	saved, previous *agentquality.Control,
	actor *services.RequestActor,
) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     saved.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(saved),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
	}, auditservice.WithComment("Agent quality controls updated")); err != nil {
		s.l.Error("failed to log agent quality control audit", zap.Error(err))
	}
}

func (s *Service) timezoneOf(
	ctx context.Context,
	control *agentquality.Control,
	tenant pagination.TenantInfo,
) string {
	organizationTimezone := ""
	if s.organizations != nil {
		organization, err := s.organizations.GetByID(ctx, tenant.OrgID)
		if err != nil {
			s.l.Warn("could not read the organization's timezone; reading windows in UTC",
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(err),
			)
		} else {
			organizationTimezone = organization.Timezone
		}
	}

	return control.EffectiveTimezone(organizationTimezone)
}

func windows(now int64, timezone string) (dayStart, monthStart int64, err error) {
	if dayStart, err = timeutils.DayStartUnix(now, timezone); err != nil {
		return 0, 0, fmt.Errorf("start of day: %w", err)
	}
	if monthStart, err = timeutils.MonthStartUnix(now, timezone); err != nil {
		return 0, 0, fmt.Errorf("start of month: %w", err)
	}

	return dayStart, monthStart, nil
}
