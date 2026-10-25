package shipmentboardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func NewCarrierPerformanceRepository(r *Repository) shipmentports.CarrierPerformanceRepository {
	return r
}

type offerTotalsRow struct {
	CarrierID pulid.ID
	Answered  int64
	Accepted  int64
}

type rateRow struct {
	CarrierID pulid.ID
	Loads     int64
	AvgRate   decimal.NullDecimal
}

func (r *Repository) ListCarrierPerformance(
	ctx context.Context,
	req *shipmentports.ListCarrierPerformanceRequest,
) ([]*shipmentports.CarrierPerformance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.CarrierPerformance, error) {
			var offers []offerTotalsRow
			var rates []rateRow
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.IntoSlice(offerTotalsQuery(req), &offers),
				keel.IntoSlice(perMileRatesQuery(req), &rates),
			); err != nil {
				return nil, fmt.Errorf("read carrier performance: %w", err)
			}

			return mergePerformance(offers, rates), nil
		},
	)
}

func offerTotalsQuery(req *shipmentports.ListCarrierPerformanceRequest) keel.Projection {
	to := tenderdb.TenderOffers
	q := to.Group(to.CarrierID).
		Pick(
			to.CarrierID,
			keel.Count().Filter(to.Status.In(
				tender.OfferStatusAccepted,
				tender.OfferStatusDeclined,
				tender.OfferStatusExpired,
			)),
			keel.Count().Filter(to.Status.Eq(tender.OfferStatusAccepted)),
		).
		Where(to.SentAt.Gte(req.Since))
	if len(req.CarrierIDs) > 0 {
		q = q.Where(to.CarrierID.In(req.CarrierIDs...))
	}

	return q
}

func perMileRatesQuery(req *shipmentports.ListCarrierPerformanceRequest) keel.Projection {
	ca := shipmentdb.CarrierAssignments
	q := ca.Group(ca.CarrierID).
		Pick(ca.CarrierID, keel.Count(), keel.Avg(ca.BaseRate)).
		Where(
			ca.RateMethod.Eq(shipment.CarrierRateMethodPerMile),
			ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
			ca.CreatedAt.Gte(req.Since),
		)
	if len(req.CarrierIDs) > 0 {
		q = q.Where(ca.CarrierID.In(req.CarrierIDs...))
	}

	return q
}

func mergePerformance(
	offers []offerTotalsRow,
	rates []rateRow,
) []*shipmentports.CarrierPerformance {
	byCarrier := make(map[pulid.ID]*shipmentports.CarrierPerformance, len(offers)+len(rates))
	entry := func(id pulid.ID) *shipmentports.CarrierPerformance {
		if existing, ok := byCarrier[id]; ok {
			return existing
		}
		created := &shipmentports.CarrierPerformance{CarrierID: id}
		byCarrier[id] = created

		return created
	}
	for _, row := range offers {
		perf := entry(row.CarrierID)
		perf.OffersAnswered = int(row.Answered)
		perf.OffersAccepted = int(row.Accepted)
	}
	for _, row := range rates {
		perf := entry(row.CarrierID)
		perf.PerMileLoads = int(row.Loads)
		perf.AvgRatePerMile = row.AvgRate
	}

	out := make([]*shipmentports.CarrierPerformance, 0, len(byCarrier))
	for _, perf := range byCarrier {
		out = append(out, perf)
	}

	return out
}
