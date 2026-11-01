package shipmentmoveresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/idempotency"
)

func (r *MutationResolver) AssignShipmentMove(ctx context.Context, moveID string, input gqlmodel.ShipmentMoveAssignmentInput) (*gqlmodel.ShipmentAssignment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpAssign)
	if err != nil {
		return nil, err
	}

	args, err := assignmentArgsFromInput(moveID, &input)
	if err != nil {
		return nil, err
	}

	entity, err := r.AssignmentService.AssignToMove(ctx, &assignmentports.AssignShipmentMoveRequest{
		TenantInfo:        resolvercore.TenantInfo(authCtx),
		ShipmentMoveID:    args.MoveID,
		PrimaryWorkerID:   args.PrimaryWorkerID,
		TractorID:         args.TractorID,
		TrailerID:         args.TrailerID,
		SecondaryWorkerID: args.SecondaryWorkerID,
	})
	if err != nil {
		return nil, err
	}

	return shipmentmapping.ShipmentAssignmentToModel(entity), nil
}

func (r *MutationResolver) ReassignShipmentMove(ctx context.Context, moveID string, input gqlmodel.ShipmentMoveAssignmentInput) (*gqlmodel.ShipmentAssignment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpAssign)
	if err != nil {
		return nil, err
	}

	args, err := assignmentArgsFromInput(moveID, &input)
	if err != nil {
		return nil, err
	}

	entity, err := r.AssignmentService.Reassign(ctx, &assignmentports.ReassignShipmentMoveRequest{
		TenantInfo:        resolvercore.TenantInfo(authCtx),
		ShipmentMoveID:    args.MoveID,
		PrimaryWorkerID:   args.PrimaryWorkerID,
		TractorID:         args.TractorID,
		TrailerID:         args.TrailerID,
		SecondaryWorkerID: args.SecondaryWorkerID,
	})
	if err != nil {
		return nil, err
	}

	return shipmentmapping.ShipmentAssignmentToModel(entity), nil
}

func (r *MutationResolver) UnassignShipmentMove(ctx context.Context, moveID string) (bool, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpUnassign)
	if err != nil {
		return false, err
	}

	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return false, err
	}

	if err = r.AssignmentService.Unassign(ctx, &assignmentports.UnassignShipmentMoveRequest{
		TenantInfo:     resolvercore.TenantInfo(authCtx),
		ShipmentMoveID: parsedMoveID,
	}); err != nil {
		return false, err
	}

	return true, nil
}

func (r *MutationResolver) SplitShipmentMove(ctx context.Context, moveID string, input gqlmodel.SplitShipmentMoveInput) (*gqlmodel.SplitShipmentMoveResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	req, err := splitMoveRequestFromInput(moveID, &input)
	if err != nil {
		return nil, err
	}
	req.TenantInfo = resolvercore.TenantInfo(authCtx)

	result, err := r.ShipmentMoveService.SplitMove(ctx, req)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.SplitShipmentMoveResult{
		OriginalMove: shipmentmapping.ShipmentMoveToModel(result.OriginalMove),
		NewMove:      shipmentmapping.ShipmentMoveToModel(result.NewMove),
	}, nil
}

func (r *MutationResolver) RecordShipmentStopActual(ctx context.Context, moveID string, stopID string, input gqlmodel.RecordShipmentStopActualInput) (*gqlmodel.ShipmentMove, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return nil, err
	}
	parsedStopID, err := gqlid.ParseIDField("stopId", stopID)
	if err != nil {
		return nil, err
	}

	sourceKey, _ := idempotency.KeyFrom(ctx)
	result, err := r.TrackingEventService.RecordReported(
		ctx,
		&trackingeventports.RecordReportedStopEventParams{
			TenantInfo:   resolvercore.TenantInfo(authCtx),
			MoveID:       parsedMoveID,
			StopID:       parsedStopID,
			Action:       input.Action,
			OccurredAt:   gqlscalar.Int64Ptr(input.OccurredAt),
			Source:       trackingevent.SourceDispatcher,
			SourceKey:    sourceKey,
			ReportedByID: authCtx.UserID,
		},
	)
	if err != nil {
		return nil, err
	}

	return shipmentmapping.ShipmentMoveToModel(result.Move), nil
}

func (r *QueryResolver) ShipmentAssignment(ctx context.Context, id string) (*gqlmodel.ShipmentAssignment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpRead)
	if err != nil {
		return nil, err
	}

	assignmentID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	entity, err := r.AssignmentService.Get(ctx, &assignmentports.GetAssignmentByIDRequest{
		TenantInfo:   resolvercore.TenantInfo(authCtx),
		AssignmentID: assignmentID,
	})
	if err != nil {
		return nil, err
	}

	return shipmentmapping.ShipmentAssignmentToModel(entity), nil
}

func (r *QueryResolver) CheckWorkerCompliance(ctx context.Context, input gqlmodel.CheckWorkerComplianceInput) (bool, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentMove, permission.OpRead)
	if err != nil {
		return false, err
	}

	moveID, err := gqlid.ParseIDField("shipmentMoveId", input.ShipmentMoveID)
	if err != nil {
		return false, err
	}
	primaryWorkerID, err := gqlid.ParseIDField("primaryWorkerId", input.PrimaryWorkerID)
	if err != nil {
		return false, err
	}
	secondaryWorkerID, err := optionalIDPtr("secondaryWorkerId", input.SecondaryWorkerID)
	if err != nil {
		return false, err
	}

	if err = r.AssignmentService.CheckWorkerCompliance(
		ctx,
		&assignmentports.CheckWorkerComplianceRequest{
			TenantInfo:        resolvercore.TenantInfo(authCtx),
			ShipmentMoveID:    moveID,
			PrimaryWorkerID:   primaryWorkerID,
			SecondaryWorkerID: secondaryWorkerID,
		},
	); err != nil {
		return false, err
	}

	return true, nil
}
