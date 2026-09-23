package aifeedbackservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const taskAgentRun = "AgentRun"

type resolveParams struct {
	tenant pagination.TenantInfo
	target repositories.AIFeedbackTargetRef
	actor  *services.RequestActor
}

type resolvedTarget struct {
	threadID          pulid.ID
	turnID            pulid.ID
	runID             pulid.ID
	agentDefinitionID pulid.ID
	definitionVersion *int64
	detectorKey       string
	task              string
	model             string
	providerID        pulid.ID
	subject           string
	toolNames         []string
	snapshot          *aifeedback.TurnSnapshot
}

type feedbackParams struct {
	tenant  pagination.TenantInfo
	userID  pulid.ID
	target  repositories.AIFeedbackTargetRef
	rating  aifeedback.Rating
	reasons []aifeedback.Reason
	comment string
}

func (r *resolvedTarget) fingerprintSource() aifeedback.FingerprintSource {
	if r.agentDefinitionID.IsNil() && r.detectorKey == "" && r.model == "" {
		return aifeedback.FingerprintNone
	}

	return aifeedback.FingerprintAtRating
}

func (r *resolvedTarget) feedback(p feedbackParams) *aifeedback.Feedback {
	reasons := p.reasons
	if reasons == nil {
		reasons = []aifeedback.Reason{}
	}

	subject := r.subject
	if subject == "" {
		subject = p.target.TargetType.String()
	}

	return &aifeedback.Feedback{
		OrganizationID:    p.tenant.OrgID,
		BusinessUnitID:    p.tenant.BuID,
		UserID:            p.userID,
		TargetType:        p.target.TargetType,
		TargetID:          p.target.TargetID,
		TargetPart:        p.target.TargetPart,
		ThreadID:          optionalID(r.threadID),
		TurnID:            optionalID(r.turnID),
		RunID:             optionalID(r.runID),
		AgentDefinitionID: optionalID(r.agentDefinitionID),
		DefinitionVersion: r.definitionVersion,
		DetectorKey:       r.detectorKey,
		Task:              r.task,
		Model:             r.model,
		ProviderID:        optionalID(r.providerID),
		FingerprintSource: r.fingerprintSource(),
		Rating:            p.rating,
		Reasons:           reasons,
		Comment:           p.comment,
		TurnSnapshot:      r.snapshot,
		PatternKey:        aifeedback.PatternKey(r.toolNames, firstReason(reasons), subject),
	}
}

func (s *Service) resolve(ctx context.Context, p resolveParams) (*resolvedTarget, error) {
	var (
		resolved *resolvedTarget
		err      error
	)

	switch p.target.TargetType {
	case aifeedback.TargetAssistantMessage, aifeedback.TargetDelegatedAnswer:
		resolved, err = s.resolveMessage(ctx, p)
	case aifeedback.TargetBriefing, aifeedback.TargetBriefingSection:
		resolved, err = s.resolveBriefing(ctx, p)
	case aifeedback.TargetInsight:
		resolved, err = s.resolveInsight(ctx, p)
	case aifeedback.TargetWatchtowerItem:
		resolved, err = s.resolveWatchtowerItem(ctx, p)
	default:
		return nil, errortypes.NewValidationError(
			"targetType",
			errortypes.ErrInvalid,
			"Target type is invalid",
		)
	}
	if err != nil {
		return nil, err
	}

	s.stampDefinitionVersion(ctx, p.tenant, resolved)

	return resolved, nil
}

func (s *Service) resolveMessage(ctx context.Context, p resolveParams) (*resolvedTarget, error) {
	source, err := s.sources.GetMessageContext(ctx, repositories.GetAIFeedbackMessageRequest{
		TenantInfo: p.tenant,
		MessageID:  p.target.TargetID,
	})
	if err != nil {
		return nil, err
	}
	if source.Thread == nil || source.Thread.UserID != p.actor.UserID {
		return nil, errortypes.NewNotFoundError("Answer not found within your organization")
	}

	message := source.Message
	delegated := message.Delegated()
	if message.Role != conversation.RoleAssistant {
		return nil, notRatable("targetId", "Only an answer can be rated")
	}
	if delegated != (p.target.TargetType == aifeedback.TargetDelegatedAnswer) {
		return nil, notRatable("targetType", "The target type does not match the answer")
	}

	agentID := source.Thread.AgentDefinitionID
	if delegated && message.AgentDefinitionID.IsNotNil() {
		agentID = message.AgentDefinitionID
	}

	exchange := exchangeOf(source.Exchange, message)
	resolved := &resolvedTarget{
		threadID:          source.Thread.ID,
		agentDefinitionID: agentID,
		task:              string(aiprovider.TaskAssistantChat),
		model:             message.Model,
		providerID:        message.ProviderID,
		subject:           string(source.Thread.SubjectType),
		toolNames:         exchange.toolNames,
		snapshot: aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
			Question: exchange.question,
			Answer:   exchange.answer,
			Tools:    exchange.tools,
			Redact:   s.redactor.restrictedValues(exchange.results),
		}),
	}
	if resolved.model == "" {
		resolved.model = exchange.model
		resolved.providerID = exchange.providerID
	}
	if source.Turn != nil {
		resolved.turnID = source.Turn.ID
		resolved.runID = source.Turn.RunID
	}

	return resolved, nil
}

type exchangeSummary struct {
	question   string
	answer     string
	tools      []aifeedback.ToolLine
	toolNames  []string
	results    []toolResult
	model      string
	providerID pulid.ID
}

func exchangeOf(messages []conversation.Message, rated *conversation.Message) exchangeSummary {
	summary := exchangeSummary{
		tools:     make([]aifeedback.ToolLine, 0, len(messages)),
		toolNames: make([]string, 0, len(messages)),
		results:   make([]toolResult, 0, len(messages)),
	}
	answers := make([]string, 0, len(messages))

	for index := range messages {
		message := &messages[index]
		switch message.Role {
		case conversation.RoleUser:
			if summary.question == "" {
				summary.question = questionText(message)
			}
		case conversation.RoleTool:
			summary.tools = append(summary.tools, aifeedback.ToolLine{
				Name:    message.ToolName,
				Summary: message.ToolSummary,
				Failed:  message.ToolFailed,
			})
			summary.toolNames = append(summary.toolNames, message.ToolName)
			summary.results = append(summary.results, toolResult{
				name:    message.ToolName,
				content: message.Content,
			})
		case conversation.RoleAssistant:
			if content := strings.TrimSpace(message.Content); content != "" {
				answers = append(answers, content)
			}
			if message.Model != "" {
				summary.model = message.Model
				summary.providerID = message.ProviderID
			}
		}
	}

	summary.answer = strings.TrimSpace(rated.Content)
	if summary.answer == "" {
		summary.answer = strings.Join(answers, "\n\n")
	}

	return summary
}

func questionText(message *conversation.Message) string {
	if message.Kind == conversation.MessageKindDecisionNote {
		first, _, _ := strings.Cut(message.Content, "\n")

		return strings.TrimSpace(first)
	}

	return strings.TrimSpace(message.Content)
}

func (s *Service) resolveBriefing(ctx context.Context, p resolveParams) (*resolvedTarget, error) {
	entity, err := s.briefings.GetByID(ctx, repositories.GetBriefingByIDRequest{
		ID:         p.target.TargetID,
		TenantInfo: p.tenant,
	})
	if err != nil {
		return nil, err
	}

	if entity.UserID != nil {
		if *entity.UserID != p.actor.UserID {
			return nil, errortypes.NewNotFoundError("Briefing not found within your organization")
		}
	} else if err = s.requireRead(ctx, p.actor, permission.ResourceBriefing); err != nil {
		return nil, err
	}

	resolved := &resolvedTarget{
		runID:      entity.RunID,
		task:       string(aiprovider.TaskDailyBriefing),
		model:      entity.ModelIdentifier,
		providerID: entity.ProviderID,
		subject:    entity.RoleKey.String(),
	}
	question := fmt.Sprintf("%s briefing for %s", entity.RoleKey.Label(), entity.BriefingDate)

	if p.target.TargetType == aifeedback.TargetBriefing {
		resolved.snapshot = aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
			Question: question,
			Answer:   briefingText(entity),
		})

		return resolved, nil
	}

	section, ok := sectionOf(entity, p.target.TargetPart)
	if !ok {
		return nil, notRatable("targetPart", "That section is not in this briefing")
	}
	resolved.subject = string(section.Key)
	resolved.snapshot = aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
		Question: question + ": " + section.Title,
		Answer:   sectionText(section),
	})

	return resolved, nil
}

func sectionOf(entity *briefing.Briefing, key string) (briefing.Section, bool) {
	for _, section := range entity.Sections {
		if string(section.Key) == key {
			return section, true
		}
	}

	return briefing.Section{}, false
}

func briefingText(entity *briefing.Briefing) string {
	parts := make([]string, 0, len(entity.Sections)+1)
	if headline := strings.TrimSpace(entity.Headline); headline != "" {
		parts = append(parts, headline)
	}
	for _, section := range entity.Sections {
		parts = append(parts, sectionText(section))
	}

	return strings.Join(parts, "\n\n")
}

func sectionText(section briefing.Section) string {
	var b strings.Builder
	b.WriteString(section.Title)
	if read := strings.TrimSpace(section.Read()); read != "" {
		b.WriteString(": ")
		b.WriteString(read)
	}
	for _, item := range section.Items {
		b.WriteString("\n- ")
		b.WriteString(item.Label)
		b.WriteString(": ")
		b.WriteString(item.Value)
	}

	return b.String()
}

func (s *Service) resolveInsight(ctx context.Context, p resolveParams) (*resolvedTarget, error) {
	if err := s.requireRead(ctx, p.actor, permission.ResourceInsight); err != nil {
		return nil, err
	}

	entity, err := s.insights.GetByID(ctx, repositories.GetInsightByIDRequest{
		ID:         p.target.TargetID,
		TenantInfo: p.tenant,
	})
	if err != nil {
		return nil, err
	}

	return &resolvedTarget{
		detectorKey: entity.DetectorKey,
		task:        string(aiprovider.TaskOperationalInsights),
		model:       entity.ModelIdentifier,
		providerID:  entity.ProviderID,
		subject:     entity.DetectorKey,
		snapshot: aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
			Question: insightQuestion(entity.Subject, string(entity.Category)),
			Answer: strings.Join(stringutils.NonEmptyStrings(
				entity.Headline,
				entity.Narrative,
				entity.Recommendation,
			), "\n\n"),
		}),
	}, nil
}

func insightQuestion(subject, category string) string {
	if strings.TrimSpace(subject) == "" {
		return category
	}

	return category + ": " + subject
}

func (s *Service) resolveWatchtowerItem(
	ctx context.Context,
	p resolveParams,
) (*resolvedTarget, error) {
	if err := s.requireRead(ctx, p.actor, permission.ResourceWatchtower); err != nil {
		return nil, err
	}

	item, err := s.watchtower.GetByID(ctx, repositories.GetWatchtowerItemRequest{
		ID:         p.target.TargetID,
		TenantInfo: p.tenant,
	})
	if err != nil {
		return nil, err
	}
	if !aiSourceKind(item.SourceKind) {
		return nil, notRatable("targetId", "Only an item raised by AI work can be rated")
	}
	if err = s.requireRead(ctx, p.actor, item.SourceKind.ReadResource()); err != nil {
		return nil, err
	}

	resolved := &resolvedTarget{
		subject: item.SourceKind.String(),
		snapshot: aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
			Question: item.SourceKind.Label(),
			Answer:   strings.Join(stringutils.NonEmptyStrings(item.Title, item.Summary), "\n\n"),
		}),
	}
	s.creditWatchtowerSource(ctx, p.tenant, item, resolved)

	return resolved, nil
}

func aiSourceKind(kind watchtower.SourceKind) bool {
	switch kind {
	case watchtower.SourceInsight,
		watchtower.SourceAgentProposal,
		watchtower.SourceAgentPlan,
		watchtower.SourceAgentRunFailed,
		watchtower.SourceAgentException:
		return true
	default:
		return false
	}
}

func (s *Service) creditWatchtowerSource(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *watchtower.Item,
	resolved *resolvedTarget,
) {
	sourceID := pulid.ID(strings.TrimSpace(item.SourceID))
	if sourceID.IsNil() {
		return
	}

	var (
		runID pulid.ID
		err   error
	)

	switch item.SourceKind {
	case watchtower.SourceInsight:
		entity, iErr := s.insights.GetByID(ctx, repositories.GetInsightByIDRequest{
			ID:         sourceID,
			TenantInfo: tenant,
		})
		if iErr != nil {
			err = iErr
			break
		}
		resolved.detectorKey = entity.DetectorKey
		resolved.task = string(aiprovider.TaskOperationalInsights)
		resolved.model = entity.ModelIdentifier
		resolved.providerID = entity.ProviderID

		return
	case watchtower.SourceAgentProposal:
		proposal, pErr := s.proposals.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
			ID:         sourceID,
			TenantInfo: &tenant,
		})
		if pErr != nil {
			err = pErr
			break
		}
		runID = proposal.RunID
	case watchtower.SourceAgentPlan:
		plan, pErr := s.plans.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
			ID:         sourceID,
			TenantInfo: tenant,
		})
		if pErr != nil {
			err = pErr
			break
		}
		runID = plan.RunID
	case watchtower.SourceAgentRunFailed:
		runID = sourceID
	case watchtower.SourceAgentException:
		exception, xErr := s.exceptions.GetByID(ctx, repositories.GetAgentExceptionByIDRequest{
			ID:         sourceID,
			TenantInfo: &tenant,
		})
		if xErr != nil {
			err = xErr
			break
		}
		runID = exception.RunID
	}

	if err == nil && runID.IsNotNil() {
		err = s.creditRun(ctx, tenant, runID, resolved)
	}
	if err != nil {
		s.l.Warn("aifeedback: could not credit a watchtower item to its source",
			zap.String("item", item.ID.String()),
			zap.String("sourceKind", item.SourceKind.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) creditRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	resolved *resolvedTarget,
) error {
	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return err
	}

	resolved.runID = run.ID
	resolved.agentDefinitionID = run.AgentDefinitionID
	resolved.model = run.ModelIdentifier
	resolved.task = taskAgentRun

	return nil
}

func (s *Service) stampDefinitionVersion(
	ctx context.Context,
	tenant pagination.TenantInfo,
	resolved *resolvedTarget,
) {
	if resolved.agentDefinitionID.IsNil() || s.definitions == nil {
		return
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         resolved.agentDefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("aifeedback: could not read the credited agent's version",
			zap.String("agent", resolved.agentDefinitionID.String()),
			zap.Error(err),
		)

		return
	}

	version := definition.Version
	resolved.definitionVersion = &version
}

func (s *Service) requireRead(
	ctx context.Context,
	actor *services.RequestActor,
	resource permission.Resource,
) error {
	if s.permissions == nil {
		return errortypes.NewAuthorizationError("Permissions could not be checked")
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return err
	}
	if result == nil || !result.Allowed {
		return errortypes.NewAuthorizationError(
			"You do not have permission to read {0}",
			resource.String(),
		)
	}

	return nil
}

func notRatable(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

func firstReason(reasons []aifeedback.Reason) aifeedback.Reason {
	if len(reasons) == 0 {
		return ""
	}

	return reasons[0]
}

func optionalID(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}

	return &id
}
