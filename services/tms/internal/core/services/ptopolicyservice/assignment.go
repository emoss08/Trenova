package ptopolicyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const dayInSeconds int64 = 86400

type AssignWorkersRequest struct {
	TenantInfo      pagination.TenantInfo
	PolicyID        pulid.ID
	WorkerIDs       []pulid.ID
	EffectiveFrom   int64
	OpeningBalances []ptoledgerservice.OpeningBalance
	Note            string
	UserID          pulid.ID
	SystemActor     bool
}

type AssignWorkerFailure struct {
	WorkerID pulid.ID `json:"workerId"`
	Error    string   `json:"error"`
}

type AssignWorkersResult struct {
	Assignments []*worker.WorkerPTOPolicyAssignment `json:"assignments"`
	Failures    []AssignWorkerFailure               `json:"failures"`
}

type EndAssignmentRequest struct {
	TenantInfo   pagination.TenantInfo
	AssignmentID pulid.ID
	EffectiveTo  int64
	Version      int64
	UserID       pulid.ID
}

func (s *Service) ListAssignments(
	ctx context.Context,
	req *repositories.ListPTOAssignmentsRequest,
) ([]*worker.WorkerPTOPolicyAssignment, error) {
	return s.repo.ListAssignments(ctx, req)
}

func (s *Service) ActiveAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerPTOPolicyAssignment, error) {
	assignment, err := s.repo.GetActiveAssignment(ctx, &repositories.GetPTOAssignmentRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		IncludePolicy: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return assignment, nil
}

func (s *Service) AssignWorkers(
	ctx context.Context,
	req *AssignWorkersRequest,
) (*AssignWorkersResult, error) {
	log := s.l.With(
		zap.String("operation", "AssignWorkers"),
		zap.String("policyId", req.PolicyID.String()),
	)

	if len(req.WorkerIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"workerIds",
			errortypes.ErrRequired,
			"Select at least one worker",
		)
	}
	if req.EffectiveFrom <= 0 {
		return nil, errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrRequired,
			"Effective date is required",
		)
	}

	policy, err := s.repo.GetByID(ctx, &repositories.GetPTOPolicyByIDRequest{
		ID:           req.PolicyID,
		TenantInfo:   req.TenantInfo,
		IncludeRules: true,
	})
	if err != nil {
		return nil, err
	}
	if !policy.IsActive() {
		return nil, errortypes.NewValidationError(
			"ptoPolicyId",
			errortypes.ErrInvalidOperation,
			"Only active policies can be assigned",
		)
	}
	for _, ob := range req.OpeningBalances {
		if policy.RuleFor(ob.PTOType) == nil {
			return nil, errortypes.NewValidationError(
				"openingBalances",
				errortypes.ErrInvalid,
				fmt.Sprintf("%s is not tracked by this policy", ob.PTOType),
			)
		}
		if ob.Days.IsNegative() {
			return nil, errortypes.NewValidationError(
				"openingBalances",
				errortypes.ErrInvalid,
				"Opening balances cannot be negative",
			)
		}
	}

	actor := ptoledgerservice.UserActor(req.UserID)
	if req.SystemActor {
		actor = ptoledgerservice.SystemActor()
	}

	result := &AssignWorkersResult{
		Assignments: make([]*worker.WorkerPTOPolicyAssignment, 0, len(req.WorkerIDs)),
	}
	for _, workerID := range req.WorkerIDs {
		assignment, assignErr := s.assignWorker(ctx, req, policy, workerID, actor)
		if assignErr != nil {
			log.Warn("failed to assign worker to PTO policy",
				zap.String("workerId", workerID.String()), zap.Error(assignErr))
			result.Failures = append(result.Failures, AssignWorkerFailure{
				WorkerID: workerID,
				Error:    assignErr.Error(),
			})
			continue
		}
		result.Assignments = append(result.Assignments, assignment)
	}

	if len(result.Assignments) > 0 && !req.SystemActor {
		entries := make([]services.BulkLogEntry, 0, len(result.Assignments))
		for _, assignment := range result.Assignments {
			entries = append(entries, services.BulkLogEntry{
				Params: &services.LogActionParams{
					Resource:       permission.ResourcePTOPolicy,
					ResourceID:     assignment.PTOPolicyID.String(),
					Operation:      permission.OpAssign,
					UserID:         req.UserID,
					CurrentState:   jsonutils.MustToJSON(assignment),
					OrganizationID: assignment.OrganizationID,
					BusinessUnitID: assignment.BusinessUnitID,
				},
				Options: []services.LogOption{auditservice.WithComment(
					fmt.Sprintf(
						"Worker %s assigned to PTO policy %s",
						assignment.WorkerID,
						policy.Code,
					),
				)},
			})
		}
		if err = s.auditService.LogActions(entries); err != nil {
			log.Error("failed to log audit actions", zap.Error(err))
		}
	}

	return result, nil
}

func (s *Service) assignWorker(
	ctx context.Context,
	req *AssignWorkersRequest,
	policy *worker.PTOPolicy,
	workerID pulid.ID,
	actor ptoledgerservice.Actor,
) (*worker.WorkerPTOPolicyAssignment, error) {
	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         workerID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	var created *worker.WorkerPTOPolicyAssignment
	err := s.ledger.DB().
		WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
			open, err := s.repo.GetActiveAssignment(txCtx, &repositories.GetPTOAssignmentRequest{
				TenantInfo: req.TenantInfo,
				WorkerID:   workerID,
			})
			if err != nil && !errortypes.IsNotFoundError(err) {
				return err
			}
			if open != nil {
				if open.PTOPolicyID == policy.ID {
					return errortypes.NewValidationError(
						"ptoPolicyId",
						errortypes.ErrInvalidOperation,
						"Worker is already assigned to this policy",
					)
				}
				if req.EffectiveFrom <= open.EffectiveFrom {
					return errortypes.NewValidationError(
						"effectiveFrom",
						errortypes.ErrInvalid,
						"Effective date must be after the current assignment started",
					)
				}
				endAt := req.EffectiveFrom
				open.EffectiveTo = &endAt
				if _, err = s.repo.UpdateAssignment(txCtx, open); err != nil {
					return err
				}
			}

			entity := &worker.WorkerPTOPolicyAssignment{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				WorkerID:       workerID,
				PTOPolicyID:    policy.ID,
				EffectiveFrom:  req.EffectiveFrom,
				AssignedByID:   req.UserID,
				Note:           req.Note,
			}
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			if multiErr.HasErrors() {
				return multiErr
			}

			created, err = s.repo.CreateAssignment(txCtx, entity)
			if err != nil {
				return err
			}
			if err = s.ledger.EnsureBalances(txCtx, req.TenantInfo, workerID, policy); err != nil {
				return err
			}
			return s.ledger.PostOpeningBalances(txCtx, created, req.OpeningBalances, actor)
		})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) EndAssignment(
	ctx context.Context,
	req *EndAssignmentRequest,
) (*worker.WorkerPTOPolicyAssignment, error) {
	log := s.l.With(
		zap.String("operation", "EndAssignment"),
		zap.String("id", req.AssignmentID.String()),
	)

	assignment, err := s.repo.GetAssignmentByID(ctx, &repositories.GetPTOAssignmentByIDRequest{
		ID:         req.AssignmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !assignment.IsOpen() {
		return nil, errortypes.NewValidationError(
			"assignmentId",
			errortypes.ErrInvalidOperation,
			"This assignment has already ended",
		)
	}
	if req.Version > 0 && assignment.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Assignment was changed by someone else. Reload and try again",
		)
	}
	effectiveTo := req.EffectiveTo
	if effectiveTo <= 0 {
		effectiveTo = time.Now().Unix()
	}
	if effectiveTo <= assignment.EffectiveFrom {
		effectiveTo = assignment.EffectiveFrom + dayInSeconds
	}

	original := *assignment
	assignment.EffectiveTo = &effectiveTo
	updated, err := s.repo.UpdateAssignment(ctx, assignment)
	if err != nil {
		log.Error("failed to end PTO assignment", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePTOPolicy,
		ResourceID:     updated.PTOPolicyID.String(),
		Operation:      permission.OpAssign,
		UserID:         req.UserID,
		PreviousState:  jsonutils.MustToJSON(&original),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("PTO policy assignment ended"),
		auditservice.WithDiff(&original, updated)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updated, nil
}

func (s *Service) AssignDefaultPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	effectiveFrom int64,
	userID pulid.ID,
) error {
	policy, err := s.repo.GetDefault(ctx, tenantInfo)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if effectiveFrom <= 0 {
		effectiveFrom = time.Now().Unix()
	}

	result, err := s.AssignWorkers(ctx, &AssignWorkersRequest{
		TenantInfo:    tenantInfo,
		PolicyID:      policy.ID,
		WorkerIDs:     []pulid.ID{workerID},
		EffectiveFrom: effectiveFrom,
		Note:          "Default policy applied at hire",
		UserID:        userID,
		SystemActor:   userID.IsNil(),
	})
	if err != nil {
		return err
	}
	if len(result.Failures) > 0 {
		return fmt.Errorf("assign default PTO policy: %s", result.Failures[0].Error)
	}
	return nil
}
