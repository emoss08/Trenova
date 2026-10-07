package aituneupservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) Apply(
	ctx context.Context,
	req *services.AITuneUpDecisionRequest,
	actor *services.RequestActor,
) (*services.AITuneUpView, error) {
	tuneUp, err := s.offered(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = s.act(ctx, req.TenantInfo, tuneUp, actor); err != nil {
		return nil, err
	}

	decided, err := s.decide(ctx, req, actor, aituneup.StatusApplied, nil)
	if err != nil {
		return nil, err
	}
	return s.single(ctx, req.TenantInfo, decided)
}

func (s *Service) Dismiss(
	ctx context.Context,
	req *services.AITuneUpDecisionRequest,
	actor *services.RequestActor,
) (*services.AITuneUpView, error) {
	if _, err := s.offered(ctx, req); err != nil {
		return nil, err
	}

	until := s.now() + int64(aituneup.ClampDismissDays(req.Days))*secondsPerDay
	decided, err := s.decide(ctx, req, actor, aituneup.StatusDismissed, &until)
	if err != nil {
		return nil, err
	}
	return s.single(ctx, req.TenantInfo, decided)
}

func (s *Service) Restore(
	ctx context.Context,
	req *services.AITuneUpDecisionRequest,
	_ *services.RequestActor,
) (*services.AITuneUpView, error) {
	restored, err := s.repo.Reopen(ctx, &repositories.ReopenAITuneUpRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		Version:    req.Version,
	})
	if err != nil {
		return nil, err
	}
	return s.single(ctx, req.TenantInfo, restored)
}

func (s *Service) offered(
	ctx context.Context,
	req *services.AITuneUpDecisionRequest,
) (*aituneup.TuneUp, error) {
	tuneUp, err := s.repo.GetByID(ctx, repositories.GetAITuneUpRequest{TenantInfo: req.TenantInfo, ID: req.ID})
	if err != nil {
		return nil, err
	}
	if !tuneUp.Visible(s.now()) {
		return nil, errortypes.NewBusinessError("This tune-up was already applied or put away")
	}
	if tuneUp.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("AITuneUp", req.ID.String())
	}
	return tuneUp, nil
}

func (s *Service) decide(
	ctx context.Context,
	req *services.AITuneUpDecisionRequest,
	actor *services.RequestActor,
	status aituneup.Status,
	dismissedUntil *int64,
) (*aituneup.TuneUp, error) {
	decision := &repositories.DecideAITuneUpRequest{
		TenantInfo:     req.TenantInfo,
		ID:             req.ID,
		Version:        req.Version,
		Status:         status,
		DismissedUntil: dismissedUntil,
		DecidedByID:    actor.AuditActor().UserID,
		DecidedAt:      s.now(),
	}
	decided, err := s.repo.Decide(ctx, decision)
	if err == nil || !errortypes.IsVersionMismatchError(err) {
		return decided, err
	}

	current, getErr := s.repo.GetByID(ctx, repositories.GetAITuneUpRequest{TenantInfo: req.TenantInfo, ID: req.ID})
	if getErr != nil {
		return nil, getErr
	}
	decision.Version = current.Version
	return s.repo.Decide(ctx, decision)
}

func (s *Service) single(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tuneUp *aituneup.TuneUp,
) (*services.AITuneUpView, error) {
	views, err := s.describe(ctx, tenantInfo, []*aituneup.TuneUp{tuneUp})
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return &services.AITuneUpView{TuneUp: tuneUp}, nil
	}
	return views[0], nil
}

func (s *Service) act(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tuneUp *aituneup.TuneUp,
	actor *services.RequestActor,
) error {
	switch tuneUp.Kind {
	case aituneup.KindRaiseToolTier:
		return s.raiseToolTier(ctx, tenantInfo, tuneUp, actor)
	case aituneup.KindLeaveShadow:
		return s.patchAgent(ctx, tenantInfo, tuneUp.AgentDefinitionID, actor,
			"Taken out of shadow from a tune-up",
			func(definition *agentdefinition.Definition) error {
				definition.ShadowMode = false
				return nil
			})
	case aituneup.KindTurnOffIdleAgent:
		return s.patchAgent(ctx, tenantInfo, tuneUp.AgentDefinitionID, actor,
			"Turned off from a tune-up: nobody had asked it anything",
			func(definition *agentdefinition.Definition) error {
				definition.Enabled = false
				return nil
			})
	case aituneup.KindReorderProviders:
		return s.putAhead(ctx, tenantInfo, tuneUp.ProviderID, tuneUp.OtherProviderID, actor)
	case aituneup.KindAssignTask:
		_, err := s.providerService.AssignTask(ctx, &services.AssignAIProviderTaskRequest{
			TenantInfo: tenantInfo,
			ProviderID: tuneUp.ProviderID,
			Task:       tuneUp.Task,
		}, actor)
		return err
	default:
		return errortypes.NewBusinessError("Unknown tune-up")
	}
}

func (s *Service) raiseToolTier(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tuneUp *aituneup.TuneUp,
	actor *services.RequestActor,
) error {
	control, err := s.control.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return err
	}
	_, err = s.trust.PromoteTool(ctx, &services.PromoteToolRequest{
		PromoteReadyRequest: services.PromoteReadyRequest{
			TenantInfo: tenantInfo,
			Threshold:  control.PromotionThreshold,
			DecidedBy:  actor.AuditActor().UserID,
		},
		AgentDefinitionID: tuneUp.AgentDefinitionID,
		ToolName:          tuneUp.ToolName,
	})
	return err
}

func (s *Service) patchAgent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	agentID pulid.ID,
	actor *services.RequestActor,
	comment string,
	edit func(*agentdefinition.Definition) error,
) error {
	_, err := s.agentDefinitions.Patch(ctx, &services.PatchAgentDefinitionRequest{
		ID:         agentID,
		TenantInfo: tenantInfo,
		Comment:    comment,
		Edit:       edit,
	}, actor)
	return err
}

func (s *Service) putAhead(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moving, ahead pulid.ID,
	actor *services.RequestActor,
) error {
	providers, err := s.providers.ListOrdered(ctx, tenantInfo)
	if err != nil {
		return err
	}
	order := make([]pulid.ID, 0, len(providers))
	for _, provider := range providers {
		order = append(order, provider.ID)
	}
	next, ok := MoveAhead(order, moving, ahead)
	if !ok {
		return errortypes.NewBusinessError("A provider in this tune-up no longer exists")
	}
	_, err = s.providerService.Reorder(ctx, &services.ReorderAIProvidersRequest{
		TenantInfo:  tenantInfo,
		ProviderIDs: next,
	}, actor)
	return err
}

func MoveAhead(order []pulid.ID, moving, ahead pulid.ID) ([]pulid.ID, bool) {
	from := slices.Index(order, moving)
	to := slices.Index(order, ahead)
	if from < 0 || to < 0 {
		return nil, false
	}
	if from < to {
		return slices.Clone(order), true
	}
	next := slices.Delete(slices.Clone(order), from, from+1)
	return slices.Insert(next, to, moving), true
}
