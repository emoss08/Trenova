package deskbench

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/deskbench/benchlive"
	"github.com/emoss08/trenova/internal/deskbench/benchnamespace"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchwatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	firstTurnWithin  = 2 * time.Minute
	followUpWithin   = 3 * time.Minute
	CleanupWithin    = 45 * time.Second
	settleAfterTurn  = 5 * time.Second
	TurnTimeout      = 15 * time.Minute
	relayGrace       = 20 * time.Second
	lookupAttempts   = 25
	lookupEvery      = 200 * time.Millisecond
	WindowLead       = time.Second
	cleanupNote      = "Closed by the desk bench at the end of its run."
	BenchTitlePrefix = "[bench] "
)

var ErrNothingToDecide = errors.New("no proposal is waiting for a decision in this conversation")

type Decide struct {
	Action        benchscenario.DecisionAction
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
	queue    *benchwatch.TurnQueue
	turns    []*benchrecord.TurnRecord
	formula  *formulaState
	palette  bool
	CodeMode codemode.Override

	NeedsText bool
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
		thread, err = s.bench.Assistant.StartThread(ctx, &assistantports.StartThreadRequest{
			AgentDefinitionID: req.Agent.ID,
			Title:             BenchTitlePrefix + strings.TrimSpace(req.Title),
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
		queue:    s.bench.watcher.Watch(thread.ID),
	}, nil
}

func (s *Session) threadRequest(threadID pulid.ID) *conversationports.GetThreadRequest {
	return &conversationports.GetThreadRequest{
		ID:         threadID,
		UserID:     s.User.ID,
		TenantInfo: s.Tenant,
	}
}

func (c *Conversation) Turns() []*benchrecord.TurnRecord {
	return c.turns
}

func (c *Conversation) Say(
	ctx context.Context,
	u *benchscenario.Utterance,
) ([]*benchrecord.TurnRecord, error) {
	ctx = c.session.Context(ctx)
	bench := c.session.bench

	surface := u.Surface
	if surface == "" {
		surface = agentpagecontext.SurfaceDesk
	}
	if c.formula != nil {
		u.Page = c.formula.page()
		surface = agentpagecontext.SurfaceAssistant
	}
	if c.palette {
		surface = agentpagecontext.SurfacePalette
	}

	attached, err := c.attach(ctx, u.Attachments)
	if err != nil {
		return nil, err
	}

	_, err = bench.Turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   c.Thread.ID,
		UserID:     c.session.User.ID,
		TenantInfo: c.session.Tenant,
		Origin:     conversationturn.AssistantTurnOriginPerson,
		Input:      u.Content,
	}, func(turn *conversationturn.AssistantTurn) (string, error) {
		start := assistantcontract.TurnStart{
			Actor:   c.session.Actor,
			Content: u.Content,
			Request: assistantcontract.AssistantTurnRequest{
				Page:                  u.Page,
				Surface:               surface,
				Mentions:              u.Mentions,
				PreferredProviderID:   benchcell.ProviderID(c.Provider),
				ProviderChosen:        c.Provider != nil,
				DirectedAgentID:       u.Directed,
				AttachmentDocumentIDs: attached,
				Awaited:               true,
				CodeMode:              c.CodeMode,
			},
		}
		var runID string
		startErr := benchnamespace.RetryLookup(ctx, func() error {
			run, err := assistantcontract.StartTurnWorkflow(ctx, bench.watcher, turn, start)
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

func (c *Conversation) Decide(
	ctx context.Context,
	d Decide,
) ([]*benchrecord.DecisionRecord, []*benchrecord.TurnRecord, error) {
	return c.decide(ctx, d, followUpWithin)
}

func (c *Conversation) decide(
	ctx context.Context,
	d Decide,
	within time.Duration,
) ([]*benchrecord.DecisionRecord, []*benchrecord.TurnRecord, error) {
	ctx = c.session.Context(ctx)

	pending, err := c.pending(ctx, d.Tool)
	if err != nil {
		return nil, nil, err
	}
	if len(pending) == 0 {
		return nil, nil, ErrNothingToDecide
	}

	decisions := make([]*benchrecord.DecisionRecord, 0, len(pending))
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
		c.session.bench.live.Linef(c.Label, "decided %s %s%s", decision.Decision, decision.Tool,
			errorSuffix(decision.Error))
	}
	turns, err := c.collect(ctx, within)
	c.settleDecisions(ctx, decisions)
	for _, decision := range decisions {
		c.session.bench.live.Linef(c.Label, "%s %s -> %s%s", decision.Decision, decision.Tool,
			decision.StatusAfter, errorSuffix(decision.ExecutionError))
	}

	return decisions, turns, err
}

type CloseOptions struct {
	LeavePending bool
	Delete       bool
}

func (c *Conversation) Close(
	ctx context.Context,
	opts CloseOptions,
) ([]*benchrecord.TurnRecord, error) {
	bench := c.session.bench
	defer bench.watcher.Forget(c.Thread.ID)

	var turns []*benchrecord.TurnRecord
	if !opts.LeavePending {
		cleanup := Decide{Action: benchscenario.DecisionReject, Note: cleanupNote}
		if _, closing, err := c.decide(ctx, cleanup, CleanupWithin); err == nil {
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

func (c *Conversation) collect(
	ctx context.Context,
	within time.Duration,
) ([]*benchrecord.TurnRecord, error) {
	records := make([]*benchrecord.TurnRecord, 0, 2)
	wait := within
	for {
		timer := time.NewTimer(wait)
		started, ok := c.queue.Next(ctx, timer.C)
		timer.Stop()
		if !ok {
			break
		}

		record := c.capture(ctx, &started)
		records = append(records, record)
		c.turns = append(c.turns, record)
		if c.formula != nil {
			c.formula.apply(record.Artifacts)
		}
		wait = settleAfterTurn
	}

	return records, ctx.Err()
}

func (c *Conversation) capture(
	ctx context.Context,
	started *benchwatch.StartedTurn,
) *benchrecord.TurnRecord {
	bench := c.session.bench
	record := &benchrecord.TurnRecord{
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

	live := bench.live.Turn(c.Label)
	live.Linef(
		"== turn %s (%s) ==",
		started.TurnID,
		stringutils.WithDefault(string(record.Origin), "Person"),
	)
	if started.Content != "" {
		live.Linef("person: %s", started.Content)
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

	runCtx, cancelRun := context.WithTimeout(ctx, TurnTimeout)
	var outcome assistantcontract.AssistantTurnResult
	runErr := started.Run.Get(runCtx, &outcome)
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	cancelRun()
	if timedOut && turn != nil {
		if stopErr := bench.Turns.Stop(ctx, turn); stopErr != nil {
			record.CaptureError = benchrecord.JoinErrors(
				record.CaptureError,
				"stop: "+stopErr.Error(),
			)
		}
	}

	select {
	case relayErr := <-relayDone:
		if relayErr != nil && !errors.Is(relayErr, context.Canceled) {
			record.CaptureError = benchrecord.JoinErrors(
				record.CaptureError,
				"relay: "+relayErr.Error(),
			)
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

	bench.Measure(ctx, c.session, record, record.StartedAt.Add(-WindowLead))

	benchlive.AnnounceTurnEnd(live, record)

	return record
}

func (c *Conversation) fill(
	ctx context.Context,
	record *benchrecord.TurnRecord,
	outcome *assistantcontract.AssistantTurnResult,
) {
	if outcome.Status != "" {
		record.Status = conversation.AssistantTurnStatus(outcome.Status)
	}
	record.Refused = outcome.Refused
	if outcome.Message != "" {
		record.Failure = benchrecord.JoinErrors(record.Failure, outcome.Message)
	}

	result := outcome.Result
	if result == nil {
		return
	}

	record.Reply = result.Reply
	record.Refused = record.Refused || result.Refused
	record.Messages = result.Messages
	record.Artifacts = result.Artifacts
	record.Proposals = make([]*benchrecord.ProposalRecord, 0, len(result.Proposals))
	for idx := range result.Proposals {
		record.Proposals = append(record.Proposals, c.describeProposal(ctx, &result.Proposals[idx]))
	}
}

func (c *Conversation) describeProposal(
	ctx context.Context,
	summary *assistantproposalports.AssistantProposal,
) *benchrecord.ProposalRecord {
	record := &benchrecord.ProposalRecord{
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

	preview, err := c.session.bench.Previews.ForProposal(
		ctx,
		&agentruntimeports.ProposalPreviewRequest{
			Proposal: proposal,
			Viewer:   &agentruntimeports.PreviewViewer{Actor: &c.session.Actor},
		},
	)
	if err != nil {
		record.Error = "preview: " + err.Error()
		return record
	}
	record.Preview = preview

	return record
}

func (c *Conversation) loadProposal(
	ctx context.Context,
	id pulid.ID,
) (*agent.AgentProposal, error) {
	return c.session.bench.Proposals.GetByID(ctx, agentproposalports.GetAgentProposalByIDRequest{
		ID:         id,
		TenantInfo: &c.session.Tenant,
	})
}

func (c *Conversation) pending(
	ctx context.Context,
	tool string,
) ([]*benchrecord.ProposalRecord, error) {
	seen := make(map[pulid.ID]struct{}, 4)
	pending := make([]*benchrecord.ProposalRecord, 0, 4)
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

func (d Decide) decision() agentdecision.DecisionType {
	if d.Action == benchscenario.DecisionReject {
		return agentdecision.DecisionRejected
	}
	if len(d.Modifications) > 0 {
		return agentdecision.DecisionModified
	}

	return agentdecision.DecisionAccepted
}

func (c *Conversation) decideProposal(
	ctx context.Context,
	proposal *benchrecord.ProposalRecord,
	d Decide,
) *benchrecord.DecisionRecord {
	bench := c.session.bench
	decision := d.decision()
	record := &benchrecord.DecisionRecord{
		ProposalID: proposal.ID,
		Tool:       proposal.Tool,
		Decision:   decision,
		Note:       d.Note,
	}

	req := &agentdecisionports.DecideAgentProposalRequest{
		ProposalID:    proposal.ID,
		Decision:      decision,
		Modifications: d.Modifications,
		Note:          d.Note,
		TenantInfo:    c.session.Tenant,
	}
	if decision != agentdecision.DecisionRejected {
		digest, err := c.proposalDigest(ctx, proposal, d.Modifications)
		if err != nil {
			record.Error = "preview: " + err.Error()
			return record
		}
		req.PreviewDigest = digest
	}

	if _, err := bench.Decisions.DecideOwn(ctx, req, &c.session.Actor); err != nil {
		record.Error = err.Error()
		return record
	}
	if decision == agentdecision.DecisionRejected {
		return record
	}

	if err := bench.Committer.CommitOwnApprovalNow(ctx, &agentapprovalports.SettleApprovalRequest{
		ProposalID: proposal.ID,
		TenantInfo: c.session.Tenant,
	}, &c.session.Actor); err != nil {
		record.Error = "commit now: " + err.Error()
	}

	return record
}

func (c *Conversation) decidePlan(
	ctx context.Context,
	proposal *benchrecord.ProposalRecord,
	d Decide,
) *benchrecord.DecisionRecord {
	bench := c.session.bench
	decision := d.decision()
	if decision == agentdecision.DecisionModified {
		decision = agentdecision.DecisionAccepted
	}
	record := &benchrecord.DecisionRecord{
		PlanID:   proposal.PlanID,
		Tool:     proposal.Tool,
		Decision: decision,
		Note:     d.Note,
	}

	planReq := &agentplanports.DecideAgentPlanRequest{
		PlanID:     proposal.PlanID,
		Decision:   decision,
		Note:       d.Note,
		TenantInfo: c.session.Tenant,
	}
	if decision != agentdecision.DecisionRejected {
		digest, err := c.planDigest(ctx, proposal.PlanID)
		if err != nil {
			record.Error = "preview: " + err.Error()
			return record
		}
		planReq.PreviewDigest = digest
	}
	if _, err := bench.Plans.DecideOwn(ctx, planReq, &c.session.Actor); err != nil {
		record.Error = err.Error()
		return record
	}
	if decision == agentdecision.DecisionRejected {
		return record
	}

	if err := bench.Committer.CommitOwnApprovalNow(ctx, &agentapprovalports.SettleApprovalRequest{
		PlanID:     proposal.PlanID,
		TenantInfo: c.session.Tenant,
	}, &c.session.Actor); err != nil {
		record.Error = "commit now: " + err.Error()
	}

	return record
}

func (c *Conversation) proposalDigest(
	ctx context.Context,
	proposal *benchrecord.ProposalRecord,
	modifications map[string]any,
) (string, error) {
	if len(modifications) == 0 && proposal.Preview != nil {
		return proposal.Preview.Digest, nil
	}
	current, err := c.loadProposal(ctx, proposal.ID)
	if err != nil {
		return "", err
	}
	preview, err := c.session.bench.Previews.ForProposal(
		ctx,
		&agentruntimeports.ProposalPreviewRequest{
			Proposal:      current,
			Modifications: modifications,
			Viewer:        &agentruntimeports.PreviewViewer{Actor: &c.session.Actor},
		},
	)
	if err != nil {
		return "", err
	}

	return preview.Digest, nil
}

func (c *Conversation) planDigest(ctx context.Context, planID pulid.ID) (string, error) {
	plan, err := c.session.bench.Plans.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: c.session.Tenant,
	})
	if err != nil {
		return "", err
	}
	preview, err := c.session.bench.Previews.ForPlan(ctx, &agentruntimeports.PlanPreviewRequest{
		Plan:   plan,
		Viewer: &agentruntimeports.PreviewViewer{Actor: &c.session.Actor},
	})
	if err != nil {
		return "", err
	}

	return preview.Digest, nil
}

func (c *Conversation) settleDecisions(
	ctx context.Context,
	decisions []*benchrecord.DecisionRecord,
) {
	for _, decision := range decisions {
		if decision.ProposalID.IsNil() {
			c.settlePlan(ctx, decision)
			continue
		}

		proposal, err := c.loadProposal(ctx, decision.ProposalID)
		if err != nil {
			decision.Error = benchrecord.JoinErrors(decision.Error, "re-read: "+err.Error())
			continue
		}
		decision.StatusAfter = proposal.Status
		decision.ExecutionError = proposal.ExecutionError
		decision.Executed = proposal.ExecutedAt != nil && proposal.ExecutionError == ""
		decision.Result = proposal.ExecutionResult
	}
}

func (c *Conversation) settlePlan(ctx context.Context, decision *benchrecord.DecisionRecord) {
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

func (c *Conversation) lookupTurn(
	ctx context.Context,
	turnID pulid.ID,
) (*conversationturn.AssistantTurn, error) {
	req := assistantturnports.GetAssistantTurnRequest{
		ID:         turnID,
		TenantInfo: c.session.Tenant,
		UserID:     c.session.User.ID,
	}

	var lastErr error
	for range lookupAttempts {
		turn, err := c.session.bench.Turns.Get(ctx, &req)
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

func (c *Conversation) DirectedAgent(ctx context.Context, ref string) (pulid.ID, error) {
	definition, err := c.session.Agent(ctx, ref)
	if err != nil {
		return pulid.Nil, err
	}
	if definition.ID == c.Thread.AgentDefinitionID {
		return pulid.Nil, nil
	}

	return definition.ID, nil
}
