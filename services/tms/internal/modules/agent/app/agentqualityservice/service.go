package agentqualityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualitysettings"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type controlSettings interface {
	Control(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*agentqualitycontrol.Control, error)
	Timezone(
		ctx context.Context,
		control *agentqualitycontrol.Control,
		tenant pagination.TenantInfo,
	) string
}

type suiteOpener interface {
	PlanSweep(
		ctx context.Context,
		req *agentqualityengine.SweepRequest,
	) (*agentqualityengine.SweepPlan, error)
	OpenSuite(
		ctx context.Context,
		req *agentqualityengine.OpenSuiteRequest,
	) (*agentqualityengine.OpenedSuite, error)
	FailSuite(ctx context.Context, req *agentqualityengine.FailSuiteRequest) error
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	SuiteRuns   agentqualityports.AgentSuiteRunRepository
	Controls    agentqualityports.AgentQualityControlRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
	Cases       agentqualityports.AgentEvalCaseRepository
	Usage       aiusageports.AIUsageRepository
	Feedback    aifeedbackports.AIFeedbackRepository
	Audit       auditports.AuditService
	Settings    *agentqualitysettings.Reader
	Engine      *agentqualityengine.Engine
	Scheduler   agentqualityports.AgentQualityScheduler `optional:"true"`
	Starter     agentqualityports.AgentSuiteStarter     `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	suiteRuns   agentqualityports.AgentSuiteRunRepository
	controls    agentqualityports.AgentQualityControlRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	cases       agentqualityports.AgentEvalCaseRepository
	usage       aiusageports.AIUsageRepository
	feedback    aifeedbackports.AIFeedbackRepository
	audit       auditports.AuditService
	settings    controlSettings
	suites      suiteOpener
	scheduler   agentqualityports.AgentQualityScheduler
	starter     agentqualityports.AgentSuiteStarter
	now         func() int64
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.agentquality"),
		suiteRuns:   p.SuiteRuns,
		controls:    p.Controls,
		definitions: p.Definitions,
		cases:       p.Cases,
		usage:       p.Usage,
		feedback:    p.Feedback,
		audit:       p.Audit,
		settings:    p.Settings,
		suites:      p.Engine,
		scheduler:   p.Scheduler,
		starter:     p.Starter,
		now:         timeutils.NowUnix,
	}
}

func AsService(s *Service) agentqualityports.AgentQualityService { return s }

var _ agentqualityports.AgentQualityService = (*Service)(nil)

func (s *Service) GetControl(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentqualitycontrol.Control, error) {
	return s.settings.Control(ctx, tenant)
}

func (s *Service) UpdateControl(
	ctx context.Context,
	entity *agentqualitycontrol.Control,
	actor *permissionports.RequestActor,
) (*agentqualitycontrol.Control, error) {
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	previous, err := s.settings.Control(ctx, tenant)
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
