package aifeedbackservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ services.AIFeedbackService     = (*Service)(nil)
	_ services.AIFeedbackMaintenance = (*Service)(nil)
)

type feedbackStore interface {
	Upsert(ctx context.Context, entity *aifeedback.Feedback) (*aifeedback.Feedback, error)
	Delete(ctx context.Context, req repositories.DeleteAIFeedbackRequest) (bool, error)
	ListForTargets(
		ctx context.Context,
		req repositories.ListAIFeedbackForTargetsRequest,
	) ([]*aifeedback.Feedback, error)
	ListByIDs(
		ctx context.Context,
		req repositories.ListAIFeedbackByIDsRequest,
	) ([]*aifeedback.Feedback, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAIFeedbackConnectionRequest,
	) (*pagination.CursorListResult[*aifeedback.Feedback], error)
	DailySatisfaction(
		ctx context.Context,
		req repositories.AIFeedbackWindowRequest,
	) ([]*repositories.AIFeedbackDay, error)
	WorstRated(
		ctx context.Context,
		req repositories.AIFeedbackWindowRequest,
	) ([]*repositories.AIFeedbackTargetScore, error)
	ListNegativeSince(
		ctx context.Context,
		req repositories.ListNegativeAIFeedbackRequest,
	) ([]*aifeedback.Feedback, error)
	PurgeBefore(ctx context.Context, req repositories.PurgeAIFeedbackRequest) (int64, error)
}

type messageSource interface {
	GetMessageContext(
		ctx context.Context,
		req repositories.GetAIFeedbackMessageRequest,
	) (*repositories.AIFeedbackMessageContext, error)
}

type briefingReader interface {
	GetByID(ctx context.Context, req repositories.GetBriefingByIDRequest) (*briefing.Briefing, error)
}

type insightReader interface {
	GetByID(ctx context.Context, req repositories.GetInsightByIDRequest) (*insight.Insight, error)
}

type watchtowerReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetWatchtowerItemRequest,
	) (*watchtower.Item, error)
}

type proposalReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

type planReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentPlanByIDRequest) (*agent.AgentPlan, error)
}

type runReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error)
}

type exceptionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentExceptionByIDRequest,
	) (*agent.AgentException, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type memoryStore interface {
	Create(ctx context.Context, entity *agent.Memory) (*agent.Memory, error)
	ListSuggestionContext(
		ctx context.Context,
		req repositories.ListAgentMemorySuggestionContextRequest,
	) ([]*agent.Memory, error)
}

type retentionReader interface {
	Get(ctx context.Context, req repositories.GetDataRetentionRequest) (*tenant.DataRetention, error)
}

type organizationReader interface {
	GetByID(ctx context.Context, orgID pulid.ID) (*tenant.Organization, error)
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *services.PermissionCheckRequest,
	) (*services.PermissionCheckResult, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.AIFeedbackRepository
	Sources       repositories.AIFeedbackSourceRepository
	Briefings     repositories.BriefingRepository
	Insights      repositories.InsightRepository
	Watchtower    repositories.WatchtowerRepository
	Proposals     repositories.AgentProposalRepository
	Plans         repositories.AgentPlanRepository
	Runs          repositories.AgentRunRepository
	Exceptions    repositories.AgentExceptionRepository
	Definitions   repositories.AgentDefinitionRepository
	Memories      repositories.AgentMemoryRepository
	Retention     repositories.DataRetentionRepository
	Organizations repositories.OrganizationCacheRepository
	Permissions   services.PermissionEngine
	Tools         services.AgentToolRegistry      `optional:"true"`
	QueryTools    services.AgentQueryToolRegistry `optional:"true"`
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
	organizations organizationReader
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

func AsService(s *Service) services.AIFeedbackService { return s }

func AsMaintenance(s *Service) services.AIFeedbackMaintenance { return s }

func (s *Service) SetMine(
	ctx context.Context,
	req *services.SetAIFeedbackRequest,
	actor *services.RequestActor,
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
	req services.ClearAIFeedbackRequest,
	actor *services.RequestActor,
) (bool, error) {
	if err := requirePerson(actor); err != nil {
		return false, err
	}

	target := normalizeTarget(req.Target)
	if err := validateTarget(target); err != nil {
		return false, err
	}

	return s.repo.Delete(ctx, repositories.DeleteAIFeedbackRequest{
		TenantInfo: req.TenantInfo,
		UserID:     actor.UserID,
		Target:     target,
	})
}

func (s *Service) ListMine(
	ctx context.Context,
	req services.ListMyAIFeedbackRequest,
	actor *services.RequestActor,
) ([]*aifeedback.Feedback, error) {
	if err := requirePerson(actor); err != nil {
		return nil, err
	}
	if len(req.Targets) == 0 {
		return []*aifeedback.Feedback{}, nil
	}
	if len(req.Targets) > repositories.MaxAIFeedbackTargetsPerRead {
		return nil, errortypes.NewValidationError(
			"targets",
			errortypes.ErrInvalid,
			"Ask for at most {0} targets at a time",
			repositories.MaxAIFeedbackTargetsPerRead,
		)
	}

	targets := make([]repositories.AIFeedbackTargetRef, 0, len(req.Targets))
	for _, target := range req.Targets {
		target = normalizeTarget(target)
		if err := validateTarget(target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return s.repo.ListForTargets(ctx, repositories.ListAIFeedbackForTargetsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     actor.UserID,
		Targets:    targets,
	})
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAIFeedbackConnectionRequest,
) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
	return s.repo.ListConnection(ctx, req)
}

func requirePerson(actor *services.RequestActor) error {
	if !actor.IsUser() || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Only a person can rate what an AI wrote")
	}

	return nil
}

func normalizeTarget(target repositories.AIFeedbackTargetRef) repositories.AIFeedbackTargetRef {
	target.TargetPart = strings.TrimSpace(target.TargetPart)

	return target
}

func validateTarget(target repositories.AIFeedbackTargetRef) error {
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
