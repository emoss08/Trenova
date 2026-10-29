package shipmentresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/operationsloaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *MutationResolver) toggleShipmentComment(
	ctx context.Context,
	shipmentID string,
	commentID string,
	op permission.Operation,
	action func(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipmentdomain.ShipmentComment, error),
) (*gqlmodel.ShipmentComment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentComment, op)
	if err != nil {
		return nil, err
	}

	parsedShipmentID, err := pulid.MustParse(shipmentID)
	if err != nil {
		return nil, err
	}
	parsedCommentID, err := pulid.MustParse(commentID)
	if err != nil {
		return nil, err
	}

	entity, err := action(ctx, &shipmentcommentports.ToggleShipmentCommentRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		ShipmentID: parsedShipmentID,
		CommentID:  parsedCommentID,
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		return nil, err
	}

	return shipmentmapping.ShipmentCommentToModel(entity)
}

func (r *ShipmentResolver) loadShipmentOrder(
	ctx context.Context,
	obj *gqlmodel.Shipment,
) (*order.Order, error) {
	if obj == nil || obj.OrderID == nil || *obj.OrderID == "" {
		return nil, nil //nolint:nilnil // a shipment without an order resolves to null
	}

	if !loadercore.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Order loader is not configured")
	}

	parent, err := operationsloaders.OrderByID(ctx, *obj.OrderID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a deleted order resolves to null
		}
		return nil, err
	}
	return parent, nil
}
