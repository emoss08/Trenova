package deskbench

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	firstTurnWithin  = 2 * time.Minute
	followUpWithin   = 3 * time.Minute
	cleanupWithin    = 45 * time.Second
	settleAfterTurn  = 5 * time.Second
	turnTimeout      = 15 * time.Minute
	relayGrace       = 20 * time.Second
	lookupAttempts   = 25
	lookupEvery      = 200 * time.Millisecond
	windowLead       = time.Second
	cleanupNote      = "Closed by the desk bench at the end of its run."
	DecisionApprove  = DecisionAction("approve")
	DecisionReject   = DecisionAction("reject")
	benchTitlePrefix = "[bench] "
)

var ErrNothingToDecide = errors.New("no proposal is waiting for a decision in this conversation")

type DecisionAction string

type Utterance struct {
	Content  string
	Page     *agent.PageContext
	Mentions []agent.EntityRef
	Surface  agent.Surface
	Directed pulid.ID
}

type Decide struct {
	Action        DecisionAction
	Note          string
	Tool          string
	Modifications map[string]any
}

type OpenConversation struct {
	Label    string
	Agent    *agentdefinition.Definition
	Provider *aiprovider.Provider
	Title    string
	ThreadID pulid.ID
}

type Conversation struct {
	Label    string
	session  *Session
	Agent    *agentdefinition.Definition
	Provider *aiprovider.Provider
	Thread   *conversation.Thread
	queue    *turnQueue
	turns    []*TurnRecord
	formula  *formulaState
}

func (s *Session) Open(ctx context.Context, req OpenConversation) (*Conversation, error) {
	ctx = s.Context(ctx)

	var thread *conversation.Thread
	var err error
	if req.ThreadID.IsNotNil() {
		thread, err = s.bench.Assistant.GetThread(ctx, s.threadRequest(req.ThreadID))
		if err != nil {
			return nil, fmt.Errorf("open thread %s: %w", req.ThreadID, err)
		}
	} else {
		if req.Agent == nil {
			return nil, errors.New("a new conversation needs an agent")
		}
		thread, err = s.bench.Assistant.StartThread(ctx, &serviceports.StartThreadRequest{
			AgentDefinitionID: req.Agent.ID,
			Title:             benchTitlePrefix + strings.TrimSpace(req.Title),
			TenantInfo:        s.Tenant,
			Origin:            conversation.ThreadOriginDesk,
		}, &s.Actor)
		if err != nil {
			return nil, fmt.Errorf("start a conversation with %s: %w", req.Agent.Name, err)
		}
	}

	label := req.Label
	if label == "" {
		label = thread.ID.String()
	}

	return &Conversation{
		Label:    label,
		session:  s,
		Agent:    req.Agent,
		Provider: req.Provider,
		Thread:   thread,
		queue:    s.bench.watcher.watch(thread.ID),
	}, nil
}

func (s *Session) threadRequest(threadID pulid.ID) repositories.GetThreadRequest {
	return repositories.GetThreadRequest{ID: threadID, UserID: s.User.ID, TenantInfo: s.Tenant}
}

func (c *Conversation) Turns() []*TurnRecord {
	return c.turns
}

func (c *Conversation) Say(ctx context.Context, u Utterance) ([]*TurnRecord, error) {
	ctx = c.session.Context(ctx)
	bench := c.session.bench

	surface := u.Surface
	if surface == "" {
		surface = agent.SurfaceDesk
	}
	if c.formula != nil {
		u.Page = c.formula.page()
		surface = agent.SurfaceAssistant
	}

	_, err := bench.Turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   c.Thread.ID,
		UserID:     c.session.User.ID,
		TenantInfo: c.session.Tenant,
		Origin:     conversation.AssistantTurnOriginPerson,
		Input:      u.Content,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		start := assistantjobs.TurnStart{
			Actor:   c.session.Actor,
			Content: u.Content,
			Request: assistantjobs.AssistantTurnRequest{
				Page:                u.Page,
				Surface:             surface,
				Mentions:            u.Mentions,
				PreferredProviderID: providerID(c.Provider),
				ProviderChosen:      c.Provider != nil,
				DirectedAgentID:     u.Directed,
				Awaited:             true,
			},
		}
		var runID string
		startErr := retryNamespaceLookup(ctx, func() error {
			run, err := assistantjobs.StartTurnWorkflow(ctx, bench.watcher, turn, start)
			if err != nil {
				return err
			}
			runID = run.GetID()

			return nil
		})

		return runID, startErr
	})
	if err != nil {
		return nil, fmt.Errorf("send the message: %w", err)
	}

	return c.collect(ctx, firstTurnWithin)
}

func (c *Conversation) Decide(ctx context.Context, d Decide) ([]*DecisionRecord, []*TurnRecord, error) {
	return c.decide(ctx, d, followUpWithin)
}

func (c *Conversation) decide(
	ctx context.Context,
	d Decide,
	within time.Duration,
) ([]*DecisionRecord, []*TurnRecord, error) {
	ctx = c.session.Context(ctx)

	pending, err := c.pending(ctx, d.Tool)
	if err != nil {
		return nil, nil, err
	}
	if len(pending) == 0 {
		return nil, nil, ErrNothingToDecide
	}

	decisions := make([]*DecisionRecord, 0, len(pending))
	plans := make(map[pulid.ID]struct{}, 2)
	for _, proposal := range pending {
		if proposal.PlanID.IsNotNil() {
			if _, done := plans[proposal.PlanID]; done {
				continue
			}
			plans[proposal.PlanID] = struct{}{}
			decisions = append(decisions, c.decidePlan(ctx, proposal, d))

			continue
		}
		decisions = append(decisions, c.decideProposal(ctx, proposal, d))
	}

	for _, decision := range decisions {
		c.session.bench.live.Line(c.Label, "decided %s %s%s", decision.Decision, decision.Tool,
			errorSuffix(decision.Error))
	}
	turns, err := c.collect(ctx, within)
	c.settleDecisions(ctx, decisions)
	for _, decision := range decisions {
		c.session.bench.live.Line(c.Label, "%s %s -> %s%s", decision.Decision, decision.Tool,
			decision.StatusAfter, errorSuffix(decision.ExecutionError))
	}

	return decisions, turns, err
}

type CloseOptions struct {
	LeavePending bool
	Delete       bool
}

func (c *Conversation) Close(ctx context.Context, opts CloseOptions) ([]*TurnRecord, error) {
	bench := c.session.bench
	defer bench.watcher.forget(c.Thread.ID)

	var turns []*TurnRecord
	if !opts.LeavePending {
		cleanup := Decide{Action: DecisionReject, Note: cleanupNote}
		if _, closing, err := c.decide(ctx, cleanup, cleanupWithin); err == nil {
			turns = closing
		} else if !errors.Is(err, ErrNothingToDecide) {
			return nil, fmt.Errorf("reject what the run left waiting: %w", err)
		}
	}

	if !opts.Delete && c.formula == nil {
		return turns, nil
	}

	err := bench.Assistant.DeleteThread(
		c.session.Context(ctx),
		c.session.threadRequest(c.Thread.ID),
	)
	if err != nil {
		return turns, fmt.Errorf("delete the bench conversation: %w", err)
	}

	return turns, nil
}

func (c *Conversation) collect(ctx context.Context, within time.Duration) ([]*TurnRecord, error) {
	records := make([]*TurnRecord, 0, 2)
	wait := within
	for {
		timer := time.NewTimer(wait)
		started, ok := c.queue.next(ctx, timer.C)
		timer.Stop()
		if !ok {
			break
		}

		record := c.capture(ctx, started)
		records = append(records, record)
		c.turns = append(c.turns, record)
		if c.formula != nil {
			c.formula.apply(record.Artifacts)
		}
		wait = settleAfterTurn
	}

	return records, ctx.Err()
}

func (c *Conversation) capture(ctx context.Context, started StartedTurn) *TurnRecord {
	bench := c.session.bench
	record := &TurnRecord{
		TurnID:    started.TurnID,
		ThreadID:  started.ThreadID,
		Input:     started.Content,
		StartedAt: started.At,
		Status:    conversation.AssistantTurnStatusRunning,
	}

	turn, lookupErr := c.lookupTurn(ctx, started.TurnID)
	if lookupErr != nil {
		record.CaptureError = lookupErr.Error()
	} else {
		record.Origin = turn.Origin
		record.TraceID = turn.TraceID
	}

	live := bench.live.turn(c.Label)
	live.line("== turn %s (%s) ==", started.TurnID, stringutils.WithDefault(string(record.Origin), "Person"))
	if started.Content != "" {
		live.line("person: %s", started.Content)
	}
	collector := newEventCollector(live)
	relayCtx, cancelRelay := context.WithCancel(ctx)
	defer cancelRelay()
	relayDone := make(chan error, 1)
	if turn != nil {
		go func() {
			relayDone <- bench.Turns.Relay(relayCtx, assistantturnservice.RelayRequest{Turn: turn},
				collector.onFrame)
		}()
	} else {
		relayDone <- nil
	}

	runCtx, cancelRun := context.WithTimeout(ctx, turnTimeout)
	var outcome assistantjobs.AssistantTurnResult
	runErr := started.Run.Get(runCtx, &outcome)
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	cancelRun()
	if timedOut && turn != nil {
		if stopErr := bench.Turns.Stop(ctx, turn); stopErr != nil {
			record.CaptureError = joinErrors(record.CaptureError, "stop: "+stopErr.Error())
		}
	}

	select {
	case relayErr := <-relayDone:
		if relayErr != nil && !errors.Is(relayErr, context.Canceled) {
			record.CaptureError = joinErrors(record.CaptureError, "relay: "+relayErr.Error())
		}
	case <-time.After(relayGrace):
		cancelRelay()
		<-relayDone
	}
	record.FinishedAt = time.Now()

	if runErr != nil {
		record.Failure = runErr.Error()
		record.Status = conversation.AssistantTurnStatusFailed
	}
	c.fill(ctx, record, &outcome)
	collector.into(record)

	if final, err := c.lookupTurn(ctx, started.TurnID); err == nil {
		record.Status = final.Status
		if final.ErrorMessage != "" && record.Failure == "" {
			record.Failure = final.ErrorMessage
		}
	}

	from := record.StartedAt.Add(-windowLead)
	record.ModelCalls = bench.recorder.For(record.ThreadID, record.TurnID, from, record.FinishedAt)
	sortCalls(record.ModelCalls)
	record.Calls = summarizeCalls(record.ModelCalls)
	record.Usage, record.Models = usageOf(record.ModelCalls)
	record.Unattributed = summarizeCalls(bench.recorder.Unattributed(from, record.FinishedAt))

	live.flush()
	live.line("== %s · %s · %d tool calls · %s ==", record.Status,
		record.Duration().Round(100*time.Millisecond), len(record.Tools), usageLine(record.Usage))
	for _, proposal := range record.Proposals {
		live.line("proposal %s %s (%s)", proposal.Tool,
			stringutils.Ellipsize(compactJSON(proposal.Arguments, liveArgsLimit), liveArgsLimit), proposal.Status)
	}
	if record.Failure != "" {
		live.line("failure: %s", record.Failure)
	}

	return record
}

func (c *Conversation) fill(
	ctx context.Context,
	record *TurnRecord,
	outcome *assistantjobs.AssistantTurnResult,
) {
	if outcome.Status != "" {
		record.Status = conversation.AssistantTurnStatus(outcome.Status)
	}
	record.Refused = outcome.Refused
	if outcome.Message != "" {
		record.Failure = joinErrors(record.Failure, outcome.Message)
	}

	result := outcome.Result
	if result == nil {
		return
	}

	record.Reply = result.Reply
	record.Refused = record.Refused || result.Refused
	record.Messages = result.Messages
	record.Artifacts = result.Artifacts
	record.Proposals = make([]*ProposalRecord, 0, len(result.Proposals))
	for idx := range result.Proposals {
		record.Proposals = append(record.Proposals, c.describeProposal(ctx, &result.Proposals[idx]))
	}
}

func (c *Conversation) describeProposal(
	ctx context.Context,
	summary *serviceports.AssistantProposal,
) *ProposalRecord {
	record := &ProposalRecord{
		ID:        summary.ID,
		PlanID:    summary.PlanID,
		PlanStep:  summary.PlanStep,
		Tool:      summary.ToolName,
		Arguments: summary.Arguments,
		Rationale: summary.Rationale,
		Tier:      summary.AutonomyTier,
		Status:    summary.Status,
		Simulated: summary.Simulation,
	}

	proposal, err := c.loadProposal(ctx, summary.ID)
	if err != nil {
		record.Error = err.Error()
		return record
	}
	record.Status = proposal.Status
	record.HeldBy = proposal.HeldBy
	record.Executed = proposal.ExecutionResult

	if proposal.Status != agent.ProposalStatusPending {
		return record
	}

	preview, err := c.session.bench.Previews.ForProposal(ctx, &serviceports.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   &serviceports.PreviewViewer{Actor: &c.session.Actor},
	})
	if err != nil {
		record.Error = "preview: " + err.Error()
		return record
	}
	record.Preview = preview

	return record
}

func (c *Conversation) loadProposal(ctx context.Context, id pulid.ID) (*agent.AgentProposal, error) {
	return c.session.bench.Proposals.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         id,
		TenantInfo: &c.session.Tenant,
	})
}

func (c *Conversation) pending(ctx context.Context, tool string) ([]*ProposalRecord, error) {
	seen := make(map[pulid.ID]struct{}, 4)
	pending := make([]*ProposalRecord, 0, 4)
	for _, turn := range c.turns {
		for _, proposal := range turn.Proposals {
			if _, dup := seen[proposal.ID]; dup {
				continue
			}
			seen[proposal.ID] = struct{}{}
			if tool != "" && !strings.EqualFold(proposal.Tool, tool) {
				continue
			}

			current, err := c.loadProposal(ctx, proposal.ID)
			if err != nil {
				return nil, fmt.Errorf("read proposal %s: %w", proposal.ID, err)
			}
			if current.Status != agent.ProposalStatusPending {
				continue
			}
			pending = append(pending, proposal)
		}
	}

	return pending, nil
}

func (d Decide) decision() agent.DecisionType {
	if d.Action == DecisionReject {
		return agent.DecisionRejected
	}
	if len(d.Modifications) > 0 {
		return agent.DecisionModified
	}

	return agent.DecisionAccepted
}

func (c *Conversation) decideProposal(
	ctx context.Context,
	proposal *ProposalRecord,
	d Decide,
) *DecisionRecord {
	bench := c.session.bench
	decision := d.decision()
	record := &DecisionRecord{
		ProposalID: proposal.ID,
		Tool:       proposal.Tool,
		Decision:   decision,
		Note:       d.Note,
	}

	req := &serviceports.DecideAgentProposalRequest{
		ProposalID:    proposal.ID,
		Decision:      decision,
		Modifications: d.Modifications,
		Note:          d.Note,
		TenantInfo:    c.session.Tenant,
	}
	if decision != agent.DecisionRejected && proposal.Preview != nil {
		req.PreviewDigest = proposal.Preview.Digest
	}

	if _, err := bench.Decisions.DecideOwn(ctx, req, &c.session.Actor); err != nil {
		record.Error = err.Error()
		return record
	}
	if decision == agent.DecisionRejected {
		return record
	}

	if err := bench.Committer.CommitOwnApprovalNow(ctx, &serviceports.SettleApprovalRequest{
		ProposalID: proposal.ID,
		TenantInfo: c.session.Tenant,
	}, &c.session.Actor); err != nil {
		record.Error = "commit now: " + err.Error()
	}

	return record
}

func (c *Conversation) decidePlan(
	ctx context.Context,
	proposal *ProposalRecord,
	d Decide,
) *DecisionRecord {
	bench := c.session.bench
	decision := d.decision()
	if decision == agent.DecisionModified {
		decision = agent.DecisionAccepted
	}
	record := &DecisionRecord{
		PlanID:   proposal.PlanID,
		Tool:     proposal.Tool,
		Decision: decision,
		Note:     d.Note,
	}

	if _, err := bench.Plans.DecideOwn(ctx, &serviceports.DecideAgentPlanRequest{
		PlanID:     proposal.PlanID,
		Decision:   decision,
		Note:       d.Note,
		TenantInfo: c.session.Tenant,
	}, &c.session.Actor); err != nil {
		record.Error = err.Error()
		return record
	}
	if decision == agent.DecisionRejected {
		return record
	}

	if err := bench.Committer.CommitOwnApprovalNow(ctx, &serviceports.SettleApprovalRequest{
		PlanID:     proposal.PlanID,
		TenantInfo: c.session.Tenant,
	}, &c.session.Actor); err != nil {
		record.Error = "commit now: " + err.Error()
	}

	return record
}

func (c *Conversation) settleDecisions(ctx context.Context, decisions []*DecisionRecord) {
	for _, decision := range decisions {
		if decision.ProposalID.IsNil() {
			c.settlePlan(ctx, decision)
			continue
		}

		proposal, err := c.loadProposal(ctx, decision.ProposalID)
		if err != nil {
			decision.Error = joinErrors(decision.Error, "re-read: "+err.Error())
			continue
		}
		decision.StatusAfter = proposal.Status
		decision.ExecutionError = proposal.ExecutionError
		decision.Executed = proposal.ExecutedAt != nil && proposal.ExecutionError == ""
		decision.Result = proposal.ExecutionResult
	}
}

func (c *Conversation) settlePlan(ctx context.Context, decision *DecisionRecord) {
	failures := make([]string, 0, 2)
	executed := true
	for _, turn := range c.turns {
		for _, proposal := range turn.Proposals {
			if proposal.PlanID != decision.PlanID {
				continue
			}
			current, err := c.loadProposal(ctx, proposal.ID)
			if err != nil {
				failures = append(failures, proposal.Tool+": "+err.Error())
				executed = false
				continue
			}
			decision.StatusAfter = current.Status
			if current.ExecutionError != "" {
				failures = append(failures, proposal.Tool+": "+current.ExecutionError)
			}
			if current.ExecutedAt == nil || current.ExecutionError != "" {
				executed = false
			}
		}
	}
	decision.ExecutionError = strings.Join(failures, "; ")
	decision.Executed = executed
}

func (c *Conversation) lookupTurn(ctx context.Context, turnID pulid.ID) (*conversation.AssistantTurn, error) {
	req := repositories.GetAssistantTurnRequest{
		ID:         turnID,
		TenantInfo: c.session.Tenant,
		UserID:     c.session.User.ID,
	}

	var lastErr error
	for range lookupAttempts {
		turn, err := c.session.bench.Turns.Get(ctx, req)
		if err == nil {
			return turn, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lookupEvery):
		}
	}

	return nil, fmt.Errorf("read turn %s: %w", turnID, lastErr)
}

func errorSuffix(message string) string {
	if message == "" {
		return ""
	}

	return " · error: " + message
}

func joinErrors(existing, next string) string {
	if existing == "" {
		return next
	}

	return existing + "; " + next
}

func (c *Conversation) directedAgent(ctx context.Context, ref string) (pulid.ID, error) {
	definition, err := c.session.Agent(ctx, ref)
	if err != nil {
		return pulid.Nil, err
	}
	if definition.ID == c.Thread.AgentDefinitionID {
		return pulid.Nil, nil
	}

	return definition.ID, nil
}
