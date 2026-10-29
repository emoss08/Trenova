package assistantthreads

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/agentaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentstarter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) OpenPageThread(
	ctx context.Context,
	req *assistantports.OpenPageThreadRequest,
	actor *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	surface, ok := agentaccess.PageSurfaces[req.Origin]
	if !ok {
		return nil, errortypes.NewValidationError(
			"origin", errortypes.ErrInvalid, "This page has no assistant conversation",
		)
	}
	subject, err := agentaccess.CheckPageSubject(&surface, req)
	if err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, agentaccess.ErrAgentsWithheld()
	}
	if s.systemAgents == nil || s.pageThreads == nil {
		return nil, errortypes.NewBusinessError(
			"The {0} assistant is not available on this server", subject.Title,
		)
	}

	usable, err := s.access().UsableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, agentaccess.ErrAgentsWithheld()
	}

	definition, err := s.systemAgents.EnsureSystem(
		ctx,
		agentdefinitionports.EnsureSystemAgentRequest{
			TenantInfo: req.TenantInfo,
			SystemKey:  surface.SystemKey,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = s.assertPageAgentUsable(ctx, actor, definition); err != nil {
		return nil, err
	}
	if req.SubjectID.IsNotNil() {
		if err = s.access().AssertSubjectReadable(ctx, actor, subject.SubjectType); err != nil {
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
	if err := agentaccess.AssertChatAgent(definition); err != nil {
		return err
	}

	return s.access().AssertMayUseAgent(ctx, actor, definition)
}

func (s *Service) claimPageThread(
	ctx context.Context,
	req *assistantports.OpenPageThreadRequest,
	subject agentaccess.PageSubject,
	definition *agentdefinition.Definition,
	actor *permissionports.RequestActor,
) (*conversation.Thread, error) {
	subjectType := agent.SubjectType("")
	if req.SubjectID.IsNotNil() {
		subjectType = subject.SubjectType
	}

	if req.SubjectID.IsNotNil() {
		existing, err := s.pageThreads.GetPageThread(ctx, &conversationports.PageThreadKey{
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
				&conversationports.ArchiveSubjectThreadsRequest{
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
		Title:             subject.Title,
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
