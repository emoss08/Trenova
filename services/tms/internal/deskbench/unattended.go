package deskbench

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

const (
	unattendedSuffix   = " (unattended)"
	unattendedCron     = "0 3 1 1 *"
	runPollInterval    = 2 * time.Second
	runStartWithin     = 2 * time.Minute
	runSettleWithin    = time.Minute
	runCleanupNote     = "The bench case ended; nobody decided this."
	placeholderRun     = "{{run}}"
	placeholderSubject = "{{subject}}"
)

var unattendedContext = []agentdefinition.ContextProvider{
	agentdefinition.ContextOrganization,
	agentdefinition.ContextClock,
	agentdefinition.ContextTools,
	agentdefinition.ContextMemory,
}

type Fire struct {
	Agent            string                        `yaml:"agent"            json:"agent"`
	Event            agent.EventKind               `yaml:"event"            json:"event,omitempty"`
	Subject          string                        `yaml:"subject"          json:"subject,omitempty"`
	Schedule         bool                          `yaml:"schedule"         json:"schedule,omitempty"`
	Ceiling          agent.AutonomyTier            `yaml:"ceiling"          json:"ceiling,omitempty"`
	ToolTiers        map[string]agent.AutonomyTier `yaml:"toolTiers"        json:"toolTiers,omitempty"`
	StopWhileWaiting bool                          `yaml:"stopWhileWaiting" json:"stopWhileWaiting,omitempty"`
}

func (f *Fire) Template() agentdefinition.Template {
	return agentdefinition.Template(strings.TrimSpace(f.Agent))
}

func (f *Fire) Label() string {
	if f.Schedule {
		return "scheduled run of " + f.Agent
	}

	return string(f.Event) + " for " + f.Agent
}

func (f *Fire) validate(where string) error {
	template := f.Template()
	if !template.IsValid() {
		return fmt.Errorf("%s: fire.agent %q is not an agent template", where, f.Agent)
	}

	switch template.StarterTrigger() {
	case agentdefinition.TriggerEvent:
		if f.Schedule {
			return fmt.Errorf("%s: %s is started by events, not a schedule", where, template)
		}
		if !slices.Contains(template.StarterEvents(), f.Event) {
			return fmt.Errorf("%s: %s does not answer the event %q", where, template, f.Event)
		}
		if strings.TrimSpace(f.Subject) == "" {
			return fmt.Errorf("%s: an event run names its subject with a SELECT", where)
		}
		fact := Fact{SQL: f.Subject}
		if err := fact.validate(); err != nil {
			return fmt.Errorf("%s: subject: %w", where, err)
		}
	case agentdefinition.TriggerScheduled, agentdefinition.TriggerContinuous:
		if !f.Schedule || f.Event != "" || f.Subject != "" {
			return fmt.Errorf("%s: %s runs on a schedule; give schedule: true and no event", where,
				template)
		}
	case agentdefinition.TriggerChat:
		return fmt.Errorf("%s: %s is a chat agent; fire starts event and scheduled runs", where,
			template)
	default:
		return fmt.Errorf("%s: %s has no trigger the bench can fire", where, template)
	}

	return f.validateShape(where, template)
}

func (f *Fire) validateShape(where string, template agentdefinition.Template) error {
	if f.Ceiling != "" && !f.Ceiling.IsValid() {
		return fmt.Errorf("%s: fire.ceiling %q is not an autonomy tier", where, f.Ceiling)
	}
	held := template.StarterTools()
	for tool, tier := range f.ToolTiers {
		if !tier.IsValid() {
			return fmt.Errorf("%s: fire.toolTiers.%s %q is not an autonomy tier", where, tool, tier)
		}
		if !agentdefinition.IsCoreTool(tool) && !slices.Contains(held, tool) {
			return fmt.Errorf("%s: fire.toolTiers names %s, which %s does not hold", where, tool,
				template)
		}
	}

	return nil
}

func (f *Fire) ceiling(template agentdefinition.Template) agent.AutonomyTier {
	if f.Ceiling != "" {
		return f.Ceiling
	}

	return template.StarterCeiling()
}

func UnattendedTemplates() []agentdefinition.Template {
	all := []agentdefinition.Template{
		agentdefinition.TemplateBillingException,
		agentdefinition.TemplateShipmentIntake,
		agentdefinition.TemplateCashApplication,
		agentdefinition.TemplateDetentionDesk,
		agentdefinition.TemplateCredentialDesk,
		agentdefinition.TemplateCustomerUpdateDesk,
		agentdefinition.TemplateCarrierRiskDesk,
		agentdefinition.TemplateIntakeDesk,
		agentdefinition.TemplateLoadEntryCheck,
		agentdefinition.TemplateServiceFailureDesk,
		agentdefinition.TemplateInsightAnalyst,
		agentdefinition.TemplateEDIDesk,
		agentdefinition.TemplateBooksKeeper,
		agentdefinition.TemplateFactoringClerk,
		agentdefinition.TemplateDispatchAssignment,
		agentdefinition.TemplateLoadMonitor,
	}

	return slices.DeleteFunc(all, func(template agentdefinition.Template) bool {
		trigger := template.StarterTrigger()
		return trigger != agentdefinition.TriggerEvent &&
			trigger != agentdefinition.TriggerScheduled
	})
}

func UnattendedAgentName(template agentdefinition.Template) string {
	return BenchAgentPrefix + template.Label() + unattendedSuffix
}

func unattendedShape(template agentdefinition.Template) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		Name:              UnattendedAgentName(template),
		Description:       template.Description(),
		Template:          template,
		Instructions:      template.StarterInstructions(),
		ToolNames:         agentdefinition.WithoutCoreTools(template.StarterTools()),
		AutonomyCeiling:   template.StarterCeiling(),
		DataAccessCeiling: template.StarterDataAccess(),
		TriggerMode:       template.StarterTrigger(),
		EventKinds:        template.StarterEvents(),
		OutputMode:        agentdefinition.OutputReport,
		ContextProviders:  slices.Clone(unattendedContext),
		LearningOff:       true,
	}
	if definition.TriggerMode == agentdefinition.TriggerScheduled {
		definition.CronExpression = unattendedCron
	}

	return definition
}

func unattendedMatches(existing, want *agentdefinition.Definition) bool {
	held := slices.Clone(existing.ToolNames)
	wanted := slices.Clone(want.ToolNames)
	slices.Sort(held)
	slices.Sort(wanted)

	return !existing.Enabled &&
		!existing.ShadowMode &&
		existing.LearningOff &&
		existing.Template == want.Template &&
		existing.Description == want.Description &&
		existing.Instructions == want.Instructions &&
		existing.AutonomyCeiling == want.AutonomyCeiling &&
		existing.DataAccessCeiling == want.DataAccessCeiling &&
		existing.TriggerMode == want.TriggerMode &&
		existing.CronExpression == want.CronExpression &&
		slices.Equal(existing.EventKinds, want.EventKinds) &&
		existing.OutputMode == want.OutputMode &&
		len(existing.ToolTiers) == 0 &&
		slices.Equal(held, wanted)
}

func (s *Session) allAgents(ctx context.Context) ([]*agentdefinition.Definition, error) {
	result, err := s.bench.Definitions.List(
		s.Context(ctx),
		&agentdefinitionports.ListAgentDefinitionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: s.Tenant,
				Pagination: pagination.Info{Limit: pagination.MaxLimit},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	return result.Items, nil
}

func (s *Session) unattendedAgent(
	ctx context.Context,
	template agentdefinition.Template,
) (*agentdefinition.Definition, error) {
	agents, err := s.allAgents(ctx)
	if err != nil {
		return nil, err
	}
	name := UnattendedAgentName(template)
	for _, definition := range agents {
		if definition.Name == name {
			return definition, nil
		}
	}

	return nil, fmt.Errorf("the bench has no %s agent; run `trenova desk run` to sync it", name)
}

func (s *Session) SyncUnattendedAgents(ctx context.Context) (*AgentSync, error) {
	ctx = s.Context(ctx)
	outcome := &AgentSync{}
	agents, err := s.allAgents(ctx)
	if err != nil {
		return outcome, err
	}
	byName := make(map[string]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byName[definition.Name] = definition
	}

	for _, template := range UnattendedTemplates() {
		want := unattendedShape(template)
		existing := byName[want.Name]
		switch {
		case existing == nil:
			err = s.createUnattended(ctx, want)
			outcome.Created = append(outcome.Created, want.Name)
		case unattendedMatches(existing, want):
			outcome.Unchanged = append(outcome.Unchanged, want.Name)
		default:
			err = s.reshapeUnattended(ctx, existing, want)
			outcome.Updated = append(outcome.Updated, want.Name)
		}
		if err != nil {
			return outcome, fmt.Errorf("sync %s: %w", want.Name, err)
		}
	}

	return outcome, nil
}

func (s *Session) createUnattended(ctx context.Context, want *agentdefinition.Definition) error {
	learningOff := true
	_, err := s.bench.AgentDefinitions.Create(ctx, &agentdefinitionports.SaveAgentDefinitionRequest{
		Name:              want.Name,
		Description:       want.Description,
		Template:          want.Template,
		Instructions:      want.Instructions,
		ToolNames:         want.ToolNames,
		AutonomyCeiling:   want.AutonomyCeiling,
		DataAccessCeiling: want.DataAccessCeiling,
		Enabled:           false,
		TriggerMode:       want.TriggerMode,
		CronExpression:    want.CronExpression,
		EventKinds:        want.EventKinds,
		ContextProviders:  want.ContextProviders,
		OutputMode:        want.OutputMode,
		LearningOff:       &learningOff,
		TenantInfo:        s.Tenant,
	}, &s.Actor)

	return err
}

func (s *Session) reshapeUnattended(
	ctx context.Context,
	existing, want *agentdefinition.Definition,
) error {
	_, err := s.bench.AgentDefinitions.Patch(ctx, &agentdefinitionports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    agentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.Description = want.Description
			definition.Template = want.Template
			definition.Instructions = want.Instructions
			definition.ToolNames = want.ToolNames
			definition.AutonomyCeiling = want.AutonomyCeiling
			definition.DataAccessCeiling = want.DataAccessCeiling
			definition.TriggerMode = want.TriggerMode
			definition.CronExpression = want.CronExpression
			definition.EventKinds = want.EventKinds
			definition.OutputMode = want.OutputMode
			definition.ContextProviders = want.ContextProviders
			definition.ToolTiers = nil
			definition.LearningOff = true
			definition.ShadowMode = false
			definition.Enabled = false

			return nil
		},
	}, &s.Actor)

	return err
}

func (s *Session) shapeForRun(
	ctx context.Context,
	definition *agentdefinition.Definition,
	fire *Fire,
	enabled bool,
) error {
	ceiling := definition.Template.StarterCeiling()
	var tiers map[string]agent.AutonomyTier
	if enabled {
		ceiling = fire.ceiling(definition.Template)
		tiers = maps.Clone(fire.ToolTiers)
	}

	_, err := s.bench.AgentDefinitions.Patch(
		s.Context(ctx),
		&agentdefinitionports.PatchAgentDefinitionRequest{
			ID:         definition.ID,
			TenantInfo: s.Tenant,
			Comment:    agentSyncComment,
			Edit: func(edited *agentdefinition.Definition) error {
				edited.Enabled = enabled
				edited.AutonomyCeiling = ceiling
				edited.ToolTiers = tiers
				return nil
			},
		},
		&s.Actor,
	)

	return err
}

type firedRun struct {
	definition *agentdefinition.Definition
	subjectID  pulid.ID
	run        *agent.AgentRun
	workflow   client.WorkflowRun
	startedAt  time.Time
}

type runOutcome struct {
	ID       pulid.ID
	Subject  pulid.ID
	Status   agent.RunStatus
	Flagged  int
	Stopping bool
	Stopped  bool
}

func (b *Bench) runFire(
	ctx context.Context,
	session *Session,
	cell Cell,
	step *Step,
	label string,
) (*TurnRecord, *runOutcome, error) {
	fire := step.Fire
	definition, err := session.unattendedAgent(ctx, fire.Template())
	if err != nil {
		return nil, nil, err
	}

	unlock := b.exclusive("unattended:" + definition.ID.String())
	defer unlock()

	subjectID, err := b.fireSubject(ctx, session, fire)
	if err != nil {
		return nil, nil, err
	}

	unpin := b.recorder.Pin(definition.ID, cell.pin())
	defer unpin()

	if err = session.shapeForRun(ctx, definition, fire, true); err != nil {
		return nil, nil, fmt.Errorf("turn %s on for the run: %w", definition.Name, err)
	}
	defer func() {
		offCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupWithin)
		defer cancel()
		if offErr := session.shapeForRun(offCtx, definition, fire, false); offErr != nil {
			b.live.Line(label, "could not turn %s off again: %v", definition.Name, offErr)
		}
	}()

	fired, err := b.startFreshRun(ctx, session, definition, fire, subjectID, label)
	if err != nil {
		return nil, nil, err
	}
	b.live.Line(label, "== run %s (%s) ==", fired.run.ID, fire.Label())

	record, outcome := b.captureRun(ctx, session, fired, label)
	if fire.StopWhileWaiting {
		b.stopWaitingRun(ctx, session, fired, record, outcome, label)
	}
	b.settleRun(ctx, session, fired, record, label)

	return record, outcome, nil
}

func (b *Bench) fireSubject(ctx context.Context, session *Session, fire *Fire) (pulid.ID, error) {
	if fire.Schedule {
		return pulid.Nil, nil
	}

	values, err := b.FactValues(ctx, session, fire.Subject)
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

func (b *Bench) startFreshRun(
	ctx context.Context,
	session *Session,
	definition *agentdefinition.Definition,
	fire *Fire,
	subjectID pulid.ID,
	label string,
) (*firedRun, error) {
	fired, err := b.startRun(ctx, session, definition, fire, subjectID)
	if err == nil || fire.Schedule || !errors.Is(err, agentports.ErrAgentRunAlreadyOpen) {
		return fired, err
	}

	workflowID, findErr := b.leftoverWorkflow(ctx, session, definition.ID, subjectID)
	if findErr != nil {
		return nil, errors.Join(
			err,
			fmt.Errorf("find the run an earlier bench left open: %w", findErr),
		)
	}
	if workflowID == "" {
		return nil, err
	}
	b.live.Line(label, "stopping %s, which an earlier bench left running", workflowID)
	if cancelErr := b.watcher.CancelWorkflow(ctx, workflowID, ""); cancelErr != nil {
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
		fired, err = b.startRun(ctx, session, definition, fire, subjectID)
		if err == nil || !errors.Is(err, agentports.ErrAgentRunAlreadyOpen) ||
			time.Now().After(deadline) {
			return fired, err
		}
	}
}

func (b *Bench) leftoverWorkflow(
	ctx context.Context,
	session *Session,
	definitionID, subjectID pulid.ID,
) (string, error) {
	values, err := b.FactValues(ctx, session,
		"SELECT workflow_id FROM agent_runs WHERE agent_definition_id = "+runKey(definitionID)+
			" AND subject_id = "+runKey(subjectID)+
			" AND workflow_id IS NOT NULL ORDER BY created_at DESC LIMIT 1")
	if err != nil || len(values) == 0 {
		return "", err
	}

	return values[0], nil
}

func (b *Bench) startRun(
	ctx context.Context,
	session *Session,
	definition *agentdefinition.Definition,
	fire *Fire,
	subjectID pulid.ID,
) (*firedRun, error) {
	req := &agentports.StartAgentRunForDefinitionRequest{
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
	run, err := b.Runs.StartForDefinition(session.Context(ctx), req, actor)
	if err != nil {
		return nil, fmt.Errorf("start the run: %w", err)
	}

	workflow, ok := b.watcher.run(run.ID)
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

func (b *Bench) captureRun(
	ctx context.Context,
	session *Session,
	fired *firedRun,
	label string,
) (*TurnRecord, *runOutcome) {
	record := &TurnRecord{
		RunID:     fired.run.ID,
		TraceID:   fired.run.TraceID,
		Origin:    conversation.AssistantTurnOrigin(fired.run.Trigger),
		StartedAt: fired.startedAt,
		Status:    conversation.AssistantTurnStatusRunning,
	}
	outcome := &runOutcome{ID: fired.run.ID, Subject: fired.subjectID}

	run, err := b.awaitLoop(ctx, session, fired)
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
		transcript, transcriptErr := b.loadTranscript(ctx, session, fired.run.ID)
		if transcriptErr != nil {
			record.CaptureError = joinErrors(
				record.CaptureError,
				"transcript: "+transcriptErr.Error(),
			)
		}
		if transcript != nil {
			record.Tools = transcriptTools(transcript)
		}
	}

	record.Proposals, err = b.runProposals(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = joinErrors(record.CaptureError, "proposals: "+err.Error())
	}
	outcome.Flagged, err = b.runFlags(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = joinErrors(record.CaptureError, "flags: "+err.Error())
	}

	b.measure(ctx, session, record, fired.startedAt.Add(-windowLead))
	live := b.live.turn(label)
	announceTurnEnd(live, record)

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

func (b *Bench) awaitLoop(
	ctx context.Context,
	session *Session,
	fired *firedRun,
) (*agent.AgentRun, error) {
	deadline := time.Now().Add(turnTimeout + runStartWithin)
	ticker := time.NewTicker(runPollInterval)
	defer ticker.Stop()

	var latest *agent.AgentRun
	for {
		run, err := b.loadRun(ctx, session, fired.run.ID)
		if err == nil {
			latest = run
			if loopFinished(run.Status) {
				return run, nil
			}
		}
		if time.Now().After(deadline) {
			return latest, fmt.Errorf("the run did not finish its loop within %s",
				turnTimeout+runStartWithin)
		}
		select {
		case <-ctx.Done():
			return latest, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (b *Bench) loadRun(
	ctx context.Context,
	session *Session,
	runID pulid.ID,
) (*agent.AgentRun, error) {
	return b.RunRecords.GetByID(session.Context(ctx), agentports.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &session.Tenant,
	})
}

func (b *Bench) loadTranscript(
	ctx context.Context,
	session *Session,
	runID pulid.ID,
) (*agent.RunTranscript, error) {
	runs, err := b.RunRecords.ListTranscriptsByIDs(
		session.Context(ctx),
		agentports.ListAgentRunsByIDsRequest{
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

func transcriptTools(transcript *agent.RunTranscript) []*ToolCallRecord {
	results := make(map[string]*agent.TranscriptMessage, len(transcript.Messages))
	for idx := range transcript.Messages {
		message := &transcript.Messages[idx]
		if message.ToolCallID != "" {
			results[message.DelegateCallID+"/"+message.ToolCallID] = message
		}
	}

	tools := make([]*ToolCallRecord, 0, len(results))
	for idx := range transcript.Messages {
		message := &transcript.Messages[idx]
		for _, call := range message.ToolCalls {
			record := &ToolCallRecord{
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
				record.Proposed = result.ToolVerdict == verdictProposed
			}
			tools = append(tools, record)
		}
	}

	return tools
}

func runKey(runID pulid.ID) string {
	return "'" + strings.ReplaceAll(runID.String(), "'", "''") + "'"
}

func (b *Bench) runProposals(
	ctx context.Context,
	session *Session,
	runID pulid.ID,
) ([]*ProposalRecord, error) {
	values, err := b.FactValues(ctx, session,
		"SELECT id FROM agent_proposals WHERE run_id = "+runKey(runID)+" ORDER BY created_at, id")
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

	proposals, err := b.Proposals.ListByIDs(
		session.Context(ctx),
		agentports.ListAgentProposalsByIDsRequest{
			IDs:        ids,
			TenantInfo: session.Tenant,
		},
	)
	if err != nil {
		return nil, err
	}

	records := make([]*ProposalRecord, 0, len(proposals))
	for _, proposal := range proposals {
		records = append(records, b.proposalRecord(ctx, session, proposal))
	}

	return records, nil
}

func (b *Bench) proposalRecord(
	ctx context.Context,
	session *Session,
	proposal *agent.AgentProposal,
) *ProposalRecord {
	record := &ProposalRecord{
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

	preview, err := b.Previews.ForProposal(session.Context(ctx), &agentports.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   &agentports.PreviewViewer{Actor: &session.Actor},
	})
	if err != nil {
		record.Error = joinErrors(record.Error, "preview: "+err.Error())
		return record
	}
	record.Preview = preview

	return record
}

func (b *Bench) runFlags(ctx context.Context, session *Session, runID pulid.ID) (int, error) {
	values, err := b.FactValues(ctx, session,
		"SELECT count(*) FROM agent_exceptions WHERE run_id = "+runKey(runID))
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

func (b *Bench) stopWaitingRun(
	ctx context.Context,
	session *Session,
	fired *firedRun,
	record *TurnRecord,
	outcome *runOutcome,
	label string,
) {
	outcome.Stopping = true
	if outcome.Status != agent.RunStatusAwaitingDecision {
		return
	}

	b.live.Line(label, "stopping run %s while it waits for a decision", fired.run.ID)
	if err := b.watcher.CancelWorkflow(ctx, fired.run.WorkflowID, ""); err != nil {
		record.CaptureError = joinErrors(record.CaptureError, "stop: "+err.Error())
		return
	}
	outcome.Stopped = true

	waitCtx, cancel := context.WithTimeout(ctx, runSettleWithin)
	defer cancel()
	if err := fired.workflow.Get(waitCtx, nil); err != nil && waitCtx.Err() != nil {
		record.CaptureError = joinErrors(record.CaptureError,
			fmt.Sprintf("the stopped run did not close within %s", runSettleWithin))
	}

	run, err := b.loadRun(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = joinErrors(record.CaptureError, "stopped run: "+err.Error())
		return
	}
	outcome.Status = run.Status
	record.Status = turnStatusOf(run.Status)
	record.Failure = run.ErrorMessage

	record.Proposals, err = b.runProposals(ctx, session, fired.run.ID)
	if err != nil {
		record.CaptureError = joinErrors(record.CaptureError, "proposals: "+err.Error())
	}
}

func (b *Bench) settleRun(
	ctx context.Context,
	session *Session,
	fired *firedRun,
	record *TurnRecord,
	label string,
) {
	settleCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		cleanupWithin+runSettleWithin,
	)
	defer cancel()

	for _, proposal := range record.Proposals {
		if proposal.Status != agent.ProposalStatusPending {
			continue
		}
		reject := &agentports.DecideAgentProposalRequest{
			ProposalID: proposal.ID,
			Decision:   agent.DecisionRejected,
			Note:       runCleanupNote,
			TenantInfo: session.Tenant,
		}
		_, err := b.Decisions.DecideWithOutcome(session.Context(settleCtx), reject, &session.Actor)
		if err != nil {
			b.live.Line(label, "could not reject %s (%s): %v", proposal.Tool, proposal.ID, err)
		}
	}

	waitCtx, stop := context.WithTimeout(settleCtx, runSettleWithin)
	defer stop()
	if err := fired.workflow.Get(waitCtx, nil); err != nil && waitCtx.Err() != nil {
		cancelErr := b.watcher.CancelWorkflow(settleCtx, fired.run.WorkflowID, "")
		if cancelErr != nil {
			b.live.Line(label, "could not stop run %s: %v", fired.run.ID, cancelErr)
		}
	}
}

func (o *runOutcome) substitute(statement string) string {
	if o == nil {
		return statement
	}
	statement = strings.ReplaceAll(statement, placeholderRun, runKey(o.ID))

	return strings.ReplaceAll(statement, placeholderSubject, runKey(o.Subject))
}

func runChecks(expect *Expect, outcome *runOutcome) []CheckResult {
	checks := make([]CheckResult, 0, 3)
	if outcome == nil {
		return checks
	}
	if outcome.Stopping {
		detail := "stopped while " + string(agent.RunStatusAwaitingDecision)
		if !outcome.Stopped {
			detail = "it was " + string(outcome.Status) + ", not waiting for a decision"
		}
		checks = append(checks, CheckResult{
			Name:   "the run was stopped while it waited for a decision",
			Passed: outcome.Stopped,
			Detail: detail,
		})
	}
	if expect.Run != "" {
		checks = append(checks, CheckResult{
			Name:   "the run ended " + expect.Run,
			Passed: strings.EqualFold(string(outcome.Status), expect.Run),
			Detail: "ended " + string(outcome.Status),
		})
	}
	if expect.Flagged != nil {
		name := "the run flagged it for a person"
		if !*expect.Flagged {
			name = "the run flagged nothing"
		}
		checks = append(checks, CheckResult{
			Name:   name,
			Passed: (outcome.Flagged > 0) == *expect.Flagged,
			Detail: fmt.Sprintf("%d flagged", outcome.Flagged),
		})
	}

	return checks
}
