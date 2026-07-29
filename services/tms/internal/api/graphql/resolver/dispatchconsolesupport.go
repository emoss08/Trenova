package resolver

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxBulkAssignBatch = 200

const genericDispatchErrorMessage = "Assignment failed due to an internal error"

func (r *mutationResolver) assignOneMove(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *gqlmodel.DispatchAssignMoveInput,
) *gqlmodel.DispatchAssignResult {
	result := &gqlmodel.DispatchAssignResult{
		MoveID:   item.MoveID,
		Findings: []*gqlmodel.DispatchFinding{},
	}

	moveID, err := parseDispatchID(item.MoveID, "moveId", "Move ID is not a valid identifier")
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	workerID, err := parseDispatchID(
		item.PrimaryWorkerID,
		"primaryWorkerId",
		"Worker ID is not a valid identifier",
	)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	tractorID, err := parseDispatchID(
		item.TractorID,
		"tractorId",
		"Tractor ID is not a valid identifier",
	)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}

	trailerID, err := optionalDispatchID(
		item.TrailerID,
		"trailerId",
		"Trailer ID is not a valid identifier",
	)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	secondaryWorkerID, err := optionalDispatchID(
		item.SecondaryWorkerID,
		"secondaryWorkerId",
		"Secondary worker ID is not a valid identifier",
	)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}

	var assignment *assignmentResult
	args := &assignmentArgs{
		TenantInfo:        tenant,
		MoveID:            moveID,
		PrimaryWorkerID:   workerID,
		TractorID:         tractorID,
		TrailerID:         trailerID,
		SecondaryWorkerID: secondaryWorkerID,
	}
	if item.Reassign != nil && *item.Reassign {
		assignment, err = r.reassignThroughService(ctx, args)
	} else {
		assignment, err = r.assignThroughService(ctx, args)
	}
	if err != nil {
		r.logDispatchError("assign_move", item.MoveID, err)
		result.Error = errorMessage(err)
		result.Findings = findingsFromError(err)
		return result
	}

	result.Success = true
	result.AssignmentID = &assignment.ID
	return result
}

type assignmentArgs struct {
	TenantInfo        pagination.TenantInfo
	MoveID            pulid.ID
	PrimaryWorkerID   pulid.ID
	TractorID         pulid.ID
	TrailerID         *pulid.ID
	SecondaryWorkerID *pulid.ID
}

type assignmentResult struct {
	ID string
}

func (r *mutationResolver) assignThroughService(
	ctx context.Context,
	args *assignmentArgs,
) (*assignmentResult, error) {
	entity, err := r.assignmentService.AssignToMove(
		ctx,
		&repositories.AssignShipmentMoveRequest{
			TenantInfo:        args.TenantInfo,
			ShipmentMoveID:    args.MoveID,
			PrimaryWorkerID:   args.PrimaryWorkerID,
			TractorID:         args.TractorID,
			TrailerID:         args.TrailerID,
			SecondaryWorkerID: args.SecondaryWorkerID,
		},
	)
	if err != nil {
		return nil, err
	}
	return &assignmentResult{ID: entity.ID.String()}, nil
}

func (r *mutationResolver) reassignThroughService(
	ctx context.Context,
	args *assignmentArgs,
) (*assignmentResult, error) {
	entity, err := r.assignmentService.Reassign(
		ctx,
		&repositories.ReassignShipmentMoveRequest{
			TenantInfo:        args.TenantInfo,
			ShipmentMoveID:    args.MoveID,
			PrimaryWorkerID:   args.PrimaryWorkerID,
			TractorID:         args.TractorID,
			TrailerID:         args.TrailerID,
			SecondaryWorkerID: args.SecondaryWorkerID,
		},
	)
	if err != nil {
		return nil, err
	}
	return &assignmentResult{ID: entity.ID.String()}, nil
}

func (r *mutationResolver) logDispatchError(operation, moveID string, err error) {
	if r.l == nil || isClientSafeError(err) {
		return
	}
	r.l.Error("dispatch console operation failed",
		zap.String("operation", operation),
		zap.String("moveId", moveID),
		zap.Error(err))
}

func summarizeBulkResults(
	results []*gqlmodel.DispatchAssignResult,
) *gqlmodel.DispatchBulkAssignResult {
	succeeded, failed := 0, 0
	for _, result := range results {
		if result.Success {
			succeeded++
			continue
		}
		failed++
	}

	return &gqlmodel.DispatchBulkAssignResult{
		Succeeded: succeeded,
		Failed:    failed,
		Results:   results,
	}
}

func findingsFromError(err error) []*gqlmodel.DispatchFinding {
	multiErr, ok := errors.AsType[*errortypes.MultiError](err)
	if !ok {
		return []*gqlmodel.DispatchFinding{}
	}

	findings := make([]*gqlmodel.DispatchFinding, 0, len(multiErr.Errors))
	for i := range multiErr.Errors {
		item := multiErr.Errors[i]
		severity := "Warn"
		if item.Code == errortypes.ErrComplianceViolation {
			severity = "Block"
		}
		findings = append(findings, &gqlmodel.DispatchFinding{
			Code:     string(item.Code),
			Severity: severity,
			Field:    item.Field,
			Message:  item.Message,
		})
	}

	return findings
}

func errorMessage(err error) *string {
	if err == nil {
		return nil
	}
	message := genericDispatchErrorMessage
	if isClientSafeError(err) {
		message = err.Error()
	}
	return &message
}

func isClientSafeError(err error) bool {
	return errortypes.IsError(err) ||
		errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err) ||
		errortypes.IsAuthorizationError(err) ||
		errortypes.IsAuthenticationError(err)
}

func parseDispatchID(raw, field, message string) (pulid.ID, error) {
	id, err := pulid.Parse(raw)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
	}
	return id, nil
}

func optionalDispatchID(raw *string, field, message string) (*pulid.ID, error) {
	if raw == nil || *raw == "" {
		return nil, nil //nolint:nilnil // an absent optional ID is not an error
	}
	id, err := parseDispatchID(*raw, field, message)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func optionalPulid(raw *string) (*pulid.ID, error) {
	if raw == nil || *raw == "" {
		return nil, nil //nolint:nilnil // an absent optional ID is not an error
	}
	id, err := pulid.Parse(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
