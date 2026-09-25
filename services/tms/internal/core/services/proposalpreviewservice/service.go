// Package proposalpreviewservice says what a proposed write would do, for
// the person reading it, from the world as it is now.
//
// A tool previews its own write (ToolPreviewer); this service runs that
// preview inside a read-only snapshot with a time limit, names the records
// it touches, compares the record with the one the proposal pinned and with
// the baseline taken when it was filed, withholds what the reader may not
// see, and digests what is left. The digest is what an approval sends back:
// a person only ever approves the preview they were shown.
package proposalpreviewservice

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// ReadTimeout bounds one tool's preview for a person reading it or a
	// decision checking it.
	ReadTimeout = 3 * time.Second
	// BaselineTimeout bounds the preview taken when a proposal is filed.
	BaselineTimeout = 5 * time.Second
)

// ErrTenantMismatch is a preview asked for by someone outside the
// proposal's tenant.
var ErrTenantMismatch = errors.New("proposal does not belong to the reader's organization")

type modificationChecker interface {
	CheckModifications(
		ctx context.Context,
		proposal *agent.AgentProposal,
		modifications map[string]any,
		actor *services.RequestActor,
	) (map[string]any, error)
}

type decisionReader interface {
	ListByProposals(
		ctx context.Context,
		req repositories.ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
}

type stepReader interface {
	ListByPlan(
		ctx context.Context,
		req repositories.ListAgentProposalsByPlanRequest,
	) ([]*agent.AgentProposal, error)
}

type baselineStore interface {
	Create(ctx context.Context, baseline *agent.ProposalBaseline) error
	ListByProposals(
		ctx context.Context,
		req repositories.ListProposalBaselinesRequest,
	) ([]*agent.ProposalBaseline, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	DB          ports.DBConnection
	Tools       services.AgentToolRegistry
	Versions    services.RecordVersionReader `optional:"true"`
	Labeler     services.RecordLabeler       `optional:"true"`
	Baselines   repositories.AgentProposalBaselineRepository
	Decisions   repositories.AgentDecisionRepository
	Proposals   repositories.AgentProposalRepository
	Executor    *proposalexecutor.Service
	Permissions services.PermissionEngine
	Metrics     *metrics.Registry `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	db          ports.DBConnection
	tools       services.AgentToolRegistry
	versions    services.RecordVersionReader
	labeler     services.RecordLabeler
	baselines   baselineStore
	decisions   decisionReader
	steps       stepReader
	checker     modificationChecker
	permissions services.PermissionEngine
	metrics     *metrics.ProposalPreview
	now         func() int64
	readTimeout time.Duration
	fileTimeout time.Duration
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) services.ProposalPreviewService {
	svc := &Service{
		l:           p.Logger.Named("service.proposal-preview"),
		db:          p.DB,
		tools:       p.Tools,
		versions:    p.Versions,
		labeler:     p.Labeler,
		permissions: p.Permissions,
		now:         timeutils.NowUnix,
		readTimeout: ReadTimeout,
		fileTimeout: BaselineTimeout,
	}
	if p.Baselines != nil {
		svc.baselines = p.Baselines
	}
	if p.Decisions != nil {
		svc.decisions = p.Decisions
	}
	if p.Proposals != nil {
		svc.steps = p.Proposals
	}
	if p.Executor != nil {
		svc.checker = p.Executor
	}
	if p.Metrics != nil {
		svc.metrics = p.Metrics.ProposalPreview
	}

	return svc
}

func (s *Service) ForProposal(
	ctx context.Context,
	req *services.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	if req == nil || req.Proposal == nil {
		return nil, errortypes.NewNotFoundError("Proposal not found")
	}
	proposal := req.Proposal
	actor := viewerActor(req.Viewer)
	if err := assertTenant(proposal.OrganizationID, proposal.BusinessUnitID, actor); err != nil {
		return nil, err
	}

	purpose := aitrace.PreviewPurposeRead
	if req.Deciding {
		purpose = aitrace.PreviewPurposeDecide
	}
	ctx, span := aitrace.StartPreview(ctx, &aitrace.PreviewSpec{
		ToolName:       proposal.ToolName,
		ProposalID:     proposal.ID,
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Purpose:        purpose,
	})
	defer span.End()
	started := time.Now()

	preview, err := s.forProposal(ctx, req, actor)
	if err != nil {
		aitrace.MarkFailed(span, aitrace.OutcomeFailed)

		return nil, err
	}

	s.observe(span, proposal.ToolName, preview, started)

	return preview, nil
}

func (s *Service) forProposal(
	ctx context.Context,
	req *services.ProposalPreviewRequest,
	actor *services.RequestActor,
) (*agent.ProposalPreview, error) {
	proposal := req.Proposal
	reader := s.accessFor(req.Viewer)
	if proposal.Status != agent.ProposalStatusPending {
		return s.recordedFor(ctx, proposal, reader)
	}

	d, err := s.draft(ctx, &draftInput{
		proposal:      proposal,
		modifications: req.Modifications,
		params:        req.Params,
		actor:         actor,
		timeout:       s.readTimeout,
	})
	if err != nil {
		return nil, err
	}

	s.markChangedSinceProposed(ctx, tenantOf(proposal), []*draft{d})

	return s.finish(ctx, d, reader)
}

func (s *Service) ForPlan(
	ctx context.Context,
	req *services.PlanPreviewRequest,
) (*agent.PlanPreview, error) {
	if req == nil || req.Plan == nil {
		return nil, errortypes.NewNotFoundError("Agent plan not found")
	}
	plan := req.Plan
	actor := viewerActor(req.Viewer)
	if err := assertTenant(plan.OrganizationID, plan.BusinessUnitID, actor); err != nil {
		return nil, err
	}

	steps, err := s.planSteps(ctx, plan, req.Steps)
	if err != nil {
		return nil, err
	}

	reader := s.accessFor(req.Viewer)
	preview := &agent.PlanPreview{
		PlanID:     plan.ID,
		Steps:      make([]agent.PlanStepPreview, 0, len(steps)),
		ComputedAt: s.now(),
	}

	pending := make([]*draft, 0, len(steps))
	for _, step := range steps {
		if step.Status != agent.ProposalStatusPending {
			continue
		}
		d, draftErr := s.draftStep(ctx, step, actor)
		if draftErr != nil {
			return nil, draftErr
		}
		pending = append(pending, d)
	}
	s.markChangedSinceProposed(ctx, tenantOf(plan), pending)
	s.chain(pending)

	digests := make([]string, 0, len(pending))
	byProposal := make(map[pulid.ID]*agent.ProposalPreview, len(pending))
	for _, d := range pending {
		finished, finishErr := s.finish(ctx, d, reader)
		if finishErr != nil {
			return nil, finishErr
		}
		byProposal[d.proposal.ID] = finished
		digests = append(digests, finished.Digest)
		preview.Stale = preview.Stale || finished.IsStale()
	}

	for _, step := range steps {
		stepPreview, ok := byProposal[step.ID]
		if !ok {
			stepPreview, err = s.recordedFor(ctx, step, reader)
			if err != nil {
				return nil, err
			}
		}
		preview.WithheldCount += stepPreview.WithheldCount
		preview.Steps = append(preview.Steps, agent.PlanStepPreview{
			ProposalID: step.ID,
			Step:       step.PlanStep,
			Preview:    stepPreview,
		})
	}

	if len(pending) > 0 {
		preview.Digest, err = agent.PlanPreviewDigest(plan.ID, digests)
		if err != nil {
			return nil, err
		}
	}

	return preview, nil
}

func (s *Service) draftStep(
	ctx context.Context,
	step *agent.AgentProposal,
	actor *services.RequestActor,
) (*draft, error) {
	ctx, span := aitrace.StartPreview(ctx, &aitrace.PreviewSpec{
		ToolName:       step.ToolName,
		ProposalID:     step.ID,
		OrganizationID: step.OrganizationID,
		BusinessUnitID: step.BusinessUnitID,
		Purpose:        aitrace.PreviewPurposeRead,
	})
	defer span.End()
	started := time.Now()

	d, err := s.draft(ctx, &draftInput{proposal: step, actor: actor, timeout: s.readTimeout})
	if err != nil {
		aitrace.MarkFailed(span, aitrace.OutcomeFailed)

		return nil, err
	}
	s.observe(span, step.ToolName, d.preview, started)

	return d, nil
}

func (s *Service) planSteps(
	ctx context.Context,
	plan *agent.AgentPlan,
	given []*agent.AgentProposal,
) ([]*agent.AgentProposal, error) {
	steps := given
	if steps == nil {
		if s.steps == nil {
			return nil, errortypes.NewBusinessError("The plan's steps cannot be read")
		}
		var err error
		steps, err = s.steps.ListByPlan(ctx, repositories.ListAgentProposalsByPlanRequest{
			PlanID:     plan.ID,
			TenantInfo: tenantOf(plan),
		})
		if err != nil {
			return nil, err
		}
	}

	ordered := make([]*agent.AgentProposal, 0, len(steps))
	for _, step := range steps {
		if step == nil || step.PlanID == nil || *step.PlanID != plan.ID ||
			step.OrganizationID != plan.OrganizationID ||
			step.BusinessUnitID != plan.BusinessUnitID {
			continue
		}
		ordered = append(ordered, step)
	}
	slices.SortStableFunc(ordered, func(a, b *agent.AgentProposal) int {
		return a.PlanStep - b.PlanStep
	})

	return ordered, nil
}

func (s *Service) ForDecision(
	ctx context.Context,
	decision *agent.AgentDecision,
	viewer *services.PreviewViewer,
) *agent.ProposalPreview {
	if decision == nil || decision.Preview == nil {
		return nil
	}
	actor := viewerActor(viewer)
	if assertTenant(decision.OrganizationID, decision.BusinessUnitID, actor) != nil {
		return nil
	}

	return s.recorded(ctx, decision, s.accessFor(viewer))
}

func (s *Service) observe(
	span trace.Span,
	tool string,
	preview *agent.ProposalPreview,
	started time.Time,
) {
	if preview == nil {
		return
	}
	aitrace.FinishPreview(span, &aitrace.PreviewResult{
		Coverage: string(preview.Coverage),
		Stale:    preview.IsStale(),
		Records:  len(preview.Changes),
		Withheld: preview.WithheldCount,
		Recorded: preview.Recorded,
	})
	s.metrics.RecordPreview(tool, string(preview.Coverage), time.Since(started).Seconds())
}

func viewerActor(viewer *services.PreviewViewer) *services.RequestActor {
	if viewer == nil {
		return nil
	}

	return viewer.Actor
}

func assertTenant(orgID, buID pulid.ID, actor *services.RequestActor) error {
	if actor == nil || actor.OrganizationID != orgID || actor.BusinessUnitID != buID {
		return ErrTenantMismatch
	}

	return nil
}

type tenanted interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

func tenantOf(record tenanted) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: record.GetOrganizationID(),
		BuID:  record.GetBusinessUnitID(),
	}
}
