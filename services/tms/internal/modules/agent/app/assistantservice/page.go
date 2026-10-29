package assistantservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentstarter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type pageSubject struct {
	subjectType agent.SubjectType
	draft       pagedraft.Surface
	title       string
}

type pageSurface struct {
	systemKey       string
	subjectRequired bool
	subjects        []pageSubject
}

func (p *pageSurface) subject(subjectType agent.SubjectType) (pageSubject, bool) {
	if subjectType == "" {
		return p.subjects[0], true
	}
	for _, subject := range p.subjects {
		if subject.subjectType == subjectType {
			return subject, true
		}
	}

	return pageSubject{}, false
}

func (p *pageSurface) takesDraft(draft pagedraft.Surface) bool {
	for _, subject := range p.subjects {
		if subject.draft == draft {
			return true
		}
	}

	return false
}

func (p *pageSurface) nouns() string {
	nouns := make([]string, 0, len(p.subjects))
	for _, subject := range p.subjects {
		nouns = append(nouns, subject.subjectType.NounWithArticle())
	}

	return strings.Join(nouns, " or ")
}

var pageSurfaces = map[conversation.ThreadOrigin]pageSurface{
	conversation.ThreadOriginImport: {
		systemKey:       agentdefinition.SystemKeyImportAssistant,
		subjectRequired: true,
		subjects: []pageSubject{{
			subjectType: agent.SubjectDocument,
			draft:       pagedraft.SurfaceShipmentImport,
			title:       "Shipment import",
		}},
	},
	conversation.ThreadOriginFormula: {
		systemKey: agentdefinition.SystemKeyFormulaAssistant,
		subjects: []pageSubject{
			{
				subjectType: agent.SubjectFormulaTemplate,
				draft:       pagedraft.SurfaceFormula,
				title:       "Formula",
			},
			{
				subjectType: agent.SubjectFormulaLibrary,
				draft:       pagedraft.SurfaceFormulaLibrary,
				title:       "Formula library",
			},
		},
	},
}

func (s *Service) OpenPageThread(
	ctx context.Context,
	req *assistantports.OpenPageThreadRequest,
	actor *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	surface, ok := pageSurfaces[req.Origin]
	if !ok {
		return nil, errortypes.NewValidationError(
			"origin", errortypes.ErrInvalid, "This page has no assistant conversation",
		)
	}
	subject, err := checkPageSubject(&surface, req)
	if err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errAgentsWithheld()
	}
	if s.systemAgents == nil || s.pageThreads == nil {
		return nil, errortypes.NewBusinessError(
			"The {0} assistant is not available on this server", subject.title,
		)
	}

	usable, err := s.usableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, errAgentsWithheld()
	}

	definition, err := s.systemAgents.EnsureSystem(
		ctx,
		agentdefinitionports.EnsureSystemAgentRequest{
			TenantInfo: req.TenantInfo,
			SystemKey:  surface.systemKey,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = s.assertPageAgentUsable(ctx, actor, definition); err != nil {
		return nil, err
	}
	if req.SubjectID.IsNotNil() {
		if err = s.assertSubjectReadable(ctx, actor, subject.subjectType); err != nil {
			return nil, err
		}
	}

	thread, err := s.claimPageThread(ctx, req, subject, definition, actor)
	if err != nil {
		return nil, err
	}
	thread.MarkContinuable()

	return &assistantports.PageThread{Thread: thread, Agent: pageAgentOf(definition)}, nil
}

func pageAgentOf(definition *agentdefinition.Definition) assistantports.PageAgent {
	return assistantports.PageAgent{
		ID:          definition.ID,
		Name:        definition.Name,
		Description: definition.Description,
		Template:    definition.Template,
		Icon:        definition.Icon,
		Accent:      definition.Accent,
		SystemKey:   definition.SystemKey,
		ToolNames:   slices.Clone(definition.ToolNames),
		Starters:    agentstarter.Starters(definition),
	}
}

func checkPageSubject(
	surface *pageSurface,
	req *assistantports.OpenPageThreadRequest,
) (pageSubject, error) {
	subject, ok := surface.subject(req.SubjectType)
	if !ok {
		return pageSubject{}, errortypes.NewValidationError(
			"subjectType", errortypes.ErrInvalid,
			"This conversation is about {0}", surface.nouns(),
		)
	}
	if surface.subjectRequired && req.SubjectID.IsNil() {
		return pageSubject{}, errortypes.NewValidationError(
			"subjectId", errortypes.ErrRequired,
			"This conversation needs the {0} it is about", subject.subjectType.Noun(),
		)
	}
	if req.SubjectID.IsNotNil() {
		if err := subject.subjectType.CheckID(req.SubjectID); err != nil {
			return pageSubject{}, errortypes.NewValidationError(
				"subjectId", errortypes.ErrInvalid, err.Error(),
			)
		}
	}

	return subject, nil
}

func (s *Service) assertPageAgentUsable(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) error {
	if !definition.Enabled {
		return errortypes.NewBusinessError(
			"{0} is turned off. An administrator can turn it on in AI Control.", definition.Name,
		)
	}
	if err := assertChatAgent(definition); err != nil {
		return err
	}

	return s.assertMayUseAgent(ctx, actor, definition)
}

func (s *Service) claimPageThread(
	ctx context.Context,
	req *assistantports.OpenPageThreadRequest,
	subject pageSubject,
	definition *agentdefinition.Definition,
	actor *permissionports.RequestActor,
) (*conversation.Thread, error) {
	subjectType := agent.SubjectType("")
	if req.SubjectID.IsNotNil() {
		subjectType = subject.subjectType
	}

	if req.SubjectID.IsNotNil() {
		existing, err := s.pageThreads.GetPageThread(ctx, assistantports.PageThreadKey{
			TenantInfo:  req.TenantInfo,
			UserID:      actor.UserID,
			Origin:      req.Origin,
			SubjectType: subjectType,
			SubjectID:   req.SubjectID,
		})
		switch {
		case err == nil:
			if existing.AgentDefinitionID == definition.ID {
				return existing, nil
			}
			if _, archiveErr := s.pageThreads.ArchiveSubjectThreads(ctx,
				assistantports.ArchiveSubjectThreadsRequest{
					TenantInfo:  req.TenantInfo,
					Origin:      req.Origin,
					SubjectType: subjectType,
					SubjectID:   req.SubjectID,
				}); archiveErr != nil {
				return nil, archiveErr
			}
		case !errortypes.IsNotFoundError(err):
			return nil, err
		}
	}

	thread := &conversation.Thread{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		UserID:            actor.UserID,
		AgentDefinitionID: definition.ID,
		Title:             subject.title,
		Status:            conversation.ThreadStatusActive,
		Origin:            req.Origin,
		SubjectType:       subjectType,
		SubjectID:         req.SubjectID,
	}
	now := timeutils.NowUnix()
	if mark, tainted := agent.SubjectTaint(subjectType, req.SubjectID.String(), now); tainted {
		thread.AbsorbTaint(&agent.RunTaint{Marks: []agent.TaintMark{mark}}, now)
	}

	multiErr := errortypes.NewMultiError()
	thread.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if req.SubjectID.IsNil() {
		return s.conversations.CreateThread(ctx, thread)
	}

	return s.pageThreads.ClaimPageThread(ctx, thread)
}

func draftFits(
	surface *pageSurface,
	subject pageSubject,
	thread *conversation.Thread,
	draft pagedraft.Surface,
) bool {
	if thread.HasSubject() {
		return draft == subject.draft
	}

	return surface.takesDraft(draft)
}

func (s *Service) assertPageTurn(
	ctx context.Context,
	thread *conversation.Thread,
	page *agentpagecontext.PageContext,
	actor *permissionports.RequestActor,
) error {
	draft := pagedraft.Surface("")
	if page != nil && page.Draft != nil {
		draft = page.Draft.Surface
	}

	if !thread.Origin.PageBound() {
		if draft != "" {
			return errortypes.NewValidationError(
				"context.draft", errortypes.ErrInvalid,
				"Only an import or formula conversation carries a page's draft",
			)
		}

		return nil
	}

	surface := pageSurfaces[thread.Origin]
	subject, _ := surface.subject(thread.SubjectType)
	if thread.Status != conversation.ThreadStatusActive {
		return errortypes.NewBusinessError(
			"This conversation was closed when its {0} changed. Reopen the page to start a new one.",
			subject.subjectType.Noun(),
		)
	}
	if draft != "" && !draftFits(&surface, subject, thread, draft) {
		return errortypes.NewValidationError(
			"context.draft", errortypes.ErrInvalid,
			"This conversation takes the page's {0} draft", string(subject.draft),
		)
	}
	if thread.HasSubject() {
		allowed, err := s.mayReadSubject(ctx, actor, thread.SubjectType)
		if err != nil {
			return err
		}
		if !allowed {
			return errortypes.NewAuthorizationError(
				"You no longer have access to the {0} this conversation is about.",
				thread.SubjectType.Noun(),
			)
		}
	}

	return nil
}
