package shipmentlistreadrepository

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
)

const (
	defaultOrderReferenceMatches = 10
	maxOrderReferenceMatches     = 25
	maxOrderReferenceCandidates  = 500
	maxActiveHoldShipments       = 100
	maxActiveHoldsPerShipment    = 25
)

type ListReadParams struct {
	fx.In

	DB *postgres.Connection
}

type orderReferenceMatcher struct {
	db *postgres.Connection
}

func NewOrderReferenceMatcher(p ListReadParams) shipmentports.OrderReferenceMatcher {
	return &orderReferenceMatcher{db: p.DB}
}

type orderReferenceCandidate struct {
	OrderID     pulid.ID
	OrderNumber string
	PONumber    string
	CreatedAt   int64
}

func (r *orderReferenceMatcher) MatchOrderReferences(
	ctx context.Context,
	req *shipmentports.MatchOrderReferencesRequest,
) ([]shipmentports.OrderReferenceMatch, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, nil
	}
	limit := orderReferenceLimit(req.Limit)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]shipmentports.OrderReferenceMatch, error) {
			candidates, err := keel.Into[orderReferenceCandidate](
				ctx,
				r.db.Keel(ctx),
				orderReferenceCandidatesQuery(query),
			)
			if err != nil {
				return nil, err
			}
			if len(candidates) <= maxOrderReferenceCandidates {
				return newestOrderReferences(candidates, limit), nil
			}
			return keel.Into[shipmentports.OrderReferenceMatch](
				ctx,
				r.db.Keel(ctx),
				orderReferenceQuery(query, limit),
			)
		},
	)
}

func orderReferenceCandidatesQuery(query string) *keel.Query[*order.Order] {
	o := orderdb.Orders

	return o.Pick(o.ID, o.OrderNumber, o.PONumber.Coalesce(""), o.CreatedAt).
		Where(listquery.MatchVector(o.SearchVector, query)).
		Limit(maxOrderReferenceCandidates + 1)
}

func newestOrderReferences(
	candidates []orderReferenceCandidate,
	limit int,
) []shipmentports.OrderReferenceMatch {
	slices.SortFunc(candidates, func(a, b orderReferenceCandidate) int {
		if c := cmp.Compare(b.CreatedAt, a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.OrderID, a.OrderID)
	})

	newest := candidates[:min(limit, len(candidates))]
	matches := make([]shipmentports.OrderReferenceMatch, 0, len(newest))
	for _, candidate := range newest {
		matches = append(matches, shipmentports.OrderReferenceMatch{
			OrderID:     candidate.OrderID,
			OrderNumber: candidate.OrderNumber,
			PONumber:    candidate.PONumber,
		})
	}
	return matches
}

func orderReferenceQuery(query string, limit int) *keel.Query[*order.Order] {
	o := orderdb.Orders

	return o.Pick(o.ID, o.OrderNumber, o.PONumber.Coalesce("")).
		Where(listquery.MatchVector(o.SearchVector, query)).
		Order(o.CreatedAt.Desc(), o.ID.Desc()).
		Limit(limit).
		CustomPlan()
}

func orderReferenceLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultOrderReferenceMatches
	case limit > maxOrderReferenceMatches:
		return maxOrderReferenceMatches
	default:
		return limit
	}
}

type activeHoldReader struct {
	db *postgres.Connection
}

func NewActiveHoldReader(p ListReadParams) shipmentports.ActiveHoldReader {
	return &activeHoldReader{db: p.DB}
}

func (r *activeHoldReader) ActiveHolds(
	ctx context.Context,
	req *shipmentports.ActiveHoldsRequest,
) ([]*shipmenthold.ShipmentHold, error) {
	ids := holdShipmentIDs(req.ShipmentIDs)
	if len(ids) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmenthold.ShipmentHold, error) {
		return activeHoldsQuery(ids).All(ctx, r.db.Keel(ctx))
	})
}

func activeHoldsQuery(ids []pulid.ID) *keel.Query[*shipmenthold.ShipmentHold] {
	sh := shipmentholddb.ShipmentHolds

	return sh.Where(sh.ShipmentID.In(ids...), sh.ReleasedAt.IsNull()).
		Load(sh.HoldReason()).
		Order(sh.ShipmentID.Asc(), sh.StartedAt.Desc(), sh.ID.Asc()).
		Limit(len(ids) * maxActiveHoldsPerShipment)
}

func holdShipmentIDs(ids []pulid.ID) []pulid.ID {
	named := sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(ids), func(id pulid.ID) bool {
		return id.IsNil()
	}))

	return named[:min(len(named), maxActiveHoldShipments)]
}

type newestCommentReader struct {
	db *postgres.Connection
}

func NewNewestCommentReader(p ListReadParams) shipmentports.NewestCommentReader {
	return &newestCommentReader{db: p.DB}
}

func (r *newestCommentReader) NewestComments(
	ctx context.Context,
	req *shipmentports.NewestCommentsRequest,
) ([]*shipment.ShipmentComment, error) {
	ids := holdShipmentIDs(req.ShipmentIDs)
	if len(ids) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.ShipmentComment, error) {
		return newestCommentsQuery(ids).All(ctx, r.db.Keel(ctx))
	})
}

func newestCommentsQuery(ids []pulid.ID) *keel.Query[*shipment.ShipmentComment] {
	sc := shipmentdb.ShipmentComments

	return sc.Where(sc.ShipmentID.In(ids...), sc.ParentCommentID.IsNull(), sc.DeletedAt.IsNull()).
		DistinctOn(sc.ShipmentID).
		Load(sc.User()).
		Order(sc.ShipmentID.Asc(), sc.CreatedAt.Desc(), sc.ID.Desc()).
		Limit(len(ids))
}
