package agentdefinitionservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentDefinitionRepository
	Tools        services.AgentToolRegistry
	AuditService services.AuditService
}

type Service struct {
	l     *zap.Logger
	repo  repositories.AgentDefinitionRepository
	tools services.AgentToolRegistry
	audit services.AuditService
}

func New(p Params) services.AgentDefinitionService {
	return &Service{
		l:     p.Logger.Named("service.agentdefinition"),
		repo:  p.Repo,
		tools: p.Tools,
		audit: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	req *services.SaveAgentDefinitionRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	definition := &agentdefinition.Definition{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	apply(definition, req)

	if err := s.validate(definition); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, definition)
	if err != nil {
		return nil, err
	}

	s.logAudit(created, nil, permission.OpCreate, actor, "Agent created")

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.SaveAgentDefinitionRequest,
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
	updated.Version = req.Version
	apply(&updated, req)

	if err = s.validate(&updated); err != nil {
		return nil, err
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	s.logAudit(saved, &previous, permission.OpUpdate, actor, "Agent updated")

	return saved, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
	actor *services.RequestActor,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.logAudit(existing, existing, permission.OpDelete, actor, "Agent deleted")

	return nil
}

func (s *Service) Templates() []services.AgentTemplateDescriptor {
	kinds := agentdefinition.AllKinds()
	descriptors := make([]services.AgentTemplateDescriptor, 0, len(kinds))

	for _, kind := range kinds {
		descriptors = append(descriptors, services.AgentTemplateDescriptor{
			Kind:            kind,
			Label:           kind.Label(),
			Description:     kind.Description(),
			MutatingAllowed: kind.MutatingAllowed(),
			AvailableTools:  AvailableTools(kind, s.tools),
		})
	}

	return descriptors
}

// validate runs the domain rules and then the membership check, which needs the
// live registry and so cannot live in the domain.
func (s *Service) validate(definition *agentdefinition.Definition) error {
	multiErr := errortypes.NewMultiError()
	definition.Validate(multiErr)
	validateToolSelection(definition, s.tools, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func apply(definition *agentdefinition.Definition, req *services.SaveAgentDefinitionRequest) {
	definition.Name = strings.TrimSpace(req.Name)
	definition.Description = strings.TrimSpace(req.Description)
	definition.Kind = req.Kind
	definition.Focus = strings.TrimSpace(req.Focus)
	definition.ToolNames = req.ToolNames
	definition.AutonomyCeiling = req.AutonomyCeiling
	definition.Enabled = req.Enabled
}

func (s *Service) logAudit(
	definition *agentdefinition.Definition,
	previous *agentdefinition.Definition,
	operation permission.Operation,
	actor *services.RequestActor,
	comment string,
) {
	auditActor := actor.AuditActor()

	var previousState map[string]any
	if previous != nil {
		previousState = jsonutils.MustToJSON(previous)
	}

	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentDefinition,
		ResourceID:     definition.GetID().String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(definition),
		PreviousState:  previousState,
		OrganizationID: definition.OrganizationID,
		BusinessUnitID: definition.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent definition audit", zap.Error(err))
	}
}
