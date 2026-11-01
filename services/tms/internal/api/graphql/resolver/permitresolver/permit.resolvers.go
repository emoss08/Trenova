package permitresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
)

func (r *MutationResolver) CreateShipmentPermit(ctx context.Context, shipmentID string, input gqlmodel.PermitInput) (*permit.Permit, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourcePermit, permission.OpCreate)
	if err != nil {
		return nil, err
	}

	entity, err := permitFromInput(&input, shipmentID, authCtx)
	if err != nil {
		return nil, err
	}

	return r.PermitService.CreatePermit(ctx, entity, actorutil.FromAuthContext(authCtx))
}

func (r *MutationResolver) UpdateShipmentPermit(ctx context.Context, shipmentID string, id string, input gqlmodel.PermitInput) (*permit.Permit, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourcePermit, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	permitID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	entity, err := permitFromInput(&input, shipmentID, authCtx)
	if err != nil {
		return nil, err
	}
	entity.ID = permitID

	return r.PermitService.UpdatePermit(ctx, entity, actorutil.FromAuthContext(authCtx))
}

func (r *MutationResolver) WaiveShipmentPermitRequirement(ctx context.Context, requirementID string, reason string) (*permit.Requirement, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourcePermit, permission.OpApprove)
	if err != nil {
		return nil, err
	}

	parsedRequirementID, err := gqlid.ParseIDField("requirementId", requirementID)
	if err != nil {
		return nil, err
	}

	return r.PermitService.WaiveRequirement(ctx, &permitports.WaiveRequirementRequest{
		TenantInfo:    resolvercore.TenantInfo(authCtx),
		RequirementID: parsedRequirementID,
		WaivedByID:    authCtx.UserID,
		Reason:        reason,
		Actor:         actorutil.FromAuthContext(authCtx),
	})
}

func (r *PermitResolver) Cost(ctx context.Context, obj *permit.Permit) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.Cost), nil
}

func (r *PermitAssessmentResolver) TotalEstimatedFee(ctx context.Context, obj *permitports.PermitAssessment) (string, error) {
	return gqldecimal.DecimalString(obj.TotalEstimatedFee), nil
}

func (r *PermitRequirementResolver) EstimatedFee(ctx context.Context, obj *permit.Requirement) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.EstimatedFee), nil
}

func (r *QueryResolver) ShipmentPermitAssessment(ctx context.Context, shipmentID string) (*permitports.PermitAssessment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipment, permission.OpRead)
	if err != nil {
		return nil, err
	}

	parsedShipmentID, err := gqlid.ParseIDField("shipmentId", shipmentID)
	if err != nil {
		return nil, err
	}

	entity, err := r.ShipmentReader.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         parsedShipmentID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return r.PermitService.Assess(ctx, entity)
}

func (r *QueryResolver) ShipmentPermits(ctx context.Context, shipmentID string) ([]*permit.Permit, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourcePermit, permission.OpRead)
	if err != nil {
		return nil, err
	}

	parsedShipmentID, err := gqlid.ParseIDField("shipmentId", shipmentID)
	if err != nil {
		return nil, err
	}

	return r.PermitService.ListPermits(ctx, parsedShipmentID, resolvercore.TenantInfo(authCtx))
}

func (r *QueryResolver) ShipmentPermitRequirements(ctx context.Context, shipmentID string) ([]*permit.Requirement, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourcePermit, permission.OpRead)
	if err != nil {
		return nil, err
	}

	parsedShipmentID, err := gqlid.ParseIDField("shipmentId", shipmentID)
	if err != nil {
		return nil, err
	}

	return r.PermitService.ListRequirements(
		ctx,
		parsedShipmentID,
		resolvercore.TenantInfo(authCtx),
	)
}
