package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
)

type OrderReferenceMatch struct {
	OrderID     pulid.ID
	OrderNumber string
	PONumber    string
}

type MatchOrderReferencesRequest struct {
	Query string
	Limit int
}

type OrderReferenceMatcher interface {
	MatchOrderReferences(
		ctx context.Context,
		req *MatchOrderReferencesRequest,
	) ([]OrderReferenceMatch, error)
}

type ActiveHoldsRequest struct {
	ShipmentIDs []pulid.ID
}

type ActiveHoldReader interface {
	ActiveHolds(
		ctx context.Context,
		req *ActiveHoldsRequest,
	) ([]*shipment.ShipmentHold, error)
}

type NewestCommentsRequest struct {
	ShipmentIDs []pulid.ID
}

type NewestCommentReader interface {
	NewestComments(
		ctx context.Context,
		req *NewestCommentsRequest,
	) ([]*shipment.ShipmentComment, error)
}
