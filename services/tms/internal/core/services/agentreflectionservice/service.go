package agentreflectionservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	Repo          repositories.AgentReflectionRepository
	Conversations repositories.ConversationRepository
	Definitions   repositories.AgentDefinitionRepository
	Controls      repositories.AgentControlRepository
	Runs          repositories.AgentRunRepository
	Proposals     repositories.AgentProposalRepository
	Decisions     repositories.AgentDecisionRepository
	Memories      services.AgentMemoryService
	Feedback      repositories.AIFeedbackRepository `optional:"true"`
	Budgets       services.AgentBudgetService       `optional:"true"`
	Permissions   services.PermissionEngine         `optional:"true"`
	Realtime      services.RealtimeService          `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          repositories.AgentReflectionRepository
	conversations repositories.ConversationRepository
	definitions   repositories.AgentDefinitionRepository
	controls      repositories.AgentControlRepository
	runs          repositories.AgentRunRepository
	proposals     repositories.AgentProposalRepository
	decisions     repositories.AgentDecisionRepository
	memories      services.AgentMemoryService
	feedback      repositories.AIFeedbackRepository
	budgets       services.AgentBudgetService
	permissions   services.PermissionEngine
	realtime      services.RealtimeService
	now           func() int64
}

func New(p Params) services.AgentReflectionService {
	return &Service{
		l:             p.Logger.Named("service.agentreflection"),
		repo:          p.Repo,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		controls:      p.Controls,
		runs:          p.Runs,
		proposals:     p.Proposals,
		decisions:     p.Decisions,
		memories:      p.Memories,
		feedback:      p.Feedback,
		budgets:       p.Budgets,
		permissions:   p.Permissions,
		realtime:      p.Realtime,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) PrepareThread(
	ctx context.Context,
	req *services.ReflectOnThreadRequest,
) (*services.ReflectionPlan, error) {
	tenant := req.TenantInfo
	thread, err := s.conversations.GetThreadOwned(ctx, repositories.GetThreadOwnedRequest{
		ID:         req.ThreadID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &services.ReflectionPlan{TenantInfo: tenant}, nil
		}

		return nil, fmt.Errorf("read the conversation to look back over: %w", err)
	}

	after := -1
	through, found, err := s.repo.ThreadReflectedThrough(
		ctx,
		repositories.ThreadReflectedThroughRequest{
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

	messages, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:      thread.ID,
		TenantInfo:    tenant,
		AfterSequence: &after,
		ExcludeKinds:  windowHiddenKinds(),
	})
	if err != nil {
		return nil, fmt.Errorf("read the conversation's latest turns: %w", err)
	}
	if len(messages) == 0 {
		return &services.ReflectionPlan{TenantInfo: tenant}, nil
	}

	person := thread.UserID
	reflection, err := s.claim(ctx, &agent.Reflection{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: thread.AgentDefinitionID,
		SubjectType:       agent.ReflectionSubjectThread,
		ThreadID:          pulid.PtrOrNil(thread.ID),
		UserID:            pulid.PtrOrNil(person),
		FromSequence:      messages[0].Sequence,
		ThroughSequence:   messages[len(messages)-1].Sequence,
	})
	if err != nil || reflection == nil {
		return &services.ReflectionPlan{TenantInfo: tenant}, err
	}

	plan := &services.ReflectionPlan{
		TenantInfo:        tenant,
		ReflectionID:      reflection.ID,
		Subject:           agent.ReflectionSubjectThread,
		AgentDefinitionID: thread.AgentDefinitionID,
		ThreadID:          thread.ID,
		PersonUserID:      person,
		ReplyMessageID:    lastReply(messages),
		Taint:             cleanOr(thread.Taint),
	}

	definition, skip, err := s.usableAgent(ctx, tenant, thread.AgentDefinitionID)
	if err != nil {
		return nil, err
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
	}

	return s.ready(ctx, &readyParams{
		reflection: reflection,
		plan:       plan,
		definition: definition,
		window:     window,
		trigger:    "a conversation on the " + string(thread.Origin),
		provider:   firstProvider(thread.PreferredProviderID, definition.PreferredProviderID),
	})
}

func (s *Service) PrepareRun(
	ctx context.Context,
	req *services.ReflectOnRunRequest,
) (*services.ReflectionPlan, error) {
	tenant := req.TenantInfo
	run, err := s.runs.GetByID(
		ctx,
		repositories.GetAgentRunByIDRequest{ID: req.RunID, TenantInfo: &tenant},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &services.ReflectionPlan{TenantInfo: tenant}, nil
		}

		return nil, fmt.Errorf("read the run to look back over: %w", err)
	}
	if run.AgentDefinitionID.IsNil() || !runSettled(run.Status) {
		return &services.ReflectionPlan{TenantInfo: tenant}, nil
	}

	reflection, err := s.claim(ctx, &agent.Reflection{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: run.AgentDefinitionID,
		SubjectType:       agent.ReflectionSubjectRun,
		RunID:             pulid.PtrOrNil(run.ID),
	})
	if err != nil || reflection == nil {
		return &services.ReflectionPlan{TenantInfo: tenant}, err
	}

	plan := &services.ReflectionPlan{
		TenantInfo:        tenant,
		ReflectionID:      reflection.ID,
		Subject:           agent.ReflectionSubjectRun,
		AgentDefinitionID: run.AgentDefinitionID,
		RunID:             run.ID,
		Taint:             runTaint(run),
	}

	definition, skip, err := s.usableAgent(ctx, tenant, run.AgentDefinitionID)
	if err != nil {
		return nil, err
	}
	if skip != "" {
		return s.skip(ctx, reflection, plan, skip)
	}

	messages, err := s.runMessages(ctx, tenant, run.ID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return s.skip(ctx, reflection, plan, agent.ReflectionSkipNothingToRead)
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
}

type readyParams struct {
	reflection *agent.Reflection
	plan       *services.ReflectionPlan
	definition *agentdefinition.Definition
	window     *Window
	trigger    string
	provider   pulid.ID
}

func (s *Service) ready(ctx context.Context, p *readyParams) (*services.ReflectionPlan, error) {
	plan := p.plan
	signals := ReadSignals(p.window)
	plan.Signals = signals
	p.reflection.Signals = signals
	if len(signals) == 0 {
		return s.skip(ctx, p.reflection, plan, agent.ReflectionSkipNoSignal)
	}
	if s.overBudget(ctx, p.definition) {
		return s.skip(ctx, p.reflection, plan, agent.ReflectionSkipOverBudget)
	}

	toolNames := windowToolNames(p.window.Messages)
	known, err := s.memories.ForContext(ctx, services.MemoryContextRequest{
		TenantInfo:        plan.TenantInfo,
		AgentDefinitionID: plan.AgentDefinitionID,
		ReaderUserID:      plan.PersonUserID,
		ToolNames:         toolNames,
		Records:           windowRecords(p.window.Messages),
	})
	if err != nil {
		return nil, fmt.Errorf("read the memories already kept: %w", err)
	}

	memories := known.Memories
	if len(memories) > maxKnownMemories {
		memories = memories[:maxKnownMemories]
	}
	plan.ToolNames = toolNames
	plan.KnownMemoryIDs = memoryIDs(memories)
	plan.Subjects = subjectRefs(known.Subjects)
	for _, memory := range memories {
		if memory.DrawnFromOutside() {
			plan.Taint.Add(agent.TaintMark{
				Source: agent.TaintSourceMemory,
				Ref: &agent.RecordRef{
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
	}
	version := p.definition.Version
	plan.Request = &services.StructuredCompletionRequest{
		TenantInfo:          plan.TenantInfo,
		Task:                aiprovider.TaskAssistantChat,
		System:              systemPrompt(in),
		Context:             contextSections(in),
		OutputSchema:        outputSchema(in.HasPerson),
		SchemaName:          schemaName,
		MaxTokens:           reflectionMaxTokens,
		PreferredProviderID: p.provider,
		Attribution: services.AIUsageAttribution{
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

func (s *Service) claim(ctx context.Context, entity *agent.Reflection) (*agent.Reflection, error) {
	entity.Status = agent.ReflectionStatusRunning
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
	if claimed.Status == agent.ReflectionStatusFailed {
		claimed.Status = agent.ReflectionStatusRunning
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
	reflection *agent.Reflection,
	plan *services.ReflectionPlan,
	reason agent.ReflectionSkip,
) (*services.ReflectionPlan, error) {
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
) (*agentdefinition.Definition, agent.ReflectionSkip, error) {
	control, err := s.controls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, "", fmt.Errorf("read the organization's agent control: %w", err)
	}
	if control.LearningOff || control.ShadowMode {
		return nil, agent.ReflectionSkipLearningOff, nil
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         agentID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, agent.ReflectionSkipAgentUnavailable, nil
		}

		return nil, "", fmt.Errorf("read the agent: %w", err)
	}
	if !definition.Enabled {
		return definition, agent.ReflectionSkipAgentUnavailable, nil
	}
	if definition.LearningOff {
		return definition, agent.ReflectionSkipLearningOff, nil
	}

	return definition, "", nil
}

func (s *Service) overBudget(ctx context.Context, definition *agentdefinition.Definition) bool {
	if s.budgets == nil {
		return false
	}

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

func (s *Service) Fail(ctx context.Context, req *services.FailReflectionRequest) error {
	if req == nil || req.Plan == nil || req.Plan.ReflectionID.IsNil() {
		return nil
	}

	reflection, err := s.repo.GetByID(ctx, repositories.GetAgentReflectionRequest{
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
	req *repositories.ListAgentReflectionConnectionRequest,
) (*pagination.CursorListResult[*agent.Reflection], error) {
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
			Ref:    &agent.RecordRef{EntityType: "agent_run", ID: run.ID.String()},
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

func memoryIDs(memories []*agent.Memory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}

	return ids
}

func subjectRefs(subjects []agent.MemorySubject) []services.ReflectionSubjectRef {
	refs := make([]services.ReflectionSubjectRef, 0, len(subjects))
	for _, subject := range subjects {
		refs = append(refs, services.ReflectionSubjectRef{Type: subject.Type, ID: subject.ID})
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
