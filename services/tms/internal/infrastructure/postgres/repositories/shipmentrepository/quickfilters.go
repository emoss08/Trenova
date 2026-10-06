package shipmentrepository

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

var (
	ErrQuickFilterBasisMissing     = errors.New("quick filters need a resolved basis")
	ErrQuickFilterMarginMissing    = errors.New("low margin quick filter needs a margin basis")
	ErrQuickFilterDetentionMissing = errors.New("detention quick filter needs a detention basis")
	ErrQuickFilterUnknown          = errors.New("unknown quick filter")
	ErrQuickFilterParameterMissing = errors.New("quick filter is missing a parameter")
	ErrShipmentScopeFilterMissing  = errors.New("shipment scope needs a tenant filter")
)

const (
	secondsPerMinute = int64(60)
	secondsPerHour   = int64(3600)
)

type stopPick int

const (
	pickFirstPickup stopPick = iota
	pickLastDelivery
)

func DeliveryAtExpr() string {
	stp := buncolgen.StopColumns
	return buncolgen.Expr("COALESCE({0}, {1})", stp.ActualArrival, stp.ScheduledWindowStart)
}

func FirstPickupStopQuery(dba bun.IDB) *bun.SelectQuery {
	return shipmentStopQuery(dba, pickFirstPickup)
}

func LastDeliveryStopQuery(dba bun.IDB) *bun.SelectQuery {
	return shipmentStopQuery(dba, pickLastDelivery)
}

func shipmentStopQuery(dba bun.IDB, pick stopPick) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns

	q := dba.NewSelect().
		Model((*shipment.Stop)(nil)).
		Join("JOIN "+buncolgen.ShipmentMoveTable.As(buncolgen.ShipmentMoveTable.Alias)).
		JoinOn(sm.ID.EqColumn(stp.ShipmentMoveID)).
		JoinOn(sm.OrganizationID.EqColumn(stp.OrganizationID)).
		JoinOn(sm.BusinessUnitID.EqColumn(stp.BusinessUnitID)).
		Where(sm.ShipmentID.EqColumn(sp.ID)).
		Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(sm.Status.Ne(), shipment.MoveStatusCanceled).
		Where(stp.Status.Ne(), shipment.StopStatusCanceled)

	if pick == pickFirstPickup {
		return q.
			Where(stp.Type.In(), bun.List(shipment.PickupStopTypes())).
			Order(sm.Sequence.OrderAsc(), stp.Sequence.OrderAsc()).
			Limit(1)
	}

	return q.
		Where(stp.Type.In(), bun.List(shipment.DeliveryStopTypes())).
		Order(sm.Sequence.OrderDesc(), stp.Sequence.OrderDesc()).
		Limit(1)
}

func BillingTransferCandidateCondition() schema.QueryAppender {
	sp := buncolgen.ShipmentColumns

	return bun.SafeQuery(
		"("+sp.Status.In()+" AND "+sp.BillingTransferStatus.Expr("COALESCE({}, '') IN (?)")+")",
		bun.List(shipment.BillingTransferCandidateStatuses()),
		bun.List([]shipment.BillingTransferStatus{
			shipment.BillingTransferNone,
			shipment.BillingTransferSentBackToOps,
		}),
	)
}

func StageCondition(stage shipment.Stage) schema.QueryAppender {
	return bun.SafeQuery(buncolgen.ShipmentColumns.StageRank.Eq(), stage.Rank())
}

type LocalDay struct {
	Start int64
	End   int64
}

func LocalDayOf(now time.Time, loc *time.Location) LocalDay {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return LocalDay{Start: start.Unix(), End: start.AddDate(0, 0, 1).Unix()}
}

func LocalHourStart(now time.Time, loc *time.Location, hour int) int64 {
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), hour, 0, 0, 0, loc).Unix()
}

func QuickFilterConditions(
	dba bun.IDB,
	opts repositories.ShipmentOptions,
) ([]schema.QueryAppender, error) {
	if !opts.HasQuickFilters() {
		return nil, nil
	}

	conds := make([]schema.QueryAppender, 0, len(opts.QuickFilters))
	for _, spec := range opts.QuickFilters {
		cond, err := QuickFilterCondition(dba, opts.QuickFilterBasis, spec)
		if err != nil {
			return nil, err
		}
		conds = append(conds, cond)
	}

	return conds, nil
}

func QuickFilterCondition(
	dba bun.IDB,
	basis *repositories.ShipmentQuickFilterBasis,
	spec shipment.QuickFilterSpec,
) (schema.QueryAppender, error) {
	if basis == nil || basis.Location == nil {
		return nil, ErrQuickFilterBasisMissing
	}

	switch spec.Filter {
	case shipment.QuickFilterLate:
		return StageCondition(shipment.StageLate), nil
	case shipment.QuickFilterUncovered:
		return StageCondition(shipment.StageNeedsCoverage), nil
	case shipment.QuickFilterMoving:
		return StageCondition(shipment.StageMoving), nil
	case shipment.QuickFilterDeliveringToday:
		day := LocalDayOf(basis.Now, basis.Location)
		return deliveryWithin(dba, day.Start, day.End), nil
	case shipment.QuickFilterDeliveryHour:
		if spec.Hour == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		start := LocalHourStart(basis.Now, basis.Location, *spec.Hour)
		return deliveryWithin(dba, start, start+secondsPerHour), nil
	case shipment.QuickFilterPickupWindow:
		if spec.WindowStartMinutes == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		return pickupWindowCondition(dba, basis.Now.Unix(), spec), nil
	case shipment.QuickFilterReefer:
		return reeferCondition(dba), nil
	case shipment.QuickFilterLowMargin:
		if basis.Margin == nil {
			return nil, ErrQuickFilterMarginMissing
		}
		return lowMarginCondition(dba, basis.Margin), nil
	case shipment.QuickFilterDetention:
		if basis.Detention == nil {
			return nil, ErrQuickFilterDetentionMissing
		}
		return DetentionCondition(dba, basis.Now.Unix(), basis.Detention), nil
	case shipment.QuickFilterReadyToBill:
		return BillingTransferCandidateCondition(), nil
	default:
		return nil, ErrQuickFilterUnknown
	}
}

func deliveryWithin(dba bun.IDB, start, end int64) schema.QueryAppender {
	sp := buncolgen.ShipmentColumns
	delivery := LastDeliveryStopQuery(dba).ColumnExpr(DeliveryAtExpr())

	return bun.SafeQuery(
		"("+sp.Status.Ne()+" AND int8range(?, ?) @> (?))",
		shipment.StatusCanceled,
		start,
		end,
		delivery,
	)
}

func pickupWindowCondition(
	dba bun.IDB,
	now int64,
	spec shipment.QuickFilterSpec,
) schema.QueryAppender {
	pickup := FirstPickupStopQuery(dba).
		ColumnExpr(buncolgen.StopColumns.ScheduledWindowStart.Qualified())

	var lower, upper *int64
	if start := int64(*spec.WindowStartMinutes); start > 0 {
		value := now + start*secondsPerMinute
		lower = &value
	}
	if spec.WindowEndMinutes != nil {
		value := now + int64(*spec.WindowEndMinutes)*secondsPerMinute
		upper = &value
	}

	return bun.SafeQuery(
		"(? AND int8range(?::bigint, ?::bigint) @> (?))",
		StageCondition(shipment.StageNeedsCoverage),
		lower,
		upper,
		pickup,
	)
}

func reeferCondition(dba bun.IDB) schema.QueryAppender {
	sp := buncolgen.ShipmentColumns
	mpf := buncolgen.ProfileColumns

	profiles := dba.NewSelect().
		Model((*modeprofile.Profile)(nil)).
		ColumnExpr("1").
		Where(mpf.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(mpf.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(mpf.Status.Eq(), modeprofile.ProfileStatusActive).
		Where(mpf.EquipmentClass.Eq(), modeprofile.EquipmentClassRefrigerated).
		Where(buncolgen.Expr(
			"{0} @> jsonb_build_array({1})",
			mpf.EquipmentTypeIDs,
			sp.TrailerTypeID,
		))

	return bun.SafeQuery(
		"("+sp.TemperatureMin.IsNotNull()+" OR "+sp.TemperatureMax.IsNotNull()+
			" OR ("+sp.TrailerTypeID.IsNotNull()+" AND EXISTS (?)))",
		profiles,
	)
}

func shipmentMilesQuery(dba bun.IDB) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns

	return dba.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		Where(sm.ShipmentID.EqColumn(sp.ID)).
		Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(sm.Status.Ne(), shipment.MoveStatusCanceled)
}

func costMilesExpr(includeDeadhead bool) string {
	sm := buncolgen.ShipmentMoveColumns
	if includeDeadhead {
		return sm.Distance.Expr("COALESCE(SUM({}), 0)")
	}
	return buncolgen.Expr("COALESCE(SUM({0}) FILTER (WHERE {1}), 0)", sm.Distance, sm.Loaded)
}

func lowMarginCondition(
	dba bun.IDB,
	margin *repositories.ShipmentMarginBasis,
) schema.QueryAppender {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	revenue := sp.TotalChargeAmount.Qualified()

	miles := shipmentMilesQuery(dba).
		ColumnExpr("1").
		Having(sm.Distance.Expr("COALESCE(SUM({}), 0) > 0")).
		Having(
			"("+revenue+" <= 0 OR ROUND(("+revenue+" - ROUND(?::numeric * ("+
				costMilesExpr(margin.IncludeDeadheadMiles)+")::numeric, 2)) * 100 / "+
				revenue+", 2) < ?)",
			margin.CostPerMile,
			margin.TargetMarginPercent,
		)

	return bun.SafeQuery("EXISTS (?)", miles)
}

func DetentionCondition(
	dba bun.IDB,
	now int64,
	basis *repositories.ShipmentDetentionBasis,
) schema.QueryAppender {
	sp := buncolgen.ShipmentColumns

	if basis.UsePolicyEngine {
		dto := buncolgen.DetentionOccurrenceColumns
		occurrences := dba.NewSelect().
			Model((*detention.DetentionOccurrence)(nil)).
			ColumnExpr("1").
			Where(dto.ShipmentID.EqColumn(sp.ID)).
			Where(dto.OrganizationID.EqColumn(sp.OrganizationID)).
			Where(dto.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
			Where(dto.IsOpen.IsTrue()).
			Where(dto.Status.Eq(), detention.OccurrenceStatusAccruing)

		return bun.SafeQuery("EXISTS (?)", occurrences)
	}

	return bun.SafeQuery("EXISTS (?)", DwellingStopsQuery(dba, now, basis.ThresholdMinutes).
		ColumnExpr("1"))
}

func DwellingStopsQuery(dba bun.IDB, now, thresholdMinutes int64) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns

	return dba.NewSelect().
		Model((*shipment.Stop)(nil)).
		Join("JOIN "+buncolgen.ShipmentMoveTable.As(buncolgen.ShipmentMoveTable.Alias)).
		JoinOn(sm.ID.EqColumn(stp.ShipmentMoveID)).
		JoinOn(sm.OrganizationID.EqColumn(stp.OrganizationID)).
		JoinOn(sm.BusinessUnitID.EqColumn(stp.BusinessUnitID)).
		Where(sm.ShipmentID.EqColumn(sp.ID)).
		Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(sm.Status.Ne(), shipment.MoveStatusCanceled).
		Where(stp.Status.Ne(), shipment.StopStatusCanceled).
		Where(stp.ActualArrival.Expr("COALESCE({}, 0) > 0")).
		Where(stp.ActualDeparture.Expr("COALESCE({}, 0) = 0")).
		Where(stp.CountDetentionOverride.Expr("COALESCE({}, TRUE)")).
		Where(stp.ActualArrival.Lte(), now-thresholdMinutes*secondsPerMinute)
}

func whereAll(conds []schema.QueryAppender) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		for _, cond := range conds {
			q = q.Where("?", cond)
		}
		return q
	}
}

func ApplyListScope(
	q *bun.SelectQuery,
	dba bun.IDB,
	filter *pagination.QueryOptions,
	opts repositories.ShipmentOptions,
) (*bun.SelectQuery, error) {
	if filter == nil {
		return nil, ErrShipmentScopeFilterMissing
	}

	quick, err := QuickFilterConditions(dba, opts)
	if err != nil {
		return nil, err
	}

	opts.ExpandShipmentDetails = false
	opts.IncludeCustomer = false

	return baseShipmentListQuery(q, dba, &repositories.ListShipmentsRequest{
		Filter:          filter,
		ShipmentOptions: opts,
	}).Apply(whereAll(quick)), nil
}
