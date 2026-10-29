package aifeedbackservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ aifeedbackports.AIFeedbackService     = (*Service)(nil)
	_ aifeedbackports.AIFeedbackMaintenance = (*Service)(nil)
)

type feedbackStore interface {
	Upsert(ctx context.Context, entity *aifeedback.Feedback) (*aifeedback.Feedback, error)
	Delete(ctx context.Context, req aifeedbackports.DeleteAIFeedbackRequest) (bool, error)
	ListForTargets(
		ctx context.Context,
		req aifeedbackports.ListAIFeedbackForTargetsRequest,
	) ([]*aifeedback.Feedback, error)
	ListByIDs(
		ctx context.Context,
		req aifeedbackports.ListAIFeedbackByIDsRequest,
	) ([]*aifeedback.Feedback, error)
	ListConnection(
		ctx context.Context,
		req *aifeedbackports.ListAIFeedbackConnectionRequest,
	) (*pagination.CursorListResult[*aifeedback.Feedback], error)
	DailySatisfaction(
		ctx context.Context,
		req aifeedbackports.AIFeedbackWindowRequest,
	) ([]*aifeedbackports.AIFeedbackDay, error)
	WorstRated(
		ctx context.Context,
		req aifeedbackports.AIFeedbackWindowRequest,
	) ([]*aifeedbackports.AIFeedbackTargetScore, error)
	ListNegativeSince(
		ctx context.Context,
		req aifeedbackports.ListNegativeAIFeedbackRequest,
	) ([]*aifeedback.Feedback, error)
	PurgeBefore(ctx context.Context, req aifeedbackports.PurgeAIFeedbackRequest) (int64, error)
}

type messageSource interface {
	GetMessageContext(
		ctx context.Context,
		req aifeedbackports.GetAIFeedbackMessageRequest,
	) (*aifeedbackports.AIFeedbackMessageContext, error)
}

type briefingReader interface {
	GetByID(
		ctx context.Context,
		req briefingports.GetBriefingByIDRequest,
	) (*briefing.Briefing, error)
}

type insightReader interface {
	GetByID(ctx context.Context, req insightports.GetInsightByIDRequest) (*insight.Insight, error)
}

type watchtowerReader interface {
	GetByID(
		ctx context.Context,
		req watchtowerports.GetWatchtowerItemRequest,
	) (*watchtower.Item, error)
}

type proposalReader interface {
	GetByID(
		ctx context.Context,
		req agentports.GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

type planReader interface {
	GetByID(
		ctx context.Context,
		req agentports.GetAgentPlanByIDRequest,
	) (*agentplan.AgentPlan, error)
}

type runReader interface {
	GetByID(
		ctx context.Context,
		req agentruntimeports.GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type exceptionReader interface {
	GetByID(
		ctx context.Context,
		req agentports.GetAgentExceptionByIDRequest,
	) (*agentexception.AgentException, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req agentdefinitionports.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type memoryStore interface {
	Create(ctx context.Context, entity *agentmemory.Memory) (*agentmemory.Memory, error)
	ListSuggestionContext(
		ctx context.Context,
		req agentmemoryports.ListAgentMemorySuggestionContextRequest,
	) ([]*agentmemory.Memory, error)
}

type retentionReader interface {
	Get(
		ctx context.Context,
		req tenantports.GetDataRetentionRequest,
	) (*dataretention.DataRetention, error)
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          aifeedbackports.AIFeedbackRepository
	Sources       aifeedbackports.AIFeedbackSourceRepository
	Briefings     briefingports.BriefingRepository
	Insights      insightports.InsightRepository
	Watchtower    watchtowerports.WatchtowerRepository
	Proposals     agentports.AgentProposalRepository
	Plans         agentports.AgentPlanRepository
	Runs          agentruntimeports.AgentRunRepository
	Exceptions    agentports.AgentExceptionRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Memories      agentmemoryports.AgentMemoryRepository
	Retention     tenantports.DataRetentionRepository
	Organizations tenantports.OrganizationRepository
	Permissions   permissionports.PermissionEngine
	Tools         agenttoolports.AgentToolRegistry      `optional:"true"`
	QueryTools    agenttoolports.AgentQueryToolRegistry `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          feedbackStore
	sources       messageSource
	briefings     briefingReader
	insights      insightReader
	watchtower    watchtowerReader
	proposals     proposalReader
	plans         planReader
	runs          runReader
	exceptions    exceptionReader
	definitions   definitionReader
	memories      memoryStore
	retention     retentionReader
	organizations orgzone.OrganizationReader
	permissions   permissionChecker
	redactor      *redactor
	now           func() int64
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.aifeedback"),
		repo:          p.Repo,
		sources:       p.Sources,
		briefings:     p.Briefings,
		insights:      p.Insights,
		watchtower:    p.Watchtower,
		proposals:     p.Proposals,
		plans:         p.Plans,
		runs:          p.Runs,
		exceptions:    p.Exceptions,
		definitions:   p.Definitions,
		memories:      p.Memories,
		retention:     p.Retention,
		organizations: p.Organizations,
		permissions:   p.Permissions,
		redactor:      newRedactor(newToolResources(p.Tools, p.QueryTools)),
		now:           timeutils.NowUnix,
	}
}

func AsService(s *Service) aifeedbackports.AIFeedbackService { return s }

func AsMaintenance(s *Service) aifeedbackports.AIFeedbackMaintenance { return s }

func (s *Service) SetMine(
	ctx context.Context,
	req *aifeedbackports.SetAIFeedbackRequest,
	actor *permissionports.RequestActor,
) (*aifeedback.Feedback, error) {
	if err := requirePerson(actor); err != nil {
		return nil, err
	}

	target := normalizeTarget(req.Target)
	if err := validateTarget(target); err != nil {
		return nil, err
	}

	resolved, err := s.resolve(ctx, resolveParams{
		tenant: req.TenantInfo,
		target: target,
		actor:  actor,
	})
	if err != nil {
		return nil, err
	}

	entity := resolved.feedback(feedbackParams{
		tenant:  req.TenantInfo,
		userID:  actor.UserID,
		target:  target,
		rating:  req.Rating,
		reasons: req.Reasons,
		comment: strings.TrimSpace(req.Comment),
	})

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	return s.repo.Upsert(ctx, entity)
}

func (s *Service) ClearMine(
	ctx context.Context,
	req aifeedbackports.ClearAIFeedbackRequest,
	actor *permissionports.RequestActor,
) (bool, error) {
	if err := requirePerson(actor); err != nil {
		return false, err
	}

	target := normalizeTarget(req.Target)
	if err := validateTarget(target); err != nil {
		return false, err
	}

	return s.repo.Delete(ctx, aifeedbackports.DeleteAIFeedbackRequest{
		TenantInfo: req.TenantInfo,
		UserID:     actor.UserID,
		Target:     target,
	})
}

func (s *Service) ListMine(
	ctx context.Context,
	req aifeedbackports.ListMyAIFeedbackRequest,
	actor *permissionports.RequestActor,
) ([]*aifeedback.Feedback, error) {
	if err := requirePerson(actor); err != nil {
		return nil, err
	}
	if len(req.Targets) == 0 {
		return []*aifeedback.Feedback{}, nil
	}
	if len(req.Targets) > aifeedbackports.MaxAIFeedbackTargetsPerRead {
		return nil, errortypes.NewValidationError(
			"targets",
			errortypes.ErrInvalid,
			"Ask for at most {0} targets at a time",
			aifeedbackports.MaxAIFeedbackTargetsPerRead,
		)
	}

	targets := make([]aifeedbackports.AIFeedbackTargetRef, 0, len(req.Targets))
	for _, target := range req.Targets {
		target = normalizeTarget(target)
		if err := validateTarget(target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return s.repo.ListForTargets(ctx, aifeedbackports.ListAIFeedbackForTargetsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     actor.UserID,
		Targets:    targets,
	})
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *aifeedbackports.ListAIFeedbackConnectionRequest,
) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
	return s.repo.ListConnection(ctx, req)
}

func requirePerson(actor *permissionports.RequestActor) error {
	if !actor.IsUser() || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Only a person can rate what an AI wrote")
	}

	return nil
}

func normalizeTarget(
	target aifeedbackports.AIFeedbackTargetRef,
) aifeedbackports.AIFeedbackTargetRef {
	target.TargetPart = strings.TrimSpace(target.TargetPart)

	return target
}

func validateTarget(target aifeedbackports.AIFeedbackTargetRef) error {
	me := errortypes.NewMultiError()
	if !target.TargetType.IsValid() {
		me.Add("targetType", errortypes.ErrInvalid, "Target type is invalid")
	}
	if target.TargetID.IsNil() {
		me.Add("targetId", errortypes.ErrRequired, "Target is required")
	}
	switch {
	case target.TargetType.RequiresPart() && target.TargetPart == "":
		me.Add("targetPart", errortypes.ErrRequired, "Name the part of the target rated")
	case !target.TargetType.RequiresPart() && target.TargetPart != "":
		me.Add("targetPart", errortypes.ErrInvalid, "This target has no parts to rate")
	}
	if me.HasErrors() {
		return me
	}

	return nil
}
