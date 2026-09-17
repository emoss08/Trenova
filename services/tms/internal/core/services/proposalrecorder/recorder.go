package proposalrecorder

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RunOpener interface {
	Create(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
}

type ProposalStore interface {
	Create(ctx context.Context, entity *agent.AgentProposal) (*agent.AgentProposal, error)
}

type Params struct {
	fx.In

	Logger    *zap.Logger
	Runs      repositories.AgentRunRepository
	Proposals repositories.AgentProposalRepository
}

type Service struct {
	logger    *zap.Logger
	runs      RunOpener
	proposals ProposalStore
}

func New(p Params) *Service {
	return NewWithStores(p.Logger, p.Runs, p.Proposals)
}

func NewWithStores(logger *zap.Logger, runs RunOpener, proposals ProposalStore) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		logger:    logger.Named("service.proposalrecorder"),
		runs:      runs,
		proposals: proposals,
	}
}

type OpenRunRequest struct {
	AgentType        agent.Type
	SubjectType      agent.SubjectType
	SubjectID        pulid.ID
	Trigger          agent.RunTrigger
	Status           agent.RunStatus
	Model            string
	PromptVersion    string
	InputContextHash string
	Summary          string
}

type EvidenceFunc func(action serviceports.PendingAction, sourceMessageID pulid.ID) []agent.EvidenceRef

type RecordRequest struct {
	Actor      *serviceports.RequestActor
	Definition *agentdefinition.Definition
	// Run is the run the proposals belong to. When nil, one is opened from Open.
	Run              *agent.AgentRun
	Open             *OpenRunRequest
	Actions          []serviceports.PendingAction
	SourceMessageIDs map[string]pulid.ID
	Evidence         EvidenceFunc
}

type RecordResult struct {
	Run       *agent.AgentRun
	Proposals []*agent.AgentProposal
}

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*RecordResult, error) {
	if len(req.Actions) == 0 {
		return &RecordResult{Run: req.Run}, nil
	}

	run := req.Run
	if run == nil {
		opened, err := s.openRun(ctx, req)
		if err != nil {
			return nil, err
		}
		run = opened
	}

	now := timeutils.NowUnix()
	proposals := make([]*agent.AgentProposal, 0, len(req.Actions))

	for _, action := range req.Actions {
		sourceMessageID := req.SourceMessageIDs[action.ToolCallID]

		proposal := &agent.AgentProposal{
			OrganizationID:  req.Actor.OrganizationID,
			BusinessUnitID:  req.Actor.BusinessUnitID,
			RunID:           run.ID,
			ToolName:        action.ToolName,
			ToolParams:      nonNilParams(action.Arguments),
			Rationale:       action.Rationale,
			AutonomyTier:    proposalTier(action.Tier),
			Status:          agent.ProposalStatusPending,
			SourceMessageID: sourceMessageID,
		}
		if req.Evidence != nil {
			proposal.Evidence = req.Evidence(action, sourceMessageID)
		}
		applyExecution(proposal, action, now)

		multiErr := errortypes.NewMultiError()
		proposal.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, multiErr
		}

		created, err := s.proposals.Create(ctx, proposal)
		if err != nil {
			return nil, err
		}

		proposals = append(proposals, created)
	}

	return &RecordResult{Run: run, Proposals: proposals}, nil
}

func (s *Service) openRun(ctx context.Context, req *RecordRequest) (*agent.AgentRun, error) {
	open := req.Open
	if open == nil {
		open = &OpenRunRequest{}
	}

	now := timeutils.NowUnix()
	status := open.Status
	if status == "" {
		status = agent.RunStatusCompleted
	}

	run := &agent.AgentRun{
		OrganizationID:   req.Actor.OrganizationID,
		BusinessUnitID:   req.Actor.BusinessUnitID,
		AgentType:        open.AgentType,
		SubjectType:      open.SubjectType,
		SubjectID:        open.SubjectID,
		Trigger:          open.Trigger,
		Status:           status,
		ModelIdentifier:  open.Model,
		PromptVersion:    open.PromptVersion,
		InputContextHash: open.InputContextHash,
		Summary:          open.Summary,
		StartedAt:        now,
		CompletedAt:      &now,
	}
	if req.Definition != nil {
		run.AgentDefinitionID = req.Definition.ID
	}
	if run.Trigger == "" {
		run.Trigger = agent.RunTriggerManual
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.runs.Create(ctx, run)
}

func applyExecution(proposal *agent.AgentProposal, action serviceports.PendingAction, now int64) {
	if !action.Executed {
		return
	}

	executedAt := now
	proposal.ExecutedAt = &executedAt
	if action.ExecutionError != "" {
		proposal.Status = agent.ProposalStatusExecutionFailed
		proposal.ExecutionError = action.ExecutionError

		return
	}

	proposal.Status = agent.ProposalStatusExecuted
}

func proposalTier(tier agent.AutonomyTier) agent.AutonomyTier {
	if tier == "" {
		return agent.TierPropose
	}

	return tier
}

func nonNilParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}

	return params
}
