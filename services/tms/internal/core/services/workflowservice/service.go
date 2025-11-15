package workflowservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.WorkflowRepository
	AuditService   services.AuditService
	TriggerService *TriggerService `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	repo           repositories.WorkflowRepository
	as             services.AuditService
	triggerService *TriggerService
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:              p.Logger.Named("service.workflow"),
		repo:           p.Repo,
		as:             p.AuditService,
		triggerService: p.TriggerService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListWorkflowRequest,
) (*pagination.ListResult[*workflow.Workflow], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetWorkflowByIDRequest,
) (*workflow.Workflow, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *workflow.Workflow,
	userID pulid.ID,
) (*workflow.Workflow, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	// Validate
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Set created by
	entity.CreatedBy = userID

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create workflow", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Workflow created"),
	)
	if err != nil {
		log.Error("failed to log workflow creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *workflow.Workflow,
	userID pulid.ID,
) (*workflow.Workflow, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("workflowID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	// Validate
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Get original
	original, err := s.repo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     entity.ID,
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		log.Error("failed to get original workflow", zap.Error(err))
		return nil, err
	}

	// Check if editable
	if !original.IsEditable() {
		return nil, errortypes.NewBusinessError("Workflow is archived and cannot be edited")
	}

	// Set updated by
	entity.UpdatedBy = &userID

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update workflow", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Workflow updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log workflow update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("workflowID", id.String()),
		zap.String("userID", userID.String()),
	)

	// Get workflow to check if it can be deleted
	wf, err := s.repo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	// Don't allow deletion of active workflows
	if wf.Status == workflow.WorkflowStatusActive {
		return errortypes.NewBusinessError(
			"Cannot delete active workflow. Please deactivate it first.",
		)
	}

	err = s.repo.Delete(ctx, id, orgID, buID)
	if err != nil {
		log.Error("failed to delete workflow", zap.Error(err))
		return err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     id.String(),
			Operation:      permission.OpDelete,
			UserID:         userID,
			PreviousState:  jsonutils.MustToJSON(wf),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("Workflow deleted"),
	)
	if err != nil {
		log.Error("failed to log workflow deletion", zap.Error(err))
	}

	return nil
}

// Version Management

func (s *Service) CreateVersion(
	ctx context.Context,
	workflowID, orgID, buID, userID pulid.ID,
	versionName, changelog string,
	workflowDefinition any,
) (*workflow.WorkflowVersion, error) {
	log := s.l.With(
		zap.String("operation", "CreateVersion"),
		zap.String("workflowID", workflowID.String()),
		zap.String("userID", userID.String()),
	)

	// Get workflow to ensure it exists
	wf, err := s.repo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     workflowID,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	// Check if workflow is editable
	if !wf.IsEditable() {
		return nil, errortypes.NewBusinessError("Workflow is archived and cannot be modified")
	}

	// Get latest version to determine next version number
	var versionNumber int
	latestVersion, err := s.repo.GetLatestVersion(ctx, workflowID, orgID, buID)
	if err != nil {
		// If no versions exist, start with version 1
		versionNumber = 1
	} else {
		versionNumber = latestVersion.VersionNumber + 1
	}

	// Create version
	version := &workflow.WorkflowVersion{
		WorkflowID:         workflowID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		VersionNumber:      versionNumber,
		VersionName:        versionName,
		Changelog:          changelog,
		WorkflowDefinition: jsonutils.MustToJSON(workflowDefinition),
		CreatedBy:          userID,
		CreatedAt:          time.Now().Unix(),
	}

	// Validate
	multiErr := errortypes.NewMultiError()
	version.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	createdVersion, err := s.repo.CreateVersion(ctx, version)
	if err != nil {
		log.Error("failed to create workflow version", zap.Error(err))
		return nil, err
	}

	// Update workflow's current_version_id
	wf.CurrentVersionID = &createdVersion.ID
	wf.UpdatedBy = &userID
	_, err = s.repo.Update(ctx, wf)
	if err != nil {
		log.Error("failed to update workflow current version", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     workflowID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdVersion),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment(fmt.Sprintf("Created version %d: %s", versionNumber, versionName)),
	)
	if err != nil {
		log.Error("failed to log version creation", zap.Error(err))
	}

	return createdVersion, nil
}

func (s *Service) PublishVersion(
	ctx context.Context,
	workflowID, versionID, orgID, buID, userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "PublishVersion"),
		zap.String("workflowID", workflowID.String()),
		zap.String("versionID", versionID.String()),
		zap.String("userID", userID.String()),
	)

	// Get workflow
	wf, err := s.repo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     workflowID,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	// Check if workflow is editable
	if !wf.IsEditable() {
		return errortypes.NewBusinessError("Workflow is archived and cannot be published")
	}

	// Get version to ensure it exists
	version, err := s.repo.GetVersionByID(ctx, versionID, orgID, buID)
	if err != nil {
		return err
	}

	// Ensure version belongs to this workflow
	if version.WorkflowID != workflowID {
		return errortypes.NewBusinessError("Version does not belong to this workflow")
	}

	// Publish version
	err = s.repo.PublishVersion(ctx, workflowID, versionID, orgID, buID, userID)
	if err != nil {
		log.Error("failed to publish version", zap.Error(err))
		return err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     workflowID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment(fmt.Sprintf("Published version %d", version.VersionNumber)),
	)
	if err != nil {
		log.Error("failed to log version publish", zap.Error(err))
	}

	return nil
}

func (s *Service) GetVersions(
	ctx context.Context,
	workflowID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowVersion, error) {
	return s.repo.GetVersionsByWorkflowID(ctx, workflowID, orgID, buID)
}

func (s *Service) GetVersion(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) (*workflow.WorkflowVersion, error) {
	return s.repo.GetVersionByID(ctx, versionID, orgID, buID)
}

// Status Management

func (s *Service) UpdateStatus(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
	status workflow.WorkflowStatus,
) error {
	log := s.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("workflowID", id.String()),
		zap.String("status", status.String()),
		zap.String("userID", userID.String()),
	)

	// Get workflow
	wf, err := s.repo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	// Validate status transition
	if status == workflow.WorkflowStatusActive && !wf.IsPublished() {
		return errortypes.NewBusinessError("Cannot activate workflow without a published version")
	}

	oldStatus := wf.Status

	err = s.repo.UpdateStatus(ctx, id, orgID, buID, status)
	if err != nil {
		log.Error("failed to update workflow status", zap.Error(err))
		return err
	}

	// Handle trigger setup/teardown for scheduled workflows
	if s.triggerService != nil && wf.TriggerType == workflow.TriggerTypeScheduled {
		// Refresh workflow to get updated status
		wf.Status = status

		if status == workflow.WorkflowStatusActive {
			// Setup scheduled trigger
			var triggerConfig workflow.ScheduledTriggerConfig
			if err := jsonutils.MustToJSON(wf.TriggerConfig); err == nil {
				if err := s.triggerService.SetupScheduledTrigger(ctx, wf, &triggerConfig); err != nil {
					log.Error("failed to setup scheduled trigger", zap.Error(err))
					// Don't fail the status update, just log the error
				} else {
					log.Info("scheduled trigger setup successfully")
				}
			} else {
				log.Error("failed to parse trigger config", zap.Error(errors.New("failed to parse trigger config")))
			}
		} else if status == workflow.WorkflowStatusInactive {
			// Pause scheduled trigger
			if err := s.triggerService.PauseScheduledTrigger(ctx, id); err != nil {
				log.Error("failed to pause scheduled trigger", zap.Error(err))
			} else {
				log.Info("scheduled trigger paused successfully")
			}
		} else if status == workflow.WorkflowStatusArchived {
			// Remove scheduled trigger completely
			if err := s.triggerService.RemoveScheduledTrigger(ctx, id); err != nil {
				log.Error("failed to remove scheduled trigger", zap.Error(err))
			} else {
				log.Info("scheduled trigger removed successfully")
			}
		}
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       workflow.ResourceWorkflow,
			ResourceID:     id.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment(fmt.Sprintf("Status changed from %s to %s", oldStatus, status)),
	)
	if err != nil {
		log.Error("failed to log status update", zap.Error(err))
	}

	return nil
}

func (s *Service) Activate(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	return s.UpdateStatus(ctx, id, orgID, buID, userID, workflow.WorkflowStatusActive)
}

func (s *Service) Deactivate(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	return s.UpdateStatus(ctx, id, orgID, buID, userID, workflow.WorkflowStatusInactive)
}

func (s *Service) Archive(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	return s.UpdateStatus(ctx, id, orgID, buID, userID, workflow.WorkflowStatusArchived)
}

// Node and Edge Management

type SaveWorkflowDefinitionRequest struct {
	WorkflowID pulid.ID
	VersionID  pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	Nodes      []*workflow.WorkflowNode
	Edges      []*workflow.WorkflowEdge
}

func (s *Service) SaveWorkflowDefinition(
	ctx context.Context,
	req *SaveWorkflowDefinitionRequest,
) error {
	log := s.l.With(
		zap.String("operation", "SaveWorkflowDefinition"),
		zap.String("workflowID", req.WorkflowID.String()),
		zap.String("versionID", req.VersionID.String()),
	)

	// Validate all nodes
	multiErr := errortypes.NewMultiError()
	for _, node := range req.Nodes {
		node.WorkflowVersionID = req.VersionID
		node.OrganizationID = req.OrgID
		node.BusinessUnitID = req.BuID
		node.CreatedAt = time.Now().Unix()

		node.Validate(multiErr)
	}

	// Validate all edges
	for _, edge := range req.Edges {
		edge.WorkflowVersionID = req.VersionID
		edge.OrganizationID = req.OrgID
		edge.BusinessUnitID = req.BuID
		edge.CreatedAt = time.Now().Unix()

		edge.Validate(multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	// Delete existing nodes and edges for this version
	if err := s.repo.DeleteNodesByVersionID(ctx, req.VersionID, req.OrgID, req.BuID); err != nil {
		log.Error("failed to delete existing nodes", zap.Error(err))
		return err
	}

	if err := s.repo.DeleteEdgesByVersionID(ctx, req.VersionID, req.OrgID, req.BuID); err != nil {
		log.Error("failed to delete existing edges", zap.Error(err))
		return err
	}

	// Create new nodes
	if err := s.repo.CreateNodes(ctx, req.Nodes); err != nil {
		log.Error("failed to create nodes", zap.Error(err))
		return err
	}

	// Create new edges
	if err := s.repo.CreateEdges(ctx, req.Edges); err != nil {
		log.Error("failed to create edges", zap.Error(err))
		return err
	}

	return nil
}
