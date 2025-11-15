package workflowservice

import (
	"context"
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

type ExecutionServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkflowExecutionRepository
	WorkflowRepo repositories.WorkflowRepository
	AuditService services.AuditService
}

type ExecutionService struct {
	l      *zap.Logger
	repo   repositories.WorkflowExecutionRepository
	wfRepo repositories.WorkflowRepository
	as     services.AuditService
}

func NewExecutionService(p ExecutionServiceParams) *ExecutionService {
	return &ExecutionService{
		l:      p.Logger.Named("service.workflow-execution"),
		repo:   p.Repo,
		wfRepo: p.WorkflowRepo,
		as:     p.AuditService,
	}
}

func (s *ExecutionService) List(
	ctx context.Context,
	req *repositories.ListWorkflowExecutionRequest,
) (*pagination.ListResult[*workflow.WorkflowExecution], error) {
	return s.repo.List(ctx, req)
}

func (s *ExecutionService) Get(
	ctx context.Context,
	req repositories.GetWorkflowExecutionByIDRequest,
) (*workflow.WorkflowExecution, error) {
	return s.repo.GetByID(ctx, req)
}

type TriggerWorkflowRequest struct {
	WorkflowID  pulid.ID
	OrgID       pulid.ID
	BuID        pulid.ID
	UserID      pulid.ID
	TriggerData map[string]any
}

func (s *ExecutionService) TriggerWorkflow(
	ctx context.Context,
	req *TriggerWorkflowRequest,
) (*workflow.WorkflowExecution, error) {
	log := s.l.With(
		zap.String("operation", "TriggerWorkflow"),
		zap.String("workflowID", req.WorkflowID.String()),
		zap.String("userID", req.UserID.String()),
	)

	// Get workflow
	wf, err := s.wfRepo.GetByID(ctx, repositories.GetWorkflowByIDRequest{
		ID:     req.WorkflowID,
		OrgID:  req.OrgID,
		BuID:   req.BuID,
		UserID: req.UserID,
	})
	if err != nil {
		log.Error("failed to get workflow", zap.Error(err))
		return nil, err
	}

	// Check if workflow can be executed
	if !wf.CanExecute() {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf("Workflow cannot be executed. Status: %s, Published: %v",
				wf.Status, wf.IsPublished()),
		)
	}

	// Get published version
	publishedVersion, err := s.wfRepo.GetVersionByID(
		ctx,
		*wf.PublishedVersionID,
		req.OrgID,
		req.BuID,
	)
	if err != nil {
		log.Error("failed to get published version", zap.Error(err))
		return nil, err
	}

	// Create execution
	execution := &workflow.WorkflowExecution{
		OrganizationID:    req.OrgID,
		BusinessUnitID:    req.BuID,
		WorkflowID:        req.WorkflowID,
		WorkflowVersionID: publishedVersion.ID,
		Status:            workflow.ExecutionStatusPending,
		TriggerType:       workflow.TriggerTypeManual,
		TriggerData:       jsonutils.MustToJSON(req.TriggerData),
		TriggeredBy:       &req.UserID,
		MaxRetries:        wf.MaxRetries,
		CreatedAt:         time.Now().Unix(),
		UpdatedAt:         time.Now().Unix(),
	}

	// Validate
	multiErr := errortypes.NewMultiError()
	execution.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	createdExecution, err := s.repo.Create(ctx, execution)
	if err != nil {
		log.Error("failed to create execution", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowExecution,
			ResourceID:     createdExecution.GetID(),
			Operation:      permission.OpCreate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(createdExecution),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment(fmt.Sprintf("Workflow execution triggered for workflow: %s", wf.Name)),
	)
	if err != nil {
		log.Error("failed to log execution creation", zap.Error(err))
	}

	return createdExecution, nil
}

func (s *ExecutionService) UpdateStatus(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	status workflow.ExecutionStatus,
) error {
	log := s.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("executionID", id.String()),
		zap.String("status", status.String()),
	)

	err := s.repo.UpdateStatus(ctx, id, orgID, buID, status)
	if err != nil {
		log.Error("failed to update execution status", zap.Error(err))
		return err
	}

	return nil
}

func (s *ExecutionService) CancelExecution(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "CancelExecution"),
		zap.String("executionID", id.String()),
		zap.String("userID", userID.String()),
	)

	// Get execution
	execution, err := s.repo.GetByID(ctx, repositories.GetWorkflowExecutionByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	// Check if execution can be canceled
	if execution.IsComplete() {
		return errortypes.NewBusinessError("Execution is already complete and cannot be canceled")
	}

	err = s.repo.CancelExecution(ctx, id, orgID, buID)
	if err != nil {
		log.Error("failed to cancel execution", zap.Error(err))
		return err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowExecution,
			ResourceID:     id.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("Workflow execution canceled"),
	)
	if err != nil {
		log.Error("failed to log execution cancellation", zap.Error(err))
	}

	return nil
}

func (s *ExecutionService) RetryExecution(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) (*workflow.WorkflowExecution, error) {
	log := s.l.With(
		zap.String("operation", "RetryExecution"),
		zap.String("executionID", id.String()),
		zap.String("userID", userID.String()),
	)

	// Get original execution
	original, err := s.repo.GetByID(ctx, repositories.GetWorkflowExecutionByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	// Check if execution can be retried
	if !original.CanRetry() {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf("Execution cannot be retried. Status: %s, Retry count: %d/%d",
				original.Status, original.RetryCount, original.MaxRetries),
		)
	}

	// Create new execution (retry)
	newExecution := &workflow.WorkflowExecution{
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		WorkflowID:        original.WorkflowID,
		WorkflowVersionID: original.WorkflowVersionID,
		Status:            workflow.ExecutionStatusPending,
		TriggerType:       original.TriggerType,
		TriggerData:       original.TriggerData,
		TriggeredBy:       &userID,
		RetryCount:        original.RetryCount + 1,
		MaxRetries:        original.MaxRetries,
		CreatedAt:         time.Now().Unix(),
		UpdatedAt:         time.Now().Unix(),
	}

	createdExecution, err := s.repo.Create(ctx, newExecution)
	if err != nil {
		log.Error("failed to create retry execution", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowExecution,
			ResourceID:     createdExecution.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdExecution),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment(
			fmt.Sprintf("Retry of execution %s (attempt %d)", original.ID, newExecution.RetryCount),
		),
	)
	if err != nil {
		log.Error("failed to log retry execution", zap.Error(err))
	}

	return createdExecution, nil
}

// Step Management

func (s *ExecutionService) CreateStep(
	ctx context.Context,
	step *workflow.WorkflowExecutionStep,
) (*workflow.WorkflowExecutionStep, error) {
	// Validate
	multiErr := errortypes.NewMultiError()
	step.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.CreateStep(ctx, step)
}

func (s *ExecutionService) UpdateStep(
	ctx context.Context,
	step *workflow.WorkflowExecutionStep,
) (*workflow.WorkflowExecutionStep, error) {
	// Validate
	multiErr := errortypes.NewMultiError()
	step.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.UpdateStep(ctx, step)
}

func (s *ExecutionService) GetSteps(
	ctx context.Context,
	executionID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowExecutionStep, error) {
	return s.repo.GetStepsByExecutionID(ctx, executionID, orgID, buID)
}
