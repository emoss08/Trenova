package agentqualityservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	) *agentfingerprint.Fingerprint
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
	DB            kernelports.Transactor
	SuiteRuns     agentqualityports.AgentSuiteRunRepository
	Controls      agentqualityports.AgentQualityControlRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Cases         agentqualityports.AgentEvalCaseRepository
	Evaluations   agentports.AgentEvaluationRepository
	Usage         aiusageports.AIUsageRepository
	Feedback      aifeedbackports.AIFeedbackRepository
	Providers     aiproviderports.AIProviderRepository
	Organizations tenantports.OrganizationRepository
	Runtime       *agentruntime.Service
	Audit         auditports.AuditService
	Completion    llmports.CompletionService              `optional:"true"`
	Batches       llmports.BatchCompletionService         `optional:"true"`
	Watchtower    watchtowerports.WatchtowerProjector     `optional:"true"`
	Notifications *notificationservice.Service            `optional:"true"`
	Scheduler     agentqualityports.AgentQualityScheduler `optional:"true"`
	Starter       agentqualityports.AgentSuiteStarter     `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	suiteRuns     agentqualityports.AgentSuiteRunRepository
	controls      agentqualityports.AgentQualityControlRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	cases         agentqualityports.AgentEvalCaseRepository
	evaluations   agentports.AgentEvaluationRepository
	usage         aiusageports.AIUsageRepository
	feedback      aifeedbackports.AIFeedbackRepository
	providers     aiproviderports.AIProviderRepository
	organizations tenantports.OrganizationRepository
	runtime       fingerprinter
	audit         auditports.AuditService
	completion    llmports.CompletionService
	batches       llmports.BatchCompletionService
	watchtower    watchtowerports.WatchtowerProjector
	notifier      regressionNotifier
	scheduler     agentqualityports.AgentQualityScheduler
	starter       agentqualityports.AgentSuiteStarter
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
		batches:       p.Batches,
		watchtower:    p.Watchtower,
		notifier:      notifier,
		scheduler:     p.Scheduler,
		starter:       p.Starter,
		now:           timeutils.NowUnix,
	}
}

func AsService(s *Service) agentqualityports.AgentQualityService { return s }

func AsEvaluationBudget(s *Service) agentqualityports.EvaluationBudget { return s }

var _ agentqualityports.AgentQualityService = (*Service)(nil)

func (s *Service) control(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentqualitycontrol.Control, error) {
	control, err := s.controls.Get(ctx, tenant)
	if err == nil {
		return control, nil
	}
	if !dberror.IsNotFoundError(err) && !errortypes.IsNotFoundError(err) {
		return nil, fmt.Errorf("read quality controls: %w", err)
	}

	return agentqualitycontrol.DefaultControl(tenant.OrgID, tenant.BuID), nil
}

func (s *Service) GetControl(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentqualitycontrol.Control, error) {
	return s.control(ctx, tenant)
}

func (s *Service) UpdateControl(
	ctx context.Context,
	entity *agentqualitycontrol.Control,
	actor *permissionports.RequestActor,
) (*agentqualitycontrol.Control, error) {
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
	saved, previous *agentqualitycontrol.Control,
	actor *permissionports.RequestActor,
) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&auditports.LogActionParams{
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
	control *agentqualitycontrol.Control,
	tenant pagination.TenantInfo,
) string {
	organizationTimezone := ""
	if s.organizations != nil {
		organization, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
			TenantInfo: tenant,
		})
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
