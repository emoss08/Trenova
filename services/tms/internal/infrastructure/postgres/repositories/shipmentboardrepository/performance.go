package shipmentboardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

func NewCarrierPerformanceRepository(r *Repository) repositories.CarrierPerformanceRepository {
	return r
}

type offerTotalsRow struct {
	CarrierID pulid.ID `bun:"carrier_id"`
	Answered  int      `bun:"answered"`
	Accepted  int      `bun:"accepted"`
}

type rateRow struct {
	CarrierID pulid.ID            `bun:"carrier_id"`
	Loads     int                 `bun:"loads"`
	AvgRate   decimal.NullDecimal `bun:"avg_rate"`
}

func (r *Repository) ListCarrierPerformance(
	ctx context.Context,
	req *repositories.ListCarrierPerformanceRequest,
) ([]*repositories.CarrierPerformance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.CarrierPerformance, error) {
			dba := r.db.DBForContext(ctx)
			offers, err := r.offerTotals(ctx, dba, req)
			if err != nil {
				return nil, err
			}
			rates, err := r.perMileRates(ctx, dba, req)
			if err != nil {
				return nil, err
			}

			return mergePerformance(offers, rates), nil
		},
	)
}

func (r *Repository) offerTotals(
	ctx context.Context,
	dba bun.IDB,
	req *repositories.ListCarrierPerformanceRequest,
) ([]offerTotalsRow, error) {
	cols := buncolgen.TenderOfferColumns
	answered := []tender.OfferStatus{
		tender.OfferStatusAccepted,
		tender.OfferStatusDeclined,
		tender.OfferStatusExpired,
	}

	rows := make([]offerTotalsRow, 0)
	q := dba.NewSelect().
		Model((*tender.TenderOffer)(nil)).
		ColumnExpr(cols.CarrierID.As("carrier_id")).
		ColumnExpr("COUNT(*) FILTER (WHERE "+cols.Status.In()+") AS answered", bun.In(answered)).
		ColumnExpr(
			"COUNT(*) FILTER (WHERE "+cols.Status.Eq()+") AS accepted",
			tender.OfferStatusAccepted,
		).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.SentAt.Gte(), req.Since).
		GroupExpr(cols.CarrierID.Qualified())
	if len(req.CarrierIDs) > 0 {
		q = q.Where(cols.CarrierID.In(), bun.In(req.CarrierIDs))
	}
	if err := q.Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("count carrier tender answers: %w", err)
	}

	return rows, nil
}

func (r *Repository) perMileRates(
	ctx context.Context,
	dba bun.IDB,
	req *repositories.ListCarrierPerformanceRequest,
) ([]rateRow, error) {
	cols := buncolgen.CarrierAssignmentColumns

	rows := make([]rateRow, 0)
	q := dba.NewSelect().
		Model((*shipment.CarrierAssignment)(nil)).
		ColumnExpr(cols.CarrierID.As("carrier_id")).
		ColumnExpr("COUNT(*) AS loads").
		ColumnExpr("AVG("+cols.BaseRate.Qualified()+") AS avg_rate").
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.RateMethod.Eq(), shipment.CarrierRateMethodPerMile).
		Where(cols.Status.NotEq(), shipment.CarrierAssignmentStatusCanceled).
		Where(cols.CreatedAt.Gte(), req.Since).
		GroupExpr(cols.CarrierID.Qualified())
	if len(req.CarrierIDs) > 0 {
		q = q.Where(cols.CarrierID.In(), bun.In(req.CarrierIDs))
	}
	if err := q.Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("average carrier rates: %w", err)
	}

	return rows, nil
}

func mergePerformance(offers []offerTotalsRow, rates []rateRow) []*repositories.CarrierPerformance {
	byCarrier := make(map[pulid.ID]*repositories.CarrierPerformance, len(offers)+len(rates))
	entry := func(id pulid.ID) *repositories.CarrierPerformance {
		if existing, ok := byCarrier[id]; ok {
			return existing
		}
		created := &repositories.CarrierPerformance{CarrierID: id}
		byCarrier[id] = created

		return created
	}
	for _, row := range offers {
		perf := entry(row.CarrierID)
		perf.OffersAnswered = row.Answered
		perf.OffersAccepted = row.Accepted
	}
	for _, row := range rates {
		perf := entry(row.CarrierID)
		perf.PerMileLoads = row.Loads
		perf.AvgRatePerMile = row.AvgRate
	}

	out := make([]*repositories.CarrierPerformance, 0, len(byCarrier))
	for _, perf := range byCarrier {
		out = append(out, perf)
	}

	return out
}
