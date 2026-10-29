package agentdecisionqueueservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionqueueports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	Queue     agentdecisionqueueports.AgentDecisionQueueRepository
	Proposals agentproposalports.AgentProposalRepository
	Plans     agentplanports.AgentPlanRepository
	Decisions agentdecisionports.AgentDecisionService
	Shadow    *agentshadow.Resolver
}

// Service reads what is waiting on a person as one queue and decides
// several proposals at once. Each decision still goes through the decision
// service one at a time: the authorization, the execution, the ledger and
// the audit line are the same whether one proposal was approved or twenty.
type Service struct {
	l         *zap.Logger
	queue     agentdecisionqueueports.AgentDecisionQueueRepository
	proposals agentproposalports.AgentProposalRepository
	plans     agentplanports.AgentPlanRepository
	decisions agentdecisionports.AgentDecisionService
	shadow    shadowReader
}

type shadowReader interface {
	Organization(ctx context.Context, tenant pagination.TenantInfo) (bool, error)
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) agentdecisionqueueports.AgentDecisionQueueService {
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
	req *agentdecisionqueueports.ListPendingDecisionsRequest,
) (*agentdecisionqueueports.PendingDecisionsPage, error) {
	if withheld(req.Usable) {
		return &agentdecisionqueueports.PendingDecisionsPage{
			Items: []agentdecisionqueueports.PendingDecision{},
		}, nil
	}

	limit := req.First
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	listReq := &agentdecisionqueueports.ListPendingDecisionsQuery{
		TenantInfo:        req.TenantInfo,
		Limit:             limit,
		AgentDefinitionID: req.AgentDefinitionID,
		ToolName:          strings.TrimSpace(req.ToolName),
		Audience:          audienceOf(req.Usable),
		Now:               timeutils.NowUnix(),
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

	out := &agentdecisionqueueports.PendingDecisionsPage{
		Items:       items,
		HasNextPage: page.HasNextPage,
	}
	if req.IncludeTotalCount {
		total, countErr := s.queue.CountPending(ctx, listReq)
		if countErr != nil {
			return nil, countErr
		}
		out.TotalCount = new(int(total.N))
		out.TotalCountExact = total.Exact
	}

	return out, nil
}

// loadEntries turns queue entries into their records, in queue order. An
// entry whose record has gone (decided between the two reads) is dropped
// rather than failing the page.
func (s *Service) loadEntries(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entries []agentdecisionqueueports.PendingDecisionEntry,
) ([]agentdecisionqueueports.PendingDecision, error) {
	proposalIDs := make([]pulid.ID, 0, len(entries))
	planIDs := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		switch entry.Kind {
		case agentdecisionqueueports.PendingDecisionProposal:
			proposalIDs = append(proposalIDs, entry.ID)
		case agentdecisionqueueports.PendingDecisionPlan:
			planIDs = append(planIDs, entry.ID)
		}
	}

	proposalsByID := make(map[pulid.ID]*agent.AgentProposal, len(proposalIDs))
	if len(proposalIDs) > 0 {
		proposals, err := s.proposals.ListByIDs(
			ctx,
			agentproposalports.ListAgentProposalsByIDsRequest{
				IDs:        proposalIDs,
				TenantInfo: tenant,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, proposal := range proposals {
			proposalsByID[proposal.ID] = proposal
		}
	}

	plansByID := make(map[pulid.ID]*agentplan.AgentPlan, len(planIDs))
	if len(planIDs) > 0 {
		plans, err := s.plans.ListByIDs(ctx, agentplanports.ListAgentPlansByIDsRequest{
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

	items := make([]agentdecisionqueueports.PendingDecision, 0, len(entries))
	for _, entry := range entries {
		cursor, err := encodeCursor(entry)
		if err != nil {
			return nil, err
		}
		item := agentdecisionqueueports.PendingDecision{CreatedAt: entry.CreatedAt, Cursor: cursor}
		switch entry.Kind {
		case agentdecisionqueueports.PendingDecisionProposal:
			proposal, ok := proposalsByID[entry.ID]
			if !ok {
				continue
			}
			item.Proposal = proposal
		case agentdecisionqueueports.PendingDecisionPlan:
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
	usable *permissionports.UsableAgents,
) (int, error) {
	if withheld(usable) {
		return 0, nil
	}
	paused, err := s.shadow.Organization(ctx, tenant)
	if err != nil || paused {
		return 0, err
	}

	count, err := s.queue.CountPending(ctx, &agentdecisionqueueports.ListPendingDecisionsQuery{
		TenantInfo:     tenant,
		Audience:       audienceOf(usable),
		Now:            timeutils.NowUnix(),
		ActionableOnly: true,
	})
	if err != nil {
		return 0, err
	}

	return int(count.N), nil
}

// withheld reports a reader who may use no agent at all, so nothing any
// agent raised is theirs to see.
// Recent reads the decisions made since a moment and the proposals they
// decided. A proposal gone since is left out rather than failing the list.
func (s *Service) Recent(
	ctx context.Context,
	req agentdecisionqueueports.ListRecentDecisionsRequest,
) ([]agentdecisionqueueports.RecentDecision, error) {
	if withheld(req.Usable) {
		return []agentdecisionqueueports.RecentDecision{}, nil
	}

	entries, err := s.queue.ListRecent(ctx, agentdecisionqueueports.ListRecentDecisionsQuery{
		TenantInfo: req.TenantInfo,
		Since:      req.Since,
		Limit:      req.First,
		Audience:   audienceOf(req.Usable),
	})
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []agentdecisionqueueports.RecentDecision{}, nil
	}

	ids := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ProposalID)
	}
	proposals, err := s.proposals.ListByIDs(ctx, agentproposalports.ListAgentProposalsByIDsRequest{
		IDs:        ids,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*agent.AgentProposal, len(proposals))
	for _, proposal := range proposals {
		byID[proposal.ID] = proposal
	}

	out := make([]agentdecisionqueueports.RecentDecision, 0, len(entries))
	for _, entry := range entries {
		proposal, ok := byID[entry.ProposalID]
		if !ok {
			continue
		}
		out = append(
			out,
			agentdecisionqueueports.RecentDecision{RecentDecisionEntry: entry, Proposal: proposal},
		)
	}

	return out, nil
}

func withheld(usable *permissionports.UsableAgents) bool {
	return usable != nil && !usable.Assistant
}

func audienceOf(usable *permissionports.UsableAgents) *kernelports.AgentAudience {
	if usable == nil {
		return nil
	}

	return usable.Audience()
}

func (s *Service) Summary(
	ctx context.Context,
	tenant pagination.TenantInfo,
	usable *permissionports.UsableAgents,
) (*agentdecisionqueueports.PendingDecisionSummary, error) {
	if withheld(usable) {
		return &agentdecisionqueueports.PendingDecisionSummary{
			ByAgent: []agentdecisionqueueports.PendingDecisionAgentCount{},
			ByTool:  []agentdecisionqueueports.PendingDecisionToolCount{},
		}, nil
	}
	return s.queue.Summary(ctx, agentdecisionqueueports.PendingDecisionSummaryRequest{
		TenantInfo: tenant,
		Audience:   audienceOf(usable),
		Now:        timeutils.NowUnix(),
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
	req *agentdecisionqueueports.DecideAgentProposalsRequest,
	actor *permissionports.RequestActor,
) ([]agentdecisionports.AgentProposalDecisionResult, error) {
	if err := s.checkBatch(ctx, req); err != nil {
		return nil, err
	}

	results := make([]agentdecisionports.AgentProposalDecisionResult, 0, len(req.ProposalIDs))
	for _, id := range req.ProposalIDs {
		result := agentdecisionports.AgentProposalDecisionResult{ProposalID: id}
		outcome, decideErr := s.decisions.DecideWithOutcome(
			ctx,
			&agentdecisionports.DecideAgentProposalRequest{
				ProposalID:    id,
				Decision:      req.Decision,
				ReasonCode:    req.ReasonCode,
				Note:          req.Note,
				TenantInfo:    req.TenantInfo,
				PreviewDigest: req.PreviewDigests[id],
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
			result = outcome.Result(id)
		}
		results = append(results, result)
	}

	return results, nil
}

// checkBatch refuses a batch before anything in it is decided: one that
// names a proposal twice, a plan's step, or more than one tool.
func (s *Service) checkBatch(
	ctx context.Context,
	req *agentdecisionqueueports.DecideAgentProposalsRequest,
) error {
	if err := validateBatch(req); err != nil {
		return err
	}

	proposals, err := s.proposals.ListByIDs(ctx, agentproposalports.ListAgentProposalsByIDsRequest{
		IDs:        req.ProposalIDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
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
		return multiErr
	}

	return nil
}

func validateBatch(req *agentdecisionqueueports.DecideAgentProposalsRequest) error {
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
	case agentdecision.DecisionAccepted, agentdecision.DecisionRejected:
	case agentdecision.DecisionModified:
		multiErr.Add("decision", errortypes.ErrInvalid,
			"A change applies to one proposal; approve with changes from its card")
	default:
		multiErr.Add("decision", errortypes.ErrInvalid, "Decision is invalid")
	}
	if strings.TrimSpace(req.ReasonCode) == "" && req.Decision == agentdecision.DecisionRejected {
		multiErr.Add("reasonCode", errortypes.ErrRequired, "Say why these are rejected")
	}
	if err := agentdecision.CheckDecisionNote(
		agentdecision.NormalizeDecisionNote(req.Note),
	); err != nil {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"A note to the agent can be at most {0} characters",
			agentdecision.MaxDecisionNoteLength,
		)
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
		errortypes.IsAuthorizationError(err) || errortypes.IsVersionMismatchError(err) ||
		errortypes.IsConflictError(err) {
		return err.Error()
	}

	return "This proposal could not be decided. Try it on its own."
}

func encodeCursor(entry agentdecisionqueueports.PendingDecisionEntry) (string, error) {
	return pagination.EncodePositionCursor(
		pagination.PositionCursor{At: entry.CreatedAt, ID: entry.ID},
	)
}

func decodeCursor(encoded string) (agentdecisionqueueports.PendingDecisionCursor, error) {
	cursor, err := pagination.DecodePositionCursor(encoded)
	if err != nil {
		return agentdecisionqueueports.PendingDecisionCursor{}, err
	}

	return agentdecisionqueueports.PendingDecisionCursor{CreatedAt: cursor.At, ID: cursor.ID}, nil
}

func (s *Service) DecideManyOwn(
	ctx context.Context,
	req *agentdecisionqueueports.DecideAgentProposalsRequest,
	actor *permissionports.RequestActor,
) ([]agentdecisionports.AgentProposalDecisionResult, error) {
	if err := validateBatch(req); err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	for i, id := range req.ProposalIDs {
		if err := s.decisions.AssertOwnProposal(ctx, id, req.TenantInfo, actor); err != nil {
			if !errortypes.IsNotFoundError(err) {
				return nil, err
			}
			multiErr.Add(fmt.Sprintf("proposalIds[%d]", i), errortypes.ErrNotFound,
				"That proposal was not raised in one of your conversations")
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if s.decisions.Defers(req.Decision) {
		return s.approveDeferred(ctx, req, actor)
	}

	return s.DecideMany(ctx, req, actor)
}

// approveDeferred approves a batch from the person's own conversations in
// one undo window: every proposal is recorded now, each reporting its own
// outcome, and all of them commit together when the window closes.
func (s *Service) approveDeferred(
	ctx context.Context,
	req *agentdecisionqueueports.DecideAgentProposalsRequest,
	actor *permissionports.RequestActor,
) ([]agentdecisionports.AgentProposalDecisionResult, error) {
	if err := s.checkBatch(ctx, req); err != nil {
		return nil, err
	}

	reqs := make([]*agentdecisionports.DecideAgentProposalRequest, 0, len(req.ProposalIDs))
	for _, id := range req.ProposalIDs {
		reqs = append(reqs, &agentdecisionports.DecideAgentProposalRequest{
			ProposalID:    id,
			Decision:      req.Decision,
			ReasonCode:    req.ReasonCode,
			Note:          req.Note,
			TenantInfo:    req.TenantInfo,
			PreviewDigest: req.PreviewDigests[id],
		})
	}

	deferred, err := s.decisions.ApproveDeferred(ctx, reqs, actor)
	if err != nil {
		return nil, err
	}

	results := make([]agentdecisionports.AgentProposalDecisionResult, 0, len(deferred))
	for _, approval := range deferred {
		result := agentdecisionports.AgentProposalDecisionResult{ProposalID: approval.ProposalID}
		if approval.Err != nil {
			result.Error = decisionErrorMessage(approval.Err)
			s.l.Warn("batch approval refused for a proposal",
				zap.String("proposal", approval.ProposalID.String()),
				zap.Error(approval.Err),
			)
		} else {
			outcome := agentdecisionports.DecisionOutcome{
				Decision:       approval.Decision,
				ExecutionError: approval.ExecutionError,
			}
			result = outcome.Result(approval.ProposalID)
		}
		results = append(results, result)
	}

	return results, nil
}
