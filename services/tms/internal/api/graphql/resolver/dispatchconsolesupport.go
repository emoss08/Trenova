package resolver

// Hand-written support for the dispatcher console resolvers. This lives outside
// dispatch_console.resolvers.go because gqlgen rewrites that file on every generate and
// shelves anything it does not recognize.

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxBulkAssignBatch bounds a single bulk assignment. A dispatcher acting by hand never
// approaches this; it exists so a malformed client cannot ask for unbounded work.
const maxBulkAssignBatch = 200

func (r *mutationResolver) assignOneMove(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *gqlmodel.DispatchAssignMoveInput,
) *gqlmodel.DispatchAssignResult {
	result := &gqlmodel.DispatchAssignResult{
		MoveID:   item.MoveID,
		Findings: []*gqlmodel.DispatchFinding{},
	}

	moveID, err := pulid.Parse(item.MoveID)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	workerID, err := pulid.Parse(item.PrimaryWorkerID)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	tractorID, err := pulid.Parse(item.TractorID)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}

	trailerID, err := optionalPulid(item.TrailerID)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}
	secondaryWorkerID, err := optionalPulid(item.SecondaryWorkerID)
	if err != nil {
		result.Error = errorMessage(err)
		return result
	}

	// Both paths go through the assignment service so events, audit, validation, and the
	// driver notification behave identically to an assignment made from a shipment.
	var assignment *assignmentResult
	if item.Reassign != nil && *item.Reassign {
		assignment, err = r.reassignThroughService(ctx, &assignmentArgs{
			TenantInfo:        tenant,
			MoveID:            moveID,
			PrimaryWorkerID:   workerID,
			TractorID:         tractorID,
			TrailerID:         trailerID,
			SecondaryWorkerID: secondaryWorkerID,
		})
	} else {
		assignment, err = r.assignThroughService(ctx, &assignmentArgs{
			TenantInfo:        tenant,
			MoveID:            moveID,
			PrimaryWorkerID:   workerID,
			TractorID:         tractorID,
			TrailerID:         trailerID,
			SecondaryWorkerID: secondaryWorkerID,
		})
	}
	if err != nil {
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
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
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
	message := err.Error()
	return &message
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
