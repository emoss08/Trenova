package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
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
	) ([]*shipmenthold.ShipmentHold, error)
}

const MaxNewestCommentsPerShipment = 20

type NewestCommentsRequest struct {
	ShipmentIDs []pulid.ID
	PerShipment int
}

type ShipmentCommentDigest struct {
	ShipmentID pulid.ID
	Total      int
	Newest     []*shipment.ShipmentComment
}

type NewestCommentReader interface {
	NewestComments(
		ctx context.Context,
		req *NewestCommentsRequest,
	) (map[pulid.ID]*ShipmentCommentDigest, error)
}
