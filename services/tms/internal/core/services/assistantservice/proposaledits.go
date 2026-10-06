package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type proposalEditStore interface {
	SetPendingModifications(
		ctx context.Context,
		req repositories.SetPendingModificationsRequest,
	) (*agent.AgentProposal, error)
}

// SaveProposalEdits keeps the values a person changed on one of the
// conversation's pending proposals, such as the wording of a drafted message,
// so the edit is still there after a reload and goes with the approval.
//
// Reading the thread first is the authorization check, as for listing its
// proposals, and the proposal must be one the thread raised: a proposal from
// someone else's conversation is not found. Only what differs from the
// agent's own values is kept, and an empty set clears what was saved.
func (s *Service) SaveProposalEdits(
	ctx context.Context,
	req repositories.GetThreadRequest,
	proposalID pulid.ID,
	modifications map[string]any,
) (*services.ProposalEdits, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.proposals == nil || s.proposalEdits == nil {
		return nil, errortypes.NewNotFoundError("Proposal not found in this conversation")
	}

	stored, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	var proposal *agent.AgentProposal
	for _, candidate := range stored {
		if candidate != nil && candidate.ID == proposalID {
			proposal = candidate
			break
		}
	}
	if proposal == nil {
		return nil, errortypes.NewNotFoundError("Proposal not found in this conversation")
	}
	if err = editable(proposal); err != nil {
		return nil, err
	}

	changed := toolschema.Changed(proposal.ToolParams, modifications)
	if err = s.checkEditedFields(proposal, changed); err != nil {
		return nil, err
	}
	if len(changed) == 0 {
		changed = nil
	}

	saved, err := s.proposalEdits.SetPendingModifications(
		ctx,
		repositories.SetPendingModificationsRequest{
			ID:            proposal.ID,
			TenantInfo:    req.TenantInfo,
			Modifications: changed,
		},
	)
	if err != nil {
		return nil, err
	}

	return &services.ProposalEdits{
		ProposalID:           saved.ID,
		PendingModifications: saved.PendingModifications,
	}, nil
}

// editable refuses a proposal no longer waiting on a decision: an edit kept
// on it would go with no approval.
func editable(proposal *agent.AgentProposal) error {
	if proposal.Status != agent.ProposalStatusPending {
		return errortypes.NewBusinessError(
			"This proposal has already been decided: it is {0}",
			strings.ToLower(string(proposal.Status)),
		)
	}
	if proposal.Expired(timeutils.NowUnix()) {
		return errortypes.NewBusinessError(
			"This proposal expired without a decision. Ask the agent again for a current one",
		)
	}

	return nil
}

// checkEditedFields refuses a change to a parameter a person may not edit:
// one the tool does not take, or the one naming the record it acts on. The
// approval checks the values again; this keeps what is saved to what the
// approval could carry.
func (s *Service) checkEditedFields(proposal *agent.AgentProposal, changed map[string]any) error {
	if s.tools == nil || len(changed) == 0 {
		return nil
	}

	editableNames := make(map[string]struct{})
	for _, field := range s.editableFields(proposal) {
		if !field.ReadOnly {
			editableNames[field.Name] = struct{}{}
		}
	}

	me := errortypes.NewMultiError()
	for name := range changed {
		if _, ok := editableNames[name]; !ok {
			me.Add(name, errortypes.ErrInvalid, "This value cannot be changed before approving")
		}
	}
	if me.HasErrors() {
		return me
	}

	return nil
}
