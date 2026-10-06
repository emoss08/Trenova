package agentreflectionservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	maxEvidenceChars = 300
	maxWhyChars      = 300
)

type lesson struct {
	Kind        string `json:"kind"`
	Content     string `json:"content"`
	Audience    string `json:"audience"`
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
	ToolName    string `json:"toolName"`
	Replaces    string `json:"replaces"`
	Evidence    string `json:"evidence"`
	Why         string `json:"why"`
}

type reflectionReply struct {
	Lessons []lesson `json:"lessons"`
	Notes   string   `json:"notes"`
}

func ParseReply(text string) (*reflectionReply, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errortypes.NewBusinessError("The model returned nothing to read")
	}

	var reply reflectionReply
	if err := sonic.UnmarshalString(text, &reply); err != nil {
		return nil, errortypes.NewBusinessError(
			"The model's answer could not be read: {0}",
			err.Error(),
		)
	}
	if len(reply.Lessons) > agent.MaxReflectionLessons {
		reply.Lessons = reply.Lessons[:agent.MaxReflectionLessons]
	}

	return &reply, nil
}

func (s *Service) Finish(
	ctx context.Context,
	plan *services.ReflectionPlan,
	result *services.StructuredCompletionResult,
) (*services.ReflectionOutcome, error) {
	reflection, err := s.repo.GetByID(ctx, repositories.GetAgentReflectionRequest{
		ID:         plan.ReflectionID,
		TenantInfo: plan.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if reflection.Status.Settled() {
		return outcomeOf(reflection), nil
	}

	if result != nil {
		reflection.Model = result.ModelIdentifier
		reflection.ProviderID = result.ProviderID
		reflection.InputTokens = result.InputTokens
		reflection.OutputTokens = result.OutputTokens
	}

	var text string
	if result != nil {
		text = result.Text
	}
	reply, err := ParseReply(text)
	if err != nil {
		reflection.Fail(err.Error(), s.now())
		if _, updateErr := s.repo.Update(ctx, reflection); updateErr != nil {
			return nil, updateErr
		}

		return outcomeOf(reflection), nil
	}

	changes := make([]agent.ReflectionChange, 0, len(reply.Lessons))
	for idx := range reply.Lessons {
		change, applyErr := s.apply(ctx, plan, &reply.Lessons[idx])
		if applyErr != nil {
			return nil, applyErr
		}
		changes = append(changes, change)
	}

	reflection.Complete(changes, reply.Notes, s.now())
	reflection.Tainted = plan.Tainted()
	if reflection, err = s.repo.Update(ctx, reflection); err != nil {
		return nil, err
	}

	s.attach(ctx, plan, changes)
	s.announce(ctx, plan, reflection)

	return outcomeOf(reflection), nil
}

func (s *Service) apply(
	ctx context.Context,
	plan *services.ReflectionPlan,
	in *lesson,
) (agent.ReflectionChange, error) {
	change := agent.ReflectionChange{
		Kind: agent.MemoryKind(in.Kind),
		Content: stringutils.Ellipsize(
			strings.TrimSpace(in.Content),
			agent.MaxReflectionLessonChars,
		),
	}

	req, reason, err := s.lessonRequest(ctx, plan, in)
	if err != nil {
		return change, err
	}
	if reason != "" {
		change.Action = agent.ReflectionActionRefused
		change.Reason = reason

		return change, nil
	}
	change.Content = req.Content

	memory, err := s.memories.Remember(ctx, req, reflectionActor(plan))
	if err != nil {
		if !refused(err) {
			return change, fmt.Errorf("keep a lesson: %w", err)
		}
		change.Action = agent.ReflectionActionRefused
		change.Reason = stringutils.Ellipsize(err.Error(), agent.MaxReflectionChangeReason)

		return change, nil
	}

	change.MemoryID = pulid.PtrOrNil(memory.ID)
	change.Scope = memory.Scope
	change.SupersedesID = pulid.ClonePointer(memory.SupersedesID)
	switch {
	case memory.Refreshed:
		change.Action = agent.ReflectionActionRefreshed
	case memory.Status == agent.MemoryStatusSuggested:
		change.Action = agent.ReflectionActionSuggested
	default:
		change.Action = agent.ReflectionActionSaved
	}

	return change, nil
}

func (s *Service) lessonRequest(
	ctx context.Context,
	plan *services.ReflectionPlan,
	in *lesson,
) (*services.RememberRequest, string, error) {
	person := plan.PersonUserID
	kind := agent.MemoryKind(strings.TrimSpace(in.Kind))
	switch kind {
	case agent.MemoryKindProcedure, agent.MemoryKindFact:
	case agent.MemoryKindInstruction:
		if person.IsNil() {
			return nil, "An agent working with nobody in the work keeps no instructions", nil
		}
	default:
		return nil, fmt.Sprintf("%q is not a kind of lesson that is kept", in.Kind), nil
	}

	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, "The lesson said nothing to keep", nil
	}
	if utf8.RuneCountInString(content) > agent.MaxReflectionLessonChars {
		content = stringutils.Ellipsize(content, agent.MaxReflectionLessonChars)
	}

	req := &services.RememberRequest{
		TenantInfo:        plan.TenantInfo,
		Kind:              kind,
		Content:           content,
		RunID:             plan.RunID,
		Taint:             plan.Taint.Clone(),
		PersonUserID:      person,
		Source:            agent.MemorySourceReflection,
		AgentDefinitionID: plan.AgentDefinitionID,
		ThreadID:          plan.ThreadID,
		ReflectionID:      plan.ReflectionID,
		Evidence: &agent.MemoryEvidence{
			Reason:  stringutils.Ellipsize(strings.TrimSpace(in.Why), maxWhyChars),
			Quotes:  quotes(in.Evidence),
			Signals: plan.Signals.Kinds(),
		},
	}

	if reason := subjectOf(plan, in, req); reason != "" {
		return nil, reason, nil
	}
	if tool := strings.TrimSpace(in.ToolName); tool != "" {
		if !slices.Contains(plan.ToolNames, tool) {
			return nil, "It names a tool the work did not use", nil
		}
		req.ToolName = tool
	}
	if raw := strings.TrimSpace(in.Replaces); raw != "" {
		replaces, err := pulid.Parse(raw)
		if err != nil || !slices.Contains(plan.KnownMemoryIDs, replaces) {
			return nil, "It replaces a memory the look back was not shown", nil
		}
		req.Replaces = replaces
	}

	reason, err := s.audience(ctx, plan, in.Audience, req)
	if err != nil || reason != "" {
		return nil, reason, err
	}

	return req, "", nil
}

func subjectOf(plan *services.ReflectionPlan, in *lesson, req *services.RememberRequest) string {
	raw := strings.TrimSpace(in.SubjectID)
	if raw == "" {
		return ""
	}

	subjectType := agent.MemorySubjectType(strings.TrimSpace(in.SubjectType))
	id, err := pulid.Parse(raw)
	if err != nil || !subjectType.IsValid() ||
		!slices.Contains(plan.Subjects, services.ReflectionSubjectRef{
			Type: subjectType,
			ID:   id,
		}) {
		return "It names a record the work did not touch"
	}
	req.SubjectType = subjectType
	req.SubjectID = id

	return ""
}

func (s *Service) audience(
	ctx context.Context,
	plan *services.ReflectionPlan,
	audience string,
	req *services.RememberRequest,
) (string, error) {
	person := plan.PersonUserID
	suggest := plan.Tainted()

	if person.IsNil() {
		switch audience {
		case audienceOrganization:
			req.Scope = agent.MemoryScopeOrganization
			suggest = true
		default:
			req.Scope = agent.MemoryScopeAgent
		}
		req.Suggest = suggest

		return "", nil
	}

	shared := audience == audienceAgent || audience == audienceTeam ||
		audience == audienceOrganization
	if shared && !s.mayShare(ctx, plan.TenantInfo, person) {
		audience = audienceMe
	}

	switch audience {
	case audienceAgent:
		req.Scope = agent.MemoryScopeAgent
	case audienceTeam:
		reader, err := s.memories.Reader(ctx, plan.TenantInfo, person)
		if err != nil {
			return "", err
		}
		if len(reader.RoleIDs) == 0 {
			req.Scope = agent.MemoryScopeUser
			req.OwnerUserID = person
			break
		}
		req.Scope = agent.MemoryScopeRole
		req.RoleID = reader.RoleIDs[0]
		suggest = true
	case audienceOrganization:
		req.Scope = agent.MemoryScopeOrganization
		suggest = true
	default:
		req.Scope = agent.MemoryScopeUser
		req.OwnerUserID = person
	}

	if !suggest {
		mode, err := s.memories.SavingMode(ctx, plan.TenantInfo, person)
		if err != nil {
			return "", err
		}
		suggest = mode == agent.MemorySavingAskFirst
	}
	req.Suggest = suggest

	return "", nil
}

func (s *Service) mayShare(
	ctx context.Context,
	tenant pagination.TenantInfo,
	person pulid.ID,
) bool {
	if s.permissions == nil {
		return false
	}

	actor := services.UserActor(
		pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID, UserID: person},
	)
	result, err := s.permissions.Check(
		ctx,
		actor.PermissionCheck(permission.ResourceAgentMemory, permission.OpCreate),
	)
	if err != nil {
		s.l.Warn(
			"agent reflection: the person's memory permission could not be read; keeping lessons for them alone",
			zap.String("user", person.String()),
			zap.Error(err),
		)

		return false
	}

	return result != nil && result.Allowed
}

func (s *Service) attach(
	ctx context.Context,
	plan *services.ReflectionPlan,
	changes []agent.ReflectionChange,
) {
	if plan.ThreadID.IsNil() || plan.ReplyMessageID.IsNil() {
		return
	}

	saved := make([]conversation.SavedMemory, 0, len(changes))
	for _, change := range changes {
		if !change.Kept() {
			continue
		}
		saved = append(saved, conversation.SavedMemory{
			ID:      *change.MemoryID,
			Pending: change.Action == agent.ReflectionActionSuggested,
		})
	}
	if len(saved) == 0 {
		return
	}

	if err := s.conversations.AddSavedMemories(ctx, repositories.AddSavedMemoriesRequest{
		MessageID:  plan.ReplyMessageID,
		ThreadID:   plan.ThreadID,
		TenantInfo: plan.TenantInfo,
		Memories:   saved,
	}); err != nil {
		s.l.Warn("agent reflection: what was learned could not be shown on the reply",
			zap.String("thread", plan.ThreadID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) announce(
	ctx context.Context,
	plan *services.ReflectionPlan,
	reflection *agent.Reflection,
) {
	kept := reflection.KeptMemoryIDs()
	if s.realtime == nil || plan.PersonUserID.IsNil() || len(kept) == 0 {
		return
	}

	if err := s.realtime.PublishResourceInvalidation(ctx, &services.PublishResourceInvalidationRequest{
		OrganizationID: plan.TenantInfo.OrgID,
		BusinessUnitID: plan.TenantInfo.BuID,
		AudienceUserID: plan.PersonUserID,
		Resource:       services.AgentMemoryResource,
		Action:         "learned",
		RecordID:       reflection.ID,
		ActorType:      services.PrincipalTypeAgent,
		ActorID:        plan.AgentDefinitionID,
		Entity: map[string]any{
			"id":        reflection.ID,
			"threadId":  plan.ThreadID,
			"memoryIds": kept,
		},
	}); err != nil {
		s.l.Warn("agent reflection: the person could not be told what was learned", zap.Error(err))
	}
}

func reflectionActor(plan *services.ReflectionPlan) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAgent,
		PrincipalID:    plan.AgentDefinitionID,
		OrganizationID: plan.TenantInfo.OrgID,
		BusinessUnitID: plan.TenantInfo.BuID,
	}
}

func quotes(evidence string) []string {
	evidence = strings.TrimSpace(evidence)
	if evidence == "" {
		return nil
	}

	return []string{stringutils.Ellipsize(evidence, maxEvidenceChars)}
}

func outcomeOf(reflection *agent.Reflection) *services.ReflectionOutcome {
	outcome := &services.ReflectionOutcome{
		ReflectionID: reflection.ID,
		Status:       reflection.Status,
	}
	for _, change := range reflection.Changes {
		switch change.Action {
		case agent.ReflectionActionSaved, agent.ReflectionActionRefreshed:
			outcome.Kept++
		case agent.ReflectionActionSuggested:
			outcome.Offered++
		case agent.ReflectionActionRefused:
			outcome.Refused++
		}
	}

	return outcome
}
