package agentdefinitionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

// Patch changes a few of an agent's settings without a full form behind the
// request: the capabilities page turns a tool off or a limit down and leaves
// everything else as it is. The change goes through the same validation,
// data-access check, schedule and audit as a full update.
func (s *Service) Patch(
	ctx context.Context,
	req *services.PatchAgentDefinitionRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	existing, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *existing
	updated := *existing
	if req.Version != 0 {
		updated.Version = req.Version
	}
	if req.Edit != nil {
		if err = req.Edit(&updated); err != nil {
			return nil, err
		}
	}
	updated.ForgetReheldTools()
	updated.PruneDelegateTopics()
	updated.NoteEnabledChange(previous.Enabled, actorUser(actor), timeutils.NowUnix())

	if err = s.validate(ctx, &updated, &previous); err != nil {
		return nil, err
	}
	if err = s.checkDataAccess(ctx, &updated, &previous, actor); err != nil {
		return nil, err
	}
	if err = s.requireAutomation(ctx, req.TenantInfo, &updated, &previous); err != nil {
		return nil, err
	}
	if scheduleChanged(&previous, &updated) {
		if err = s.schedule(&updated); err != nil {
			return nil, err
		}
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	if scheduleChanged(&previous, saved) || !typeutils.EqualPtr(previous.EndsAt, saved.EndsAt) {
		s.schedules.Sync(ctx, saved)
	}
	comment := req.Comment
	if comment == "" {
		comment = "Agent updated"
	}
	s.logAudit(saved, &previous, permission.OpUpdate, actor, comment)

	return saved, nil
}
