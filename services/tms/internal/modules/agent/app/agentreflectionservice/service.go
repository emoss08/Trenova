package agentreflectionservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          agentreflectionports.AgentReflectionRepository
	Corrections   agentreflectionports.CorrectedWorkRepository
	Conversations conversationports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Controls      tenantports.AgentControlRepository
	Runs          agentruntimeports.AgentRunRepository
	Proposals     agentproposalports.AgentProposalRepository
	Decisions     agentdecisionports.AgentDecisionRepository
	Memories      agentmemoryports.AgentMemoryService
	Feedback      aifeedbackports.AIFeedbackRepository
	Budgets       agentbudgetports.AgentBudgetService
	Realtime      realtimeports.RealtimeService
}

type Service struct {
	l             *zap.Logger
	repo          agentreflectionports.AgentReflectionRepository
	corrections   agentreflectionports.CorrectedWorkRepository
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	controls      tenantports.AgentControlRepository
	runs          agentruntimeports.AgentRunRepository
	proposals     agentproposalports.AgentProposalRepository
	decisions     agentdecisionports.AgentDecisionRepository
	memories      agentmemoryports.AgentMemoryService
	feedback      aifeedbackports.AIFeedbackRepository
	budgets       agentbudgetports.AgentBudgetService
	realtime      realtimeports.RealtimeService
	now           func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) agentreflectionports.AgentReflectionService {
	return &Service{
		l:             p.Logger.Named("service.agentreflection"),
		repo:          p.Repo,
		corrections:   p.Corrections,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		controls:      p.Controls,
		runs:          p.Runs,
		proposals:     p.Proposals,
		decisions:     p.Decisions,
		memories:      p.Memories,
		feedback:      p.Feedback,
		budgets:       p.Budgets,
		realtime:      p.Realtime,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) PrepareThread(
	ctx context.Context,
	req *agentreflectionports.ReflectOnThreadRequest,
) (*agentreflectionports.ReflectionPlan, error) {
	tenant := req.TenantInfo
	thread, err := s.conversations.GetThreadOwned(ctx, conversationports.GetThreadOwnedRequest{
		ID:         req.ThreadID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, nil
		}

		return nil, fmt.Errorf("read the conversation to look back over: %w", err)
	}

	after := -1
	through, found, err := s.repo.ThreadReflectedThrough(
		ctx,
		agentreflectionports.ThreadReflectedThroughRequest{
			ThreadID:   thread.ID,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	if found {
		after = through
	}

	messages, err := s.conversations.ListMessages(ctx, &conversationports.ListMessagesRequest{
		ThreadID:      thread.ID,
		TenantInfo:    tenant,
		AfterSequence: &after,
		ExcludeKinds:  windowHiddenKinds(),
	})
	if err != nil {
		return nil, fmt.Errorf("read the conversation's latest turns: %w", err)
	}
	if len(messages) == 0 {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, nil
	}

	person := thread.UserID
	reflection, err := s.claim(ctx, &agentreflection.Reflection{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: thread.AgentDefinitionID,
		SubjectType:       agentreflection.ReflectionSubjectThread,
		ThreadID:          pulid.PtrOrNil(thread.ID),
		UserID:            pulid.PtrOrNil(person),
		FromSequence:      messages[0].Sequence,
		ThroughSequence:   messages[len(messages)-1].Sequence,
	})
	if err != nil || reflection == nil {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, err
	}

	plan := &agentreflectionports.ReflectionPlan{
		TenantInfo:        tenant,
		ReflectionID:      reflection.ID,
		Subject:           agentreflection.ReflectionSubjectThread,
		AgentDefinitionID: thread.AgentDefinitionID,
		ThreadID:          thread.ID,
		PersonUserID:      person,
		ReplyMessageID:    lastReply(messages),
		Taint:             cleanOr(thread.Taint),
	}

	return s.claimed(ctx, reflection, func() (*agentreflectionports.ReflectionPlan, error) {
		if failErr := s.repo.FailInterrupted(ctx, &agentreflectionports.FailInterruptedRequest{
			TenantInfo: tenant,
			ThreadID:   thread.ID,
			ExceptID:   reflection.ID,
			Message:    interruptedMessage,
			At:         s.now(),
		}); failErr != nil {
			return nil, failErr
		}

		definition, control, skip, agentErr := s.usableAgent(ctx, tenant, thread.AgentDefinitionID)
		if agentErr != nil {
			return nil, agentErr
		}
		if skip != "" {
			return s.skip(ctx, reflection, plan, skip)
		}

		window := &Window{
			Messages:        messages,
			Proposals:       s.threadProposals(ctx, tenant, thread.ID, messages),
			NegativeRatings: s.negativeRatings(ctx, tenant, person, messages),
			HasPerson:       true,
			Continues:       found,
			InPassing:       !control.PassingLearningOff,
		}

		return s.ready(ctx, &readyParams{
			reflection: reflection,
			plan:       plan,
			definition: definition,
			window:     window,
			trigger:    "a conversation on the " + string(thread.Origin),
			provider: firstProvider(
				thread.PreferredProviderID,
				definition.PreferredProviderID,
			),
		})
	})
}

const interruptedMessage = "The look back was interrupted before it finished"

// claimed runs what follows a claim and records the look back as failed when
// it cannot be prepared. A claimed row left Running reads as a stretch already
// looked over, so the conversation's next look back would start after it and
// what was said in it would never be read.
func (s *Service) claimed(
	ctx context.Context,
	reflection *agentreflection.Reflection,
	prepare func() (*agentreflectionports.ReflectionPlan, error),
) (*agentreflectionports.ReflectionPlan, error) {
	plan, err := prepare()
	if err == nil {
		return plan, nil
	}

	current, readErr := s.repo.GetByID(
		context.WithoutCancel(ctx),
		agentreflectionports.GetAgentReflectionRequest{
			ID: reflection.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: reflection.OrganizationID,
				BuID:  reflection.BusinessUnitID,
			},
		},
	)
	if readErr == nil && !current.Status.Settled() {
		current.Fail(err.Error(), s.now())
		_, readErr = s.repo.Update(context.WithoutCancel(ctx), current)
	}
	if readErr != nil {
		s.l.Warn("agent reflection: a look back that could not be prepared was not marked failed",
			zap.String("reflection", reflection.ID.String()),
			zap.Error(readErr),
		)
	}

	return nil, err
}

func (s *Service) PrepareRun(
	ctx context.Context,
	req *agentreflectionports.ReflectOnRunRequest,
) (*agentreflectionports.ReflectionPlan, error) {
	tenant := req.TenantInfo
	run, err := s.runs.GetByID(
		ctx,
		agentruntimeports.GetAgentRunByIDRequest{ID: req.RunID, TenantInfo: &tenant},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, nil
		}

		return nil, fmt.Errorf("read the run to look back over: %w", err)
	}
	if run.AgentDefinitionID.IsNil() || !runSettled(run.Status) {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, nil
	}

	reflection, err := s.claim(ctx, &agentreflection.Reflection{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: run.AgentDefinitionID,
		SubjectType:       agentreflection.ReflectionSubjectRun,
		RunID:             pulid.PtrOrNil(run.ID),
	})
	if err != nil || reflection == nil {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, err
	}

	plan := &agentreflectionports.ReflectionPlan{
		TenantInfo:        tenant,
		ReflectionID:      reflection.ID,
		Subject:           agentreflection.ReflectionSubjectRun,
		AgentDefinitionID: run.AgentDefinitionID,
		RunID:             run.ID,
		Taint:             runTaint(run),
	}

	return s.claimed(ctx, reflection, func() (*agentreflectionports.ReflectionPlan, error) {
		definition, _, skip, agentErr := s.usableAgent(ctx, tenant, run.AgentDefinitionID)
		if agentErr != nil {
			return nil, agentErr
		}
		if skip != "" {
			return s.skip(ctx, reflection, plan, skip)
		}

		messages, readErr := s.runMessages(ctx, tenant, run.ID)
		if readErr != nil {
			return nil, readErr
		}
		if len(messages) == 0 {
			return s.skip(ctx, reflection, plan, agentreflection.ReflectionSkipNothingToRead)
		}

		return s.ready(ctx, &readyParams{
			reflection: reflection,
			plan:       plan,
			definition: definition,
			window: &Window{
				Messages:  messages,
				Proposals: s.runProposals(ctx, tenant, run.ID),
			},
			trigger:  "a " + string(run.Trigger) + " run about a " + string(run.SubjectType),
			provider: definition.PreferredProviderID,
		})
	})
}

type readyParams struct {
	reflection *agentreflection.Reflection
	plan       *agentreflectionports.ReflectionPlan
	definition *agentdefinition.Definition
	window     *Window
	trigger    string
	provider   pulid.ID
	// records are the records the work touched when there is no transcript to
	// read them from.
	records []agentpagecontext.EntityRef
}

func (s *Service) ready(
	ctx context.Context,
	p *readyParams,
) (*agentreflectionports.ReflectionPlan, error) {
	plan := p.plan
	signals := ReadSignals(p.window)
	plan.Signals = signals
	p.reflection.Signals = signals
	if len(signals) == 0 {
		return s.skip(ctx, p.reflection, plan, agentreflection.ReflectionSkipNoSignal)
	}
	if s.overBudget(ctx, p.definition) {
		return s.skip(ctx, p.reflection, plan, agentreflection.ReflectionSkipOverBudget)
	}

	toolNames := windowToolNames(p.window.Messages)
	records := p.records
	if records == nil {
		records = windowRecords(p.window.Messages)
	}
	known, err := s.memories.ForContext(ctx, &agentmemoryports.MemoryContextRequest{
		TenantInfo:        plan.TenantInfo,
		AgentDefinitionID: plan.AgentDefinitionID,
		ReaderUserID:      plan.PersonUserID,
		ToolNames:         toolNames,
		Records:           records,
	})
	if err != nil {
		return nil, fmt.Errorf("read the memories already kept: %w", err)
	}

	memories := known.Memories
	if len(memories) > maxKnownMemories {
		memories = memories[:maxKnownMemories]
	}
	plan.KnownMemoryIDs = memoryIDs(memories)
	plan.Subjects = subjectRefs(known.Subjects)
	for _, memory := range memories {
		if memory.DrawnFromOutside() {
			plan.Taint.Add(agent.TaintMark{
				Source: agent.TaintSourceMemory,
				Ref: &agentrecord.RecordRef{
					EntityType: agent.TaintEntityAgentMemory,
					ID:         memory.ID.String(),
				},
				At: s.now(),
			})
		}
	}

	in := &promptInput{
		AgentName:   p.definition.Name,
		Description: p.definition.Description,
		Subject:     plan.Subject,
		HasPerson:   plan.PersonUserID.IsNotNil(),
		Trigger:     p.trigger,
		Signals:     signals,
		Memories:    memories,
		Subjects:    plan.Subjects,
		ToolNames:   toolNames,
		Proposals:   p.window.Proposals,
		Messages:    p.window.Messages,
		Corrections: p.window.Corrections,
		Corrected:   p.window.CorrectedWhat,
	}
	version := p.definition.Version
	plan.Request = &llmports.StructuredCompletionRequest{
		TenantInfo:          plan.TenantInfo,
		Task:                aitask.TaskAssistantChat,
		System:              systemPrompt(in),
		Context:             contextSections(in),
		OutputSchema:        outputSchema(in.HasPerson),
		SchemaName:          schemaName,
		MaxTokens:           reflectionMaxTokens,
		PreferredProviderID: p.provider,
		Attribution: aiusageports.AIUsageAttribution{
			UserID:            plan.PersonUserID,
			AgentDefinitionID: plan.AgentDefinitionID,
			ThreadID:          plan.ThreadID,
			RunID:             plan.RunID,
			Feature:           aiusage.FeatureAgentReflection,
			DefinitionVersion: &version,
		},
	}

	p.reflection.Tainted = plan.Tainted()
	if _, err = s.repo.Update(ctx, p.reflection); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) claim(
	ctx context.Context,
	entity *agentreflection.Reflection,
) (*agentreflection.Reflection, error) {
	entity.Status = agentreflection.ReflectionStatusRunning
	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	claimed, err := s.repo.Claim(ctx, entity)
	if err != nil {
		return nil, err
	}
	if claimed.Status.Settled() {
		return nil, nil
	}
	if claimed.Status == agentreflection.ReflectionStatusFailed {
		claimed.Status = agentreflection.ReflectionStatusRunning
		claimed.ErrorMessage = ""
		claimed.FinishedAt = nil
		if claimed, err = s.repo.Update(ctx, claimed); err != nil {
			return nil, err
		}
	}

	return claimed, nil
}

func (s *Service) skip(
	ctx context.Context,
	reflection *agentreflection.Reflection,
	plan *agentreflectionports.ReflectionPlan,
	reason agentreflection.ReflectionSkip,
) (*agentreflectionports.ReflectionPlan, error) {
	reflection.Skip(reason, s.now())
	reflection.Tainted = plan.Tainted()
	if _, err := s.repo.Update(ctx, reflection); err != nil {
		return nil, err
	}
	plan.Request = nil

	return plan, nil
}

func (s *Service) usableAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	agentID pulid.ID,
) (*agentdefinition.Definition, *agentcontrol.AgentControl, agentreflection.ReflectionSkip, error) {
	control, err := s.controls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, nil, "", fmt.Errorf("read the organization's agent control: %w", err)
	}
	if control.LearningOff || control.ShadowMode {
		return nil, control, agentreflection.ReflectionSkipLearningOff, nil
	}

	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         agentID,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, control, agentreflection.ReflectionSkipAgentUnavailable, nil
		}

		return nil, nil, "", fmt.Errorf("read the agent: %w", err)
	}
	if !definition.Enabled {
		return definition, control, agentreflection.ReflectionSkipAgentUnavailable, nil
	}
	if definition.LearningOff {
		return definition, control, agentreflection.ReflectionSkipLearningOff, nil
	}

	return definition, control, "", nil
}

func (s *Service) overBudget(ctx context.Context, definition *agentdefinition.Definition) bool {
	refusal, err := s.budgets.CheckRun(ctx, definition)
	if err != nil {
		s.l.Warn("agent reflection: the agent's budget could not be read; looking back anyway",
			zap.String("agent", definition.ID.String()),
			zap.Error(err),
		)

		return false
	}

	return refusal.Refused()
}

func (s *Service) Fail(ctx context.Context, req *agentreflectionports.FailReflectionRequest) error {
	if req == nil || req.Plan == nil || req.Plan.ReflectionID.IsNil() {
		return nil
	}

	reflection, err := s.repo.GetByID(ctx, agentreflectionports.GetAgentReflectionRequest{
		ID:         req.Plan.ReflectionID,
		TenantInfo: req.Plan.TenantInfo,
	})
	if err != nil {
		return err
	}
	if reflection.Status.Settled() {
		return nil
	}

	reflection.Fail(req.Message, s.now())
	_, err = s.repo.Update(ctx, reflection)

	return err
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentreflectionports.ListAgentReflectionConnectionRequest,
) (*pagination.CursorListResult[*agentreflection.Reflection], error) {
	return s.repo.ListConnection(ctx, req)
}

func refused(err error) bool {
	var validation *errortypes.Error

	return errors.As(err, &validation) ||
		errortypes.IsMultiError(err) ||
		errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err) ||
		errortypes.IsAuthorizationError(err)
}

func runSettled(status agent.RunStatus) bool {
	return status == agent.RunStatusCompleted ||
		status == agent.RunStatusShadowCompleted ||
		status == agent.RunStatusFailed
}

func windowHiddenKinds() []conversation.MessageKind {
	return append(conversation.ModelHiddenKinds(), conversation.MessageKindCompaction)
}

func cleanOr(taint *agent.RunTaint) *agent.RunTaint {
	if taint == nil {
		return &agent.RunTaint{}
	}

	return taint.Clone()
}

func runTaint(run *agent.AgentRun) *agent.RunTaint {
	taint := cleanOr(run.Taint)
	if run.Tainted && !taint.Tainted() {
		taint.Add(agent.TaintMark{
			Source: agent.TaintSourceRunRecord,
			Ref:    &agentrecord.RecordRef{EntityType: "agent_run", ID: run.ID.String()},
			At:     timeutils.NowUnix(),
		})
	}

	return taint
}

func firstProvider(ids ...pulid.ID) pulid.ID {
	for _, id := range ids {
		if id.IsNotNil() {
			return id
		}
	}

	return pulid.Nil
}

func lastReply(messages []conversation.Message) pulid.ID {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		message := &messages[idx]
		if message.Role == conversation.RoleAssistant && message.AgentDefinitionID.IsNil() &&
			message.Kind == conversation.MessageKindMessage {
			return message.ID
		}
	}

	return pulid.Nil
}

func memoryIDs(memories []*agentmemory.Memory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}

	return ids
}

func subjectRefs(subjects []agentmemory.MemorySubject) []agentreflectionports.ReflectionSubjectRef {
	refs := make([]agentreflectionports.ReflectionSubjectRef, 0, len(subjects))
	for _, subject := range subjects {
		refs = append(
			refs,
			agentreflectionports.ReflectionSubjectRef{Type: subject.Type, ID: subject.ID},
		)
	}

	return refs
}

func windowToolNames(messages []conversation.Message) []string {
	names := make([]string, 0, 8)
	for idx := range messages {
		for _, call := range messages[idx].ToolCalls {
			if call.Name != "" && !slices.Contains(names, call.Name) {
				names = append(names, call.Name)
			}
		}
	}

	return names
}
