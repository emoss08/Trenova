package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StubRunRepo struct {
	agentruntimeports.AgentRunRepository

	Created []*agent.AgentRun
	Err     error
	// ByID serves reads for runs this stub did not create; reads counts them.
	ByID  map[pulid.ID]*agent.AgentRun
	Reads int

	Transcripts     map[pulid.ID]*agent.RunTranscript
	TranscriptReads []agentruntimeports.ListAgentRunsByIDsRequest
}

func (r *StubRunRepo) ListTranscriptsByIDs(
	_ context.Context,
	req agentruntimeports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	r.TranscriptReads = append(r.TranscriptReads, req)
	runs := make([]*agent.AgentRun, 0, len(req.IDs))
	for _, id := range req.IDs {
		if transcript, ok := r.Transcripts[id]; ok {
			runs = append(runs, &agent.AgentRun{ID: id, Transcript: transcript})
		}
	}

	return runs, nil
}

func (r *StubRunRepo) GetByID(
	_ context.Context,
	req agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	r.Reads++
	if run, ok := r.ByID[req.ID]; ok {
		return run, nil
	}
	for _, run := range r.Created {
		if run.ID == req.ID {
			return run, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Agent run not found")
}

type StubControlRepo struct {
	tenantports.AgentControlRepository

	Paused bool
}

func (r StubControlRepo) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	return &agentcontrol.AgentControl{ShadowMode: r.Paused}, nil
}

type StubDefinitionRepo struct {
	agentdefinitionports.AgentDefinitionRepository

	Definition *agentdefinition.Definition
}

func (r StubDefinitionRepo) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if r.Definition == nil || r.Definition.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Agent definition not found")
	}

	return r.Definition, nil
}

// ShadowSwitches is the pair of switches a proposal can be held behind.
type ShadowSwitches struct {
	OrganizationPaused bool
	Definition         *agentdefinition.Definition
}

func (r *StubRunRepo) Create(
	_ context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	if r.Err != nil {
		return nil, r.Err
	}

	entity.ID = pulid.MustNew("ar_")
	r.Created = append(r.Created, entity)

	return entity, nil
}

type StubProposalRepo struct {
	Created  []*agent.AgentProposal
	ByThread []*agent.AgentProposal
	err      error
	ListErr  error
	LastList agentproposalports.ListAgentProposalsByThreadRequest

	ByRun   []*agent.AgentProposal
	LastRun agentproposalports.ListAgentProposalsByRunRequest

	ExpiredThreads []pulid.ID
}

func (r *StubProposalRepo) ListByRun(
	_ context.Context,
	req agentproposalports.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	r.LastRun = req
	if r.ListErr != nil {
		return nil, r.ListErr
	}

	return r.ByRun, nil
}

func (r *StubProposalRepo) Create(
	_ context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	if r.err != nil {
		return nil, r.err
	}

	entity.ID = pulid.MustNew("ap_")
	r.Created = append(r.Created, entity)

	return entity, nil
}

func (r *StubProposalRepo) ListByThread(
	_ context.Context,
	req agentproposalports.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	r.LastList = req
	if r.ListErr != nil {
		return nil, r.ListErr
	}

	return r.ByThread, nil
}

// The conversation repository is embedded rather than implemented: this suite
// only reads a thread, and an unexpected call panics on the nil interface, which
// is exactly the failure a test wants.
type StubConversationRepo struct {
	conversationports.ConversationRepository

	Thread *conversation.Thread
	GetErr error
}

func (r *StubConversationRepo) GetThread(
	_ context.Context,
	_ conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	if r.GetErr != nil {
		return nil, r.GetErr
	}

	return r.Thread, nil
}

func (r *StubProposalRepo) ExpirePendingByThread(
	_ context.Context,
	req agentproposalports.ExpireAgentProposalsByThreadRequest,
) (int, error) {
	r.ExpiredThreads = append(r.ExpiredThreads, req.ThreadID)

	return 0, nil
}
