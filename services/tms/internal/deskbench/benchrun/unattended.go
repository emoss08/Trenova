package benchrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/deskbench/benchcheck"
	"github.com/emoss08/trenova/internal/deskbench/benchlive"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchsql"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

const (
	runPollInterval = 2 * time.Second
	runStartWithin  = 2 * time.Minute
	runSettleWithin = time.Minute
	runCleanupNote  = "The bench case ended; nobody decided this."
)

type firedRun struct {
	definition *agentdefinition.Definition
	subjectID  pulid.ID
	run        *agent.AgentRun
	workflow   client.WorkflowRun
	startedAt  time.Time
}

func (r *Runner) runFire(
	ctx context.Context,
	session *deskbench.Session,
	cell benchcell.Cell,
	step *benchscenario.Step,
	label string,
) (*benchrecord.TurnRecord, *benchcheck.RunOutcome, error) {
	fire := step.Fire
	definition, err := session.UnattendedAgent(ctx, fire.Template())
	if err != nil {
		return nil, nil, err
	}

	unlock := r.exclusive("unattended:" + definition.ID.String())
	defer unlock()

	subjectID, err := r.fireSubject(ctx, session, fire)
	if err != nil {
		return nil, nil, err
	}

	unpin := r.recorder.Pin(definition.ID, cell.Pin())
	defer unpin()

	if err = session.ShapeForRun(ctx, definition, fire, true); err != nil {
		return nil, nil, fmt.Errorf("turn %s on for the run: %w", definition.Name, err)
	}
	defer func() {
		offCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deskbench.CleanupWithin)
		defer cancel()
		if offErr := session.ShapeForRun(offCtx, definition, fire, false); offErr != nil {
			r.live.Linef(label, "could not turn %s off again: %v", definition.Name, offErr)
		}
	}()

	fired, err := r.startFreshRun(ctx, session, definition, fire, subjectID, label)
	if err != nil {
		return nil, nil, err
	}
	r.live.Linef(label, "== run %s (%s) ==", fired.run.ID, fire.Label())

	record, outcome := r.captureRun(ctx, session, fired, label)
	if fire.StopWhileWaiting {
		r.stopWaitingRun(ctx, session, fired, record, outcome, label)
	}
	r.settleRun(ctx, session, fired, record, label)

	return record, outcome, nil
}

func (r *Runner) fireSubject(
	ctx context.Context,
	session *deskbench.Session,
	fire *benchscenario.Fire,
) (pulid.ID, error) {
	if fire.Schedule {
		return pulid.Nil, nil
	}

	values, err := r.facts(ctx, session, fire.Subject)
	if err != nil {
		return pulid.Nil, fmt.Errorf("find the run's subject: %w", err)
	}
	if len(values) != 1 {
		return pulid.Nil, fmt.Errorf("the subject query returned %d rows, not one", len(values))
	}
	id, err := pulid.Parse(strings.TrimSpace(values[0]))
	if err != nil {
		return pulid.Nil, fmt.Errorf("the subject query returned %q, not an id: %w", values[0], err)
	}

	return id, nil
}

func (r *Runner) startFreshRun(
	ctx context.Context,
	session *deskbench.Session,
	definition *agentdefinition.Definition,
	fire *benchscenario.Fire,
	subjectID pulid.ID,
	label string,
) (*firedRun, error) {
	fired, err := r.startRun(ctx, session, definition, fire, subjectID)
	if err == nil || fire.Schedule || !errors.Is(err, agentruntimeports.ErrAgentRunAlreadyOpen) {
		return fired, err
	}

	workflowID, findErr := r.leftoverWorkflow(ctx, session, definition.ID, subjectID)
	if findErr != nil {
		return nil, errors.Join(
			err,
			fmt.Errorf("find the run an earlier bench left open: %w", findErr),
		)
	}
	if workflowID == "" {
		return nil, err
	}
	r.live.Linef(label, "stopping %s, which an earlier bench left running", workflowID)
	if cancelErr := r.watcher.CancelWorkflow(ctx, workflowID, ""); cancelErr != nil {
		return nil, errors.Join(
			err,
			fmt.Errorf("stop the run an earlier bench left open: %w", cancelErr),
		)
	}

	deadline := time.Now().Add(runSettleWithin)
	ticker := time.NewTicker(runPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		fired, err = r.startRun(ctx, session, definition, fire, subjectID)
		if err == nil || !errors.Is(err, agentruntimeports.ErrAgentRunAlreadyOpen) ||
			time.Now().After(deadline) {
			return fired, err
		}
	}
}

func (r *Runner) leftoverWorkflow(
	ctx context.Context,
	session *deskbench.Session,
	definitionID, subjectID pulid.ID,
) (string, error) {
	values, err := r.facts(
		ctx,
		session,
		"SELECT workflow_id FROM agent_runs WHERE agent_definition_id = "+benchsql.Quote(
			definitionID,
		)+
			" AND subject_id = "+benchsql.Quote(
			subjectID,
		)+
			" AND workflow_id IS NOT NULL ORDER BY created_at DESC LIMIT 1",
	)
	if err != nil || len(values) == 0 {
		return "", err
	}

	return values[0], nil
}

func (r *Runner) startRun(
	ctx context.Context,
	session *deskbench.Session,
	definition *agentdefinition.Definition,
	fire *benchscenario.Fire,
	subjectID pulid.ID,
) (*firedRun, error) {
	req := &agentruntimeports.StartAgentRunForDefinitionRequest{
		DefinitionID: definition.ID,
		TenantInfo:   session.Tenant,
	}
	var actor *permissionports.RequestActor
	if fire.Schedule {
		req.Trigger = definition.TriggerMode.RunTrigger()
		req.Slot = time.Now().UnixNano()
		actor = &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    definition.ID,
			OrganizationID: session.Tenant.OrgID,
			BusinessUnitID: session.Tenant.BuID,
		}
	} else {
		req.Trigger = agent.RunTriggerEvent
		req.EventKind = fire.Event
		req.SubjectType = fire.Event.SubjectType()
		req.SubjectID = subjectID
	}

	started := time.Now()
	run, err := r.runs.StartForDefinition(session.Context(ctx), req, actor)
	if err != nil {
		return nil, fmt.Errorf("start the run: %w", err)
	}

	workflow, ok := r.watcher.Workflow(run.ID)
	if !ok {
		return nil, fmt.Errorf("the run %s started without its workflow being seen", run.ID)
	}

	return &firedRun{
		definition: definition,
		subjectID:  subjectID,
		run:        run,
		workflow:   workflow,
		startedAt:  started,
	}, nil
}

func loopFinished(status agent.RunStatus) bool {
	switch status {
	case agent.RunStatusAwaitingDecision, agent.RunStatusCompleted,
		agent.RunStatusShadowCompleted, agent.RunStatusFailed:
		return true
	case agent.RunStatusPending, agent.RunStatusGatheringContext, agent.RunStatusDiagnosing:
		return false
	default:
		return false
	}
}

func (r *Runner) captureRun(
	ctx context.Context,
	session *deskbench.Session,
	fired *firedRun,
	label string,
) (*benchrecord.TurnRecord, *benchcheck.RunOutcome) {
	record := &benchrecord.TurnRecord{
		RunID:     fired.run.ID,
		TraceID:   fired.run.TraceID,
		Origin:    conversationturn.AssistantTurnOrigin(fired.run.Trigger),
		StartedAt: fired.startedAt,
		Status:    conversation.AssistantTurnStatusRunning,
	}
	outcome := &benchcheck.RunOutcome{ID: fired.run.ID, Subject: fired.subjectID}

	run, err := r.awaitLoop(ctx, session, fired)
	record.FinishedAt = time.Now()
	if err != nil {
		record.CaptureError = err.Error()
	}
	if run != nil {
		outcome.Status = run.Status
		record.Reply = run.Summary
		record.Failure = run.ErrorMessage
		record.Status = turnStatusOf(run.Status)
		record.Input = string(run.Trigger)
		transcript, transcriptErr := r.loadTranscript(ctx, session, fired.run.ID)
		if transcriptErr != nil {
			record.CaptureError = benchrecord.JoinErrors(
				record.CaptureError,
				"transcript: "+transcriptErr.Error(),
			)
		}
		if transcript != nil {
			record.Tools = transcriptTools(transcript)
		}
	}

	record.Proposals, err = r.runProposals(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError, "proposals: "+err.Error())
	}
	outcome.Flagged, err = r.runFlags(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError, "flags: "+err.Error())
	}

	r.measure(ctx, session, record, fired.startedAt.Add(-deskbench.WindowLead))
	live := r.live.Turn(label)
	benchlive.AnnounceTurnEnd(live, record)

	return record, outcome
}

func turnStatusOf(status agent.RunStatus) conversation.AssistantTurnStatus {
	switch status {
	case agent.RunStatusFailed:
		return conversation.AssistantTurnStatusFailed
	case agent.RunStatusAwaitingDecision, agent.RunStatusCompleted, agent.RunStatusShadowCompleted:
		return conversation.AssistantTurnStatusCompleted
	case agent.RunStatusPending, agent.RunStatusGatheringContext, agent.RunStatusDiagnosing:
		return conversation.AssistantTurnStatusRunning
	default:
		return conversation.AssistantTurnStatusRunning
	}
}

func (r *Runner) awaitLoop(
	ctx context.Context,
	session *deskbench.Session,
	fired *firedRun,
) (*agent.AgentRun, error) {
	deadline := time.Now().Add(deskbench.TurnTimeout + runStartWithin)
	ticker := time.NewTicker(runPollInterval)
	defer ticker.Stop()

	var latest *agent.AgentRun
	for {
		run, err := r.loadRun(ctx, session, fired.run.ID)
		if err == nil {
			latest = run
			if loopFinished(run.Status) {
				return run, nil
			}
		}
		if time.Now().After(deadline) {
			return latest, fmt.Errorf("the run did not finish its loop within %s",
				deskbench.TurnTimeout+runStartWithin)
		}
		select {
		case <-ctx.Done():
			return latest, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Runner) loadRun(
	ctx context.Context,
	session *deskbench.Session,
	runID pulid.ID,
) (*agent.AgentRun, error) {
	return r.runRecords.GetByID(session.Context(ctx), agentruntimeports.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &session.Tenant,
	})
}

func (r *Runner) loadTranscript(
	ctx context.Context,
	session *deskbench.Session,
	runID pulid.ID,
) (*agent.RunTranscript, error) {
	runs, err := r.runRecords.ListTranscriptsByIDs(
		session.Context(ctx),
		agentruntimeports.ListAgentRunsByIDsRequest{
			IDs:        []pulid.ID{runID},
			TenantInfo: session.Tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, errors.New("the run is not recorded yet")
	}

	return runs[0].Transcript, nil
}

func transcriptTools(transcript *agent.RunTranscript) []*benchrecord.ToolCallRecord {
	results := make(map[string]*agent.TranscriptMessage, len(transcript.Messages))
	for idx := range transcript.Messages {
		message := &transcript.Messages[idx]
		if message.ToolCallID != "" {
			results[message.DelegateCallID+"/"+message.ToolCallID] = message
		}
	}

	tools := make([]*benchrecord.ToolCallRecord, 0, len(results))
	for idx := range transcript.Messages {
		message := &transcript.Messages[idx]
		for _, call := range message.ToolCalls {
			record := &benchrecord.ToolCallRecord{
				CallID:         call.ID,
				Name:           call.Name,
				Arguments:      call.Arguments,
				AgentID:        message.AgentDefinitionID,
				DelegateCallID: message.DelegateCallID,
				StartedAt:      time.Unix(message.CreatedAt, 0),
			}
			if result, ok := results[message.DelegateCallID+"/"+call.ID]; ok {
				record.Finished = true
				record.FinishedAt = time.Unix(result.CreatedAt, 0)
				record.Failed = result.ToolFailed
				record.Verdict = result.ToolVerdict
				record.Summary = result.ToolSummary
				record.Result = result.Content
				record.Proposed = result.ToolVerdict == benchrecord.VerdictProposed
			}
			tools = append(tools, record)
		}
	}

	return tools
}

func (r *Runner) runProposals(
	ctx context.Context,
	session *deskbench.Session,
	runID pulid.ID,
) ([]*benchrecord.ProposalRecord, error) {
	values, err := r.facts(
		ctx,
		session,
		"SELECT id FROM agent_proposals WHERE run_id = "+benchsql.Quote(
			runID,
		)+" ORDER BY created_at, id",
	)
	if err != nil {
		return nil, err
	}

	ids := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		if id, parseErr := pulid.Parse(value); parseErr == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	proposals, err := r.proposals.ListByIDs(
		session.Context(ctx),
		agentproposalports.ListAgentProposalsByIDsRequest{
			IDs:        ids,
			TenantInfo: session.Tenant,
		},
	)
	if err != nil {
		return nil, err
	}

	records := make([]*benchrecord.ProposalRecord, 0, len(proposals))
	for _, proposal := range proposals {
		records = append(records, r.proposalRecord(ctx, session, proposal))
	}

	return records, nil
}

func (r *Runner) proposalRecord(
	ctx context.Context,
	session *deskbench.Session,
	proposal *agent.AgentProposal,
) *benchrecord.ProposalRecord {
	record := &benchrecord.ProposalRecord{
		ID:        proposal.ID,
		Tool:      proposal.ToolName,
		Arguments: proposal.ToolParams,
		Rationale: proposal.Rationale,
		Tier:      proposal.AutonomyTier,
		Status:    proposal.Status,
		HeldBy:    proposal.HeldBy,
		Simulated: proposal.Simulation,
		Executed:  proposal.ExecutionResult,
		Error:     proposal.ExecutionError,
		PlanStep:  proposal.PlanStep,
	}
	if proposal.PlanID != nil {
		record.PlanID = *proposal.PlanID
	}
	if proposal.Status != agent.ProposalStatusPending {
		return record
	}

	preview, err := r.previews.ForProposal(
		session.Context(ctx),
		&agentruntimeports.ProposalPreviewRequest{
			Proposal: proposal,
			Viewer:   &agentruntimeports.PreviewViewer{Actor: &session.Actor},
		},
	)
	if err != nil {
		record.Error = benchrecord.JoinErrors(record.Error, "preview: "+err.Error())
		return record
	}
	record.Preview = preview

	return record
}

func (r *Runner) runFlags(
	ctx context.Context,
	session *deskbench.Session,
	runID pulid.ID,
) (int, error) {
	values, err := r.facts(ctx, session,
		"SELECT count(*) FROM agent_exceptions WHERE run_id = "+benchsql.Quote(runID))
	if err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, nil
	}

	var count int
	if _, err = fmt.Sscan(values[0], &count); err != nil {
		return 0, fmt.Errorf("count the run's flags: %w", err)
	}

	return count, nil
}

func (r *Runner) stopWaitingRun(
	ctx context.Context,
	session *deskbench.Session,
	fired *firedRun,
	record *benchrecord.TurnRecord,
	outcome *benchcheck.RunOutcome,
	label string,
) {
	outcome.Stopping = true
	if outcome.Status != agent.RunStatusAwaitingDecision {
		return
	}

	r.live.Linef(label, "stopping run %s while it waits for a decision", fired.run.ID)
	if err := r.watcher.CancelWorkflow(ctx, fired.run.WorkflowID, ""); err != nil {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError, "stop: "+err.Error())
		return
	}
	outcome.Stopped = true

	waitCtx, cancel := context.WithTimeout(ctx, runSettleWithin)
	defer cancel()
	if err := fired.workflow.Get(waitCtx, nil); err != nil && waitCtx.Err() != nil {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError,
			fmt.Sprintf("the stopped run did not close within %s", runSettleWithin))
	}

	run, err := r.loadRun(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = benchrecord.JoinErrors(
			record.CaptureError,
			"stopped run: "+err.Error(),
		)
		return
	}
	outcome.Status = run.Status
	record.Status = turnStatusOf(run.Status)
	record.Failure = run.ErrorMessage

	record.Proposals, err = r.runProposals(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError, "proposals: "+err.Error())
	}
}

func (r *Runner) settleRun(
	ctx context.Context,
	session *deskbench.Session,
	fired *firedRun,
	record *benchrecord.TurnRecord,
	label string,
) {
	settleCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		deskbench.CleanupWithin+runSettleWithin,
	)
	defer cancel()

	for _, proposal := range record.Proposals {
		if proposal.Status != agent.ProposalStatusPending {
			continue
		}
		reject := &agentdecisionports.DecideAgentProposalRequest{
			ProposalID: proposal.ID,
			Decision:   agentdecision.DecisionRejected,
			Note:       runCleanupNote,
			TenantInfo: session.Tenant,
		}
		_, err := r.decisions.DecideWithOutcome(session.Context(settleCtx), reject, &session.Actor)
		if err != nil {
			r.live.Linef(label, "could not reject %s (%s): %v", proposal.Tool, proposal.ID, err)
		}
	}

	waitCtx, stop := context.WithTimeout(settleCtx, runSettleWithin)
	defer stop()
	if err := fired.workflow.Get(waitCtx, nil); err != nil && waitCtx.Err() != nil {
		cancelErr := r.watcher.CancelWorkflow(settleCtx, fired.run.WorkflowID, "")
		if cancelErr != nil {
			r.live.Linef(label, "could not stop run %s: %v", fired.run.ID, cancelErr)
		}
	}
}
