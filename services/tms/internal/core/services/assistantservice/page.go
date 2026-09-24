package assistantservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type pageSurface struct {
	systemKey       string
	subjectType     agent.SubjectType
	subjectRequired bool
	draft           pagedraft.Surface
	title           string
}

var pageSurfaces = map[conversation.ThreadOrigin]pageSurface{
	conversation.ThreadOriginImport: {
		systemKey:       agentdefinition.SystemKeyImportAssistant,
		subjectType:     agent.SubjectDocument,
		subjectRequired: true,
		draft:           pagedraft.SurfaceShipmentImport,
		title:           "Shipment import",
	},
	conversation.ThreadOriginFormula: {
		systemKey:   agentdefinition.SystemKeyFormulaAssistant,
		subjectType: agent.SubjectFormulaTemplate,
		draft:       pagedraft.SurfaceFormula,
		title:       "Formula",
	},
}

func (s *Service) OpenPageThread(
	ctx context.Context,
	req *services.OpenPageThreadRequest,
	actor *services.RequestActor,
) (*services.PageThread, error) {
	surface, ok := pageSurfaces[req.Origin]
	if !ok {
		return nil, errortypes.NewValidationError(
			"origin", errortypes.ErrInvalid, "This page has no assistant conversation",
		)
	}
	if err := checkPageSubject(surface, req); err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errAgentsWithheld()
	}
	if s.systemAgents == nil || s.pageThreads == nil {
		return nil, errortypes.NewBusinessError(
			"The {0} assistant is not available on this server", surface.title,
		)
	}

	usable, err := s.usableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, errAgentsWithheld()
	}

	definition, err := s.systemAgents.EnsureSystem(ctx, services.EnsureSystemAgentRequest{
		TenantInfo: req.TenantInfo,
		SystemKey:  surface.systemKey,
	})
	if err != nil {
		return nil, err
	}
	if err = s.assertPageAgentUsable(ctx, actor, definition); err != nil {
		return nil, err
	}
	if req.SubjectID.IsNotNil() {
		if err = s.assertSubjectReadable(ctx, actor, surface.subjectType); err != nil {
			return nil, err
		}
	}

	thread, err := s.claimPageThread(ctx, req, surface, definition, actor)
	if err != nil {
		return nil, err
	}
	thread.MarkContinuable()

	return &services.PageThread{Thread: thread, Agent: pageAgentOf(definition)}, nil
}

func pageAgentOf(definition *agentdefinition.Definition) services.PageAgent {
	return services.PageAgent{
		ID:          definition.ID,
		Name:        definition.Name,
		Description: definition.Description,
		Template:    definition.Template,
		Icon:        definition.Icon,
		Accent:      definition.Accent,
		SystemKey:   definition.SystemKey,
		ToolNames:   slices.Clone(definition.ToolNames),
		Starters:    definition.Starters(),
	}
}

func checkPageSubject(surface pageSurface, req *services.OpenPageThreadRequest) error {
	if req.SubjectType != "" && req.SubjectType != surface.subjectType {
		return errortypes.NewValidationError(
			"subjectType", errortypes.ErrInvalid,
			"This conversation is about a {0}", surface.subjectType.Noun(),
		)
	}
	if surface.subjectRequired && req.SubjectID.IsNil() {
		return errortypes.NewValidationError(
			"subjectId", errortypes.ErrRequired,
			"This conversation needs the {0} it is about", surface.subjectType.Noun(),
		)
	}
	if req.SubjectID.IsNotNil() {
		if err := surface.subjectType.CheckID(req.SubjectID); err != nil {
			return errortypes.NewValidationError("subjectId", errortypes.ErrInvalid, err.Error())
		}
	}

	return nil
}

func (s *Service) assertPageAgentUsable(
	ctx context.Context,
	actor *services.RequestActor,
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
	req *services.OpenPageThreadRequest,
	surface pageSurface,
	definition *agentdefinition.Definition,
	actor *services.RequestActor,
) (*conversation.Thread, error) {
	subjectType := agent.SubjectType("")
	if req.SubjectID.IsNotNil() {
		subjectType = surface.subjectType
	}

	if req.SubjectID.IsNotNil() {
		existing, err := s.pageThreads.GetPageThread(ctx, repositories.PageThreadKey{
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
				repositories.ArchiveSubjectThreadsRequest{
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
		Title:             surface.title,
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

func (s *Service) ClosePageThreads(
	ctx context.Context,
	req repositories.ArchiveSubjectThreadsRequest,
) error {
	if s.pageThreads == nil {
		return nil
	}
	if _, err := s.pageThreads.ArchiveSubjectThreads(ctx, req); err != nil {
		return fmt.Errorf("close the conversations about %s %s: %w",
			req.SubjectType.Noun(), req.SubjectID, err)
	}

	return nil
}

func (s *Service) assertPageTurn(
	ctx context.Context,
	thread *conversation.Thread,
	page *agent.PageContext,
	actor *services.RequestActor,
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
	if thread.Status != conversation.ThreadStatusActive {
		return errortypes.NewBusinessError(
			"This conversation was closed when its {0} changed. Reopen the page to start a new one.",
			surface.subjectType.Noun(),
		)
	}
	if draft != "" && draft != surface.draft {
		return errortypes.NewValidationError(
			"context.draft", errortypes.ErrInvalid,
			"This conversation takes the page's {0} draft", string(surface.draft),
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
