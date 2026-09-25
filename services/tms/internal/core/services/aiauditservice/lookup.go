package aiauditservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// owner is everything a row learns from the run or turn it belongs to: who
// the work was for, which agent did it, and how to find it again.
type owner struct {
	kind           agent.RunOwnerKind
	id             pulid.ID
	runID          pulid.ID
	turnID         pulid.ID
	threadID       pulid.ID
	agentID        pulid.ID
	agentVersion   *int64
	principalType  aiaudit.PrincipalType
	principalID    string
	onBehalfOf     pulid.ID
	traceID        string
	purpose        aiaudit.Purpose
	parentOwnerID  pulid.ID
	delegateCallID string
	startAt        int64
	terminal       bool
	simulated      bool
	known          bool
}

// startedAt is when the owner began, when it is known.
func (o owner) startedAt() (int64, bool) {
	return o.startAt, o.startAt > 0
}

// needs collects, across one tenant's source rows in a pass, what has to be
// read to explain them.
type needs struct {
	runs        map[pulid.ID]struct{}
	turns       map[pulid.ID]struct{}
	evaluations map[pulid.ID]struct{}
	proposals   map[pulid.ID]struct{}
	usageTurns  map[pulid.ID]struct{}
	calls       []repositories.OwnerCall
	windowFrom  int64
	windowTo    int64
}

func newNeeds() *needs {
	return &needs{
		runs:        make(map[pulid.ID]struct{}),
		turns:       make(map[pulid.ID]struct{}),
		evaluations: make(map[pulid.ID]struct{}),
		proposals:   make(map[pulid.ID]struct{}),
		usageTurns:  make(map[pulid.ID]struct{}),
	}
}

func (n *needs) owner(kind string, id pulid.ID) {
	if id.IsNil() {
		return
	}

	switch {
	case id.Prefix() == agent.EvaluationIDPrefix:
		n.evaluations[id] = struct{}{}
	case kind == string(agent.RunOwnerAssistantTurn):
		n.turns[id] = struct{}{}
	default:
		n.runs[id] = struct{}{}
	}
}

func (n *needs) usageWindow(threadID pulid.ID, at int64) {
	if threadID.IsNil() || at <= 0 {
		return
	}
	n.usageTurns[threadID] = struct{}{}
	if n.windowFrom == 0 || at < n.windowFrom {
		n.windowFrom = at
	}
	if at > n.windowTo {
		n.windowTo = at
	}
}

func idsOf(set map[pulid.ID]struct{}) []pulid.ID {
	ids := make([]pulid.ID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}

	return ids
}

// lookups is what one tenant's pass read to explain its rows.
type lookups struct {
	runs        map[pulid.ID]*agent.AgentRun
	turns       map[pulid.ID]*conversation.AssistantTurn
	turnByRun   map[pulid.ID]*conversation.AssistantTurn
	threads     map[pulid.ID]*conversation.Thread
	evaluations map[pulid.ID]*agent.Evaluation
	proposals   map[pulid.ID]*agent.AgentProposal
	threadTurns map[pulid.ID][]*conversation.AssistantTurn
	users       map[pulid.ID]string
	agents      map[pulid.ID]string
	calls       map[repositories.OwnerCall]struct{}
}

func emptyLookups() *lookups {
	return &lookups{
		runs:        map[pulid.ID]*agent.AgentRun{},
		turns:       map[pulid.ID]*conversation.AssistantTurn{},
		turnByRun:   map[pulid.ID]*conversation.AssistantTurn{},
		threads:     map[pulid.ID]*conversation.Thread{},
		evaluations: map[pulid.ID]*agent.Evaluation{},
		proposals:   map[pulid.ID]*agent.AgentProposal{},
		threadTurns: map[pulid.ID][]*conversation.AssistantTurn{},
		users:       map[pulid.ID]string{},
		agents:      map[pulid.ID]string{},
		calls:       map[repositories.OwnerCall]struct{}{},
	}
}

const usageTurnSlackSeconds = 3600

// load reads what a tenant's rows name, in as few queries as the chain of
// references allows: runs, turns and evaluations first, then the turns and
// conversations those point at.
func load(
	ctx context.Context,
	source repositories.AIAuditSourceRepository,
	tenantInfo pagination.TenantInfo,
	need *needs,
) (*lookups, error) {
	found := emptyLookups()

	proposals, err := source.ProposalsByIDs(ctx, tenantInfo, idsOf(need.proposals))
	if err != nil {
		return nil, err
	}
	for _, proposal := range proposals {
		found.proposals[proposal.ID] = proposal
		need.owner(string(agent.RunOwnerAgentRun), proposal.RunID)
	}

	runs, err := source.RunsByIDs(ctx, tenantInfo, idsOf(need.runs))
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		found.runs[run.ID] = run
		if run.TurnID.IsNotNil() {
			need.turns[run.TurnID] = struct{}{}
		}
		if run.ParentOwnerKind == agent.RunOwnerAssistantTurn && run.ParentOwnerID.IsNotNil() {
			need.turns[run.ParentOwnerID] = struct{}{}
		}
	}

	byRun, err := source.TurnsByRunIDs(ctx, tenantInfo, idsOf(need.runs))
	if err != nil {
		return nil, err
	}
	for _, turn := range byRun {
		found.turnByRun[turn.RunID] = turn
		found.turns[turn.ID] = turn
		delete(need.turns, turn.ID)
	}

	turns, err := source.TurnsByIDs(ctx, tenantInfo, idsOf(need.turns))
	if err != nil {
		return nil, err
	}
	for _, turn := range turns {
		found.turns[turn.ID] = turn
		if turn.RunID.IsNotNil() {
			if _, seen := found.turnByRun[turn.RunID]; !seen {
				found.turnByRun[turn.RunID] = turn
			}
		}
	}

	if err = loadUsageTurns(ctx, source, tenantInfo, need, found); err != nil {
		return nil, err
	}

	threadIDs := make(map[pulid.ID]struct{}, len(found.turns))
	for _, turn := range found.turns {
		threadIDs[turn.ThreadID] = struct{}{}
	}
	threads, err := source.ThreadsByIDs(ctx, tenantInfo, idsOf(threadIDs))
	if err != nil {
		return nil, err
	}
	for _, thread := range threads {
		found.threads[thread.ID] = thread
	}

	evaluations, err := source.EvaluationsByIDs(ctx, tenantInfo, idsOf(need.evaluations))
	if err != nil {
		return nil, err
	}
	for _, evaluation := range evaluations {
		found.evaluations[evaluation.ID] = evaluation
	}

	if found.calls, err = source.ToolStepCalls(ctx, tenantInfo, need.calls); err != nil {
		return nil, err
	}

	return found, nil
}

func loadUsageTurns(
	ctx context.Context,
	source repositories.AIAuditSourceRepository,
	tenantInfo pagination.TenantInfo,
	need *needs,
	found *lookups,
) error {
	if len(need.usageTurns) == 0 {
		return nil
	}

	turns, err := source.TurnsInWindow(ctx, repositories.AIAuditTurnWindow{
		TenantInfo: tenantInfo,
		ThreadIDs:  idsOf(need.usageTurns),
		From:       need.windowFrom - usageTurnSlackSeconds,
		To:         need.windowTo + usageTurnSlackSeconds,
	})
	if err != nil {
		return fmt.Errorf("read the turns usage rows were for: %w", err)
	}
	for _, turn := range turns {
		found.turns[turn.ID] = turn
		found.threadTurns[turn.ThreadID] = append(found.threadTurns[turn.ThreadID], turn)
	}

	return nil
}

// loadNames reads the names the rows are snapshotted with, once every other
// lookup has said which people and agents they name.
func loadNames(
	ctx context.Context,
	source repositories.AIAuditSourceRepository,
	tenantInfo pagination.TenantInfo,
	found *lookups,
	events []*aiaudit.AIAuditEvent,
) error {
	users := make(map[pulid.ID]struct{}, len(events))
	agents := make(map[pulid.ID]struct{}, len(events))
	for _, event := range events {
		if event.OnBehalfOfUserID.IsNotNil() {
			users[event.OnBehalfOfUserID] = struct{}{}
		}
		if event.DecidedByUserID.IsNotNil() {
			users[event.DecidedByUserID] = struct{}{}
		}
		if event.AgentDefinitionID.IsNotNil() {
			agents[event.AgentDefinitionID] = struct{}{}
		}
	}

	var err error
	if found.users, err = source.UserNames(ctx, tenantInfo, idsOf(users)); err != nil {
		return err
	}
	if found.agents, err = source.AgentNames(ctx, tenantInfo, idsOf(agents)); err != nil {
		return err
	}

	return nil
}

// ownerOf explains a run, turn or evaluation id. An owner that can no longer
// be read is still named, with nothing more known about it.
func (l *lookups) ownerOf(kind string, id pulid.ID) owner {
	if id.IsNil() {
		return owner{principalType: aiaudit.PrincipalSystem, purpose: aiaudit.PurposeLive}
	}

	if id.Prefix() == agent.EvaluationIDPrefix {
		return l.evaluationOwner(id)
	}
	if kind == string(agent.RunOwnerAssistantTurn) {
		return l.turnOwner(id)
	}

	return l.runOwner(id)
}

func (l *lookups) evaluationOwner(id pulid.ID) owner {
	found := owner{
		kind:          agent.RunOwnerAgentRun,
		id:            id,
		runID:         id,
		principalType: aiaudit.PrincipalAgent,
		purpose:       aiaudit.PurposeEvaluation,
		simulated:     true,
		terminal:      true,
	}
	if evaluation, ok := l.evaluations[id]; ok {
		found.known = true
		found.agentID = evaluation.AgentDefinitionID
		found.principalID = evaluation.AgentDefinitionID.String()
		version := evaluation.DefinitionVersion
		found.agentVersion = &version
	}

	return found
}

func (l *lookups) turnOwner(id pulid.ID) owner {
	found := owner{
		kind:          agent.RunOwnerAssistantTurn,
		id:            id,
		turnID:        id,
		principalType: aiaudit.PrincipalUser,
		purpose:       aiaudit.PurposeLive,
	}

	turn, ok := l.turns[id]
	if !ok {
		return found
	}

	found.known = true
	found.threadID = turn.ThreadID
	found.runID = turn.RunID
	found.onBehalfOf = turn.UserID
	found.principalID = turn.UserID.String()
	found.traceID = turn.TraceID
	found.startAt = firstPositive(turn.StartedAt, turn.CreatedAt)
	found.terminal = turn.Status.Terminal()
	if turn.Fingerprint != nil {
		version := turn.Fingerprint.DefinitionVersion
		found.agentVersion = &version
	}
	if thread, threadOK := l.threads[turn.ThreadID]; threadOK {
		found.agentID = thread.AgentDefinitionID
	}

	return found
}

func (l *lookups) runOwner(id pulid.ID) owner {
	found := owner{
		kind:          agent.RunOwnerAgentRun,
		id:            id,
		runID:         id,
		principalType: aiaudit.PrincipalAgent,
		purpose:       aiaudit.PurposeLive,
	}

	run, ok := l.runs[id]
	if !ok {
		return found
	}

	found.known = true
	found.agentID = run.AgentDefinitionID
	found.principalID = run.AgentDefinitionID.String()
	found.traceID = run.TraceID
	found.startAt = firstPositive(run.StartedAt, run.CreatedAt)
	found.terminal = run.CompletedAt != nil
	found.simulated = run.Status == agent.RunStatusShadowCompleted
	found.parentOwnerID = run.ParentOwnerID
	found.delegateCallID = run.DelegateCallID
	if run.Fingerprint != nil {
		version := run.Fingerprint.DefinitionVersion
		found.agentVersion = &version
	}

	turn := l.turnForRun(run)
	if turn != nil {
		found.turnID = turn.ID
		found.threadID = turn.ThreadID
		found.onBehalfOf = turn.UserID
	}

	return found
}

func (l *lookups) turnForRun(run *agent.AgentRun) *conversation.AssistantTurn {
	if run.TurnID.IsNotNil() {
		if turn, ok := l.turns[run.TurnID]; ok {
			return turn
		}
	}
	if turn, ok := l.turnByRun[run.ID]; ok {
		return turn
	}
	if run.ParentOwnerKind == agent.RunOwnerAssistantTurn {
		if turn, ok := l.turns[run.ParentOwnerID]; ok {
			return turn
		}
	}

	return nil
}

// turnAt is the turn of a conversation that was running at a moment, for a
// usage row written before rows named their turn.
func (l *lookups) turnAt(threadID pulid.ID, at int64) *conversation.AssistantTurn {
	var best *conversation.AssistantTurn
	for _, turn := range l.threadTurns[threadID] {
		if turn.StartedAt > at+1 {
			continue
		}
		if turn.CompletedAt != nil && *turn.CompletedAt < at-1 {
			continue
		}
		if best == nil || turn.StartedAt > best.StartedAt {
			best = turn
		}
	}

	return best
}
