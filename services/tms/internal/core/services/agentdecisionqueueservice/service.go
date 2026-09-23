package agentdecisionqueueservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
	// MaxBatch bounds one batch decision. Each proposal is decided and, when
	// approved, executed in turn, so the bound is what keeps one request
	// from running for minutes.
	MaxBatch = 50
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Queue     repositories.AgentDecisionQueueRepository
	Proposals repositories.AgentProposalRepository
	Plans     repositories.AgentPlanRepository
	Decisions services.AgentDecisionService
	Shadow    *agentshadow.Resolver
}

// Service reads what is waiting on a person as one queue and decides
// several proposals at once. Each decision still goes through the decision
// service one at a time: the authorization, the execution, the ledger and
// the audit line are the same whether one proposal was approved or twenty.
type Service struct {
	l         *zap.Logger
	queue     repositories.AgentDecisionQueueRepository
	proposals repositories.AgentProposalRepository
	plans     repositories.AgentPlanRepository
	decisions services.AgentDecisionService
	shadow    shadowReader
}

type shadowReader interface {
	Organization(ctx context.Context, tenant pagination.TenantInfo) (bool, error)
}

func New(p Params) services.AgentDecisionQueueService {
	return &Service{
		l:         p.Logger.Named("service.agentdecisionqueue"),
		queue:     p.Queue,
		proposals: p.Proposals,
		plans:     p.Plans,
		decisions: p.Decisions,
		shadow:    p.Shadow,
	}
}

func (s *Service) ListPending(
	ctx context.Context,
	req services.ListPendingDecisionsRequest,
) (*services.PendingDecisionsPage, error) {
	shadow, err := s.shadow.Organization(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if shadow || withheld(req.Usable) {
		return &services.PendingDecisionsPage{Items: []services.PendingDecision{}}, nil
	}

	limit := req.First
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	listReq := repositories.ListPendingDecisionsRequest{
		TenantInfo:               req.TenantInfo,
		Limit:                    limit,
		AgentDefinitionID:        req.AgentDefinitionID,
		ToolName:                 strings.TrimSpace(req.ToolName),
		ExcludeShadowDefinitions: true,
		Audience:                 audienceOf(req.Usable),
		Now:                      timeutils.NowUnix(),
	}
	if req.After != "" {
		after, decodeErr := decodeCursor(req.After)
		if decodeErr != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalid,
				"Cursor is invalid",
			)
		}
		listReq.After = &after
	}

	page, err := s.queue.ListPending(ctx, listReq)
	if err != nil {
		return nil, err
	}

	items, err := s.loadEntries(ctx, req.TenantInfo, page.Entries)
	if err != nil {
		return nil, err
	}

	out := &services.PendingDecisionsPage{Items: items, HasNextPage: page.HasNextPage}
	if req.IncludeTotalCount {
		total, countErr := s.queue.CountPending(ctx, listReq)
		if countErr != nil {
			return nil, countErr
		}
		out.TotalCount = &total
	}

	return out, nil
}

// loadEntries turns queue entries into their records, in queue order. An
// entry whose record has gone (decided between the two reads) is dropped
// rather than failing the page.
func (s *Service) loadEntries(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entries []repositories.PendingDecisionEntry,
) ([]services.PendingDecision, error) {
	proposalIDs := make([]pulid.ID, 0, len(entries))
	planIDs := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		switch entry.Kind {
		case repositories.PendingDecisionProposal:
			proposalIDs = append(proposalIDs, entry.ID)
		case repositories.PendingDecisionPlan:
			planIDs = append(planIDs, entry.ID)
		}
	}

	proposalsByID := make(map[pulid.ID]*agent.AgentProposal, len(proposalIDs))
	if len(proposalIDs) > 0 {
		proposals, err := s.proposals.ListByIDs(ctx, repositories.ListAgentProposalsByIDsRequest{
			IDs:        proposalIDs,
			TenantInfo: tenant,
		})
		if err != nil {
			return nil, err
		}
		for _, proposal := range proposals {
			proposalsByID[proposal.ID] = proposal
		}
	}

	plansByID := make(map[pulid.ID]*agent.AgentPlan, len(planIDs))
	if len(planIDs) > 0 {
		plans, err := s.plans.ListByIDs(ctx, repositories.ListAgentPlansByIDsRequest{
			IDs:        planIDs,
			TenantInfo: tenant,
		})
		if err != nil {
			return nil, err
		}
		for _, plan := range plans {
			plansByID[plan.ID] = plan
		}
	}

	items := make([]services.PendingDecision, 0, len(entries))
	for _, entry := range entries {
		cursor, err := encodeCursor(entry)
		if err != nil {
			return nil, err
		}
		item := services.PendingDecision{CreatedAt: entry.CreatedAt, Cursor: cursor}
		switch entry.Kind {
		case repositories.PendingDecisionProposal:
			proposal, ok := proposalsByID[entry.ID]
			if !ok {
				continue
			}
			item.Proposal = proposal
		case repositories.PendingDecisionPlan:
			plan, ok := plansByID[entry.ID]
			if !ok {
				continue
			}
			item.Plan = plan
		default:
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *Service) Count(
	ctx context.Context,
	tenant pagination.TenantInfo,
	usable *services.UsableAgents,
) (int, error) {
	shadow, err := s.shadow.Organization(ctx, tenant)
	if err != nil {
		return 0, err
	}
	if shadow || withheld(usable) {
		return 0, nil
	}

	return s.queue.CountPending(ctx, repositories.ListPendingDecisionsRequest{
		TenantInfo:               tenant,
		ExcludeShadowDefinitions: true,
		Audience:                 audienceOf(usable),
		Now:                      timeutils.NowUnix(),
	})
}

// withheld reports a reader who may use no agent at all, so nothing any
// agent raised is theirs to see.
func withheld(usable *services.UsableAgents) bool {
	return usable != nil && !usable.Assistant
}

func audienceOf(usable *services.UsableAgents) *repositories.AgentAudience {
	if usable == nil {
		return nil
	}

	return usable.Audience()
}

func (s *Service) Summary(
	ctx context.Context,
	tenant pagination.TenantInfo,
	usable *services.UsableAgents,
) (*repositories.PendingDecisionSummary, error) {
	shadow, err := s.shadow.Organization(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if shadow || withheld(usable) {
		return &repositories.PendingDecisionSummary{
			ByAgent: []repositories.PendingDecisionAgentCount{},
			ByTool:  []repositories.PendingDecisionToolCount{},
		}, nil
	}

	return s.queue.Summary(ctx, repositories.PendingDecisionSummaryRequest{
		TenantInfo:               tenant,
		ExcludeShadowDefinitions: true,
		Audience:                 audienceOf(usable),
		Now:                      timeutils.NowUnix(),
	})
}

// DecideMany decides several proposals of one tool the same way. It refuses
// a mixed batch before touching anything: an approver reads the tool once
// and approves what they read, and "approve these twelve" must not carry a
// thirteenth of another kind. Once it starts, every proposal is decided in
// turn and the outcome of each is reported, so one failure neither stops
// the rest nor hides.
func (s *Service) DecideMany(
	ctx context.Context,
	req *services.DecideAgentProposalsRequest,
	actor *services.RequestActor,
) ([]services.AgentProposalDecisionResult, error) {
	if err := validateBatch(req); err != nil {
		return nil, err
	}

	proposals, err := s.proposals.ListByIDs(ctx, repositories.ListAgentProposalsByIDsRequest{
		IDs:        req.ProposalIDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*agent.AgentProposal, len(proposals))
	for _, proposal := range proposals {
		byID[proposal.ID] = proposal
	}

	multiErr := errortypes.NewMultiError()
	tools := make(map[string]struct{}, 1)
	for i, id := range req.ProposalIDs {
		field := fmt.Sprintf("proposalIds[%d]", i)
		proposal, ok := byID[id]
		switch {
		case !ok:
			multiErr.Add(field, errortypes.ErrNotFound, "Proposal not found")
		case proposal.PlanID != nil:
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"This proposal is a step of a plan; decide the plan",
			)
		default:
			tools[proposal.ToolName] = struct{}{}
		}
	}
	if len(tools) > 1 {
		names := make([]string, 0, len(tools))
		for name := range tools {
			names = append(names, name)
		}
		sort.Strings(names)
		multiErr.Add("proposalIds", errortypes.ErrInvalid,
			"A batch decides one tool at a time; these span "+strings.Join(names, ", "))
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	results := make([]services.AgentProposalDecisionResult, 0, len(req.ProposalIDs))
	for _, id := range req.ProposalIDs {
		result := services.AgentProposalDecisionResult{ProposalID: id}
		outcome, decideErr := s.decisions.DecideWithOutcome(
			ctx,
			&services.DecideAgentProposalRequest{
				ProposalID: id,
				Decision:   req.Decision,
				ReasonCode: req.ReasonCode,
				TenantInfo: req.TenantInfo,
			},
			actor,
		)
		switch {
		case decideErr != nil:
			result.Error = decisionErrorMessage(decideErr)
			s.l.Warn("batch decision failed for a proposal",
				zap.String("proposal", id.String()),
				zap.Error(decideErr),
			)
		default:
			result.Decision = outcome.Decision
			if outcome.ExecutionError != nil {
				result.Error = outcome.ExecutionError.Error()
			} else {
				result.Executed = req.Decision == agent.DecisionAccepted
			}
		}
		results = append(results, result)
	}

	return results, nil
}

func validateBatch(req *services.DecideAgentProposalsRequest) error {
	multiErr := errortypes.NewMultiError()
	switch {
	case len(req.ProposalIDs) == 0:
		multiErr.Add("proposalIds", errortypes.ErrRequired, "Choose at least one proposal")
	case len(req.ProposalIDs) > MaxBatch:
		multiErr.Add("proposalIds", errortypes.ErrInvalid,
			fmt.Sprintf("Decide at most %d proposals at once", MaxBatch))
	}
	seen := make(map[pulid.ID]struct{}, len(req.ProposalIDs))
	for i, id := range req.ProposalIDs {
		if _, dup := seen[id]; dup {
			multiErr.Add(
				fmt.Sprintf("proposalIds[%d]", i),
				errortypes.ErrInvalid,
				"Proposal listed twice",
			)
		}
		seen[id] = struct{}{}
	}
	switch req.Decision {
	case agent.DecisionAccepted, agent.DecisionRejected:
	case agent.DecisionModified:
		multiErr.Add("decision", errortypes.ErrInvalid,
			"A change applies to one proposal; approve with changes from its card")
	default:
		multiErr.Add("decision", errortypes.ErrInvalid, "Decision is invalid")
	}
	if strings.TrimSpace(req.ReasonCode) == "" && req.Decision == agent.DecisionRejected {
		multiErr.Add("reasonCode", errortypes.ErrRequired, "Say why these are rejected")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// decisionErrorMessage keeps what a person may read of a failure. A
// business or validation error was written for them; anything else is an
// internal fault whose wording belongs in the log.
func decisionErrorMessage(err error) string {
	if errortypes.IsBusinessError(err) || errortypes.IsMultiError(err) ||
		errortypes.IsNotFoundError(err) || errortypes.IsError(err) ||
		errortypes.IsAuthorizationError(err) || errortypes.IsVersionMismatchError(err) {
		return err.Error()
	}

	return "This proposal could not be decided. Try it on its own."
}

func encodeCursor(entry repositories.PendingDecisionEntry) (string, error) {
	return pagination.EncodeCursor(pagination.Cursor{CreatedAt: entry.CreatedAt, ID: entry.ID})
}

func decodeCursor(encoded string) (repositories.PendingDecisionCursor, error) {
	cursor, err := pagination.DecodeCursor(encoded)
	if err != nil {
		return repositories.PendingDecisionCursor{}, err
	}

	return repositories.PendingDecisionCursor{CreatedAt: cursor.CreatedAt, ID: cursor.ID}, nil
}
