package shipmentrepository

import (
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile/modeprofiledb"
	"github.com/emoss08/trenova/pkg/pagination"
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

	quickStopAlias   = "qf_stp"
	quickMoveAlias   = "qf_sm"
	quickDwellAlias  = "qf_dw"
	quickDwellMove   = "qf_dwm"
	quickMilesAlias  = "qf_mi"
	quickDetention   = "qf_do"
	quickModeProfile = "qf_mpf"
)

type stopPick int

const (
	pickFirstPickup stopPick = iota
	pickLastDelivery
)

func DeliveryAt(st *shipmentdb.StopsTable) keel.Int64Expr {
	return st.ActualArrival.Coalesce(st.ScheduledWindowStart)
}

func FirstPickupStop(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
) *keel.Query[*shipment.Stop] {
	return shipmentStop(st, sm, pickFirstPickup)
}

func LastDeliveryStop(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
) *keel.Query[*shipment.Stop] {
	return shipmentStop(st, sm, pickLastDelivery)
}

func shipmentStop(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
	pick stopPick,
) *keel.Query[*shipment.Stop] {
	sp := shipmentdb.Shipments
	q := st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			sm.Status.Ne(shipment.MoveStatusCanceled),
			st.Status.Ne(shipment.StopStatusCanceled),
		)

	if pick == pickFirstPickup {
		return q.Where(st.Type.In(shipment.PickupStopTypes()...)).
			Order(sm.Sequence.Asc(), st.Sequence.Asc()).
			Limit(1)
	}

	return q.Where(st.Type.In(shipment.DeliveryStopTypes()...)).
		Order(sm.Sequence.Desc(), st.Sequence.Desc()).
		Limit(1)
}

func BillingTransferCandidateCondition() keel.Pred {
	sp := shipmentdb.Shipments

	return keel.And(
		sp.Status.In(shipment.BillingTransferCandidateStatuses()...),
		sp.BillingTransferStatus.In(
			shipment.BillingTransferNone,
			shipment.BillingTransferSentBackToOps,
		),
	)
}

func StageCondition(stage shipment.Stage) keel.Pred {
	return shipmentdb.Shipments.StageRank.Eq(stage.Rank())
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

func QuickFilterConditions(opts *shipmentports.ShipmentOptions) ([]keel.Pred, error) {
	if opts == nil || !opts.HasQuickFilters() {
		return nil, nil
	}

	conds := make([]keel.Pred, 0, len(opts.QuickFilters))
	for _, spec := range opts.QuickFilters {
		cond, err := QuickFilterCondition(opts.QuickFilterBasis, spec)
		if err != nil {
			return nil, err
		}
		conds = append(conds, cond)
	}

	return conds, nil
}

func QuickFilterCondition(
	basis *shipmentports.ShipmentQuickFilterBasis,
	spec shipment.QuickFilterSpec,
) (keel.Pred, error) {
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
		return deliveryWithin(day.Start, day.End), nil
	case shipment.QuickFilterDeliveryHour:
		if spec.Hour == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		start := LocalHourStart(basis.Now, basis.Location, *spec.Hour)
		return deliveryWithin(start, start+secondsPerHour), nil
	case shipment.QuickFilterPickupWindow:
		if spec.WindowStartMinutes == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		return pickupWindowCondition(basis.Now.Unix(), spec), nil
	case shipment.QuickFilterReefer:
		return reeferCondition(), nil
	case shipment.QuickFilterLowMargin:
		if basis.Margin == nil {
			return nil, ErrQuickFilterMarginMissing
		}
		return lowMarginCondition(basis.Margin), nil
	case shipment.QuickFilterDetention:
		if basis.Detention == nil {
			return nil, ErrQuickFilterDetentionMissing
		}
		return DetentionCondition(basis.Now.Unix(), basis.Detention), nil
	case shipment.QuickFilterReadyToBill:
		return BillingTransferCandidateCondition(), nil
	default:
		return nil, ErrQuickFilterUnknown
	}
}

func deliveryWithin(start, end int64) keel.Pred {
	sp := shipmentdb.Shipments
	st := shipmentdb.Stops.As(quickStopAlias)
	delivery := LastDeliveryStop(
		st,
		shipmentdb.ShipmentMoves.As(quickMoveAlias),
	).Pick(DeliveryAt(st))

	return keel.And(
		sp.Status.Ne(shipment.StatusCanceled),
		keel.Int8Range(keel.Int64(start), keel.Int64(end)).
			Covers(keel.Int64Of(keel.Scalar(delivery))),
	)
}

func pickupWindowCondition(now int64, spec shipment.QuickFilterSpec) keel.Pred {
	st := shipmentdb.Stops.As(quickStopAlias)
	pickup := FirstPickupStop(st, shipmentdb.ShipmentMoves.As(quickMoveAlias)).
		Pick(st.ScheduledWindowStart)

	var lower, upper *int64
	if start := int64(*spec.WindowStartMinutes); start > 0 {
		value := now + start*secondsPerMinute
		lower = &value
	}
	if spec.WindowEndMinutes != nil {
		value := now + int64(*spec.WindowEndMinutes)*secondsPerMinute
		upper = &value
	}

	return keel.And(
		StageCondition(shipment.StageNeedsCoverage),
		keel.Int8Range(keel.Int64Ptr(lower), keel.Int64Ptr(upper)).
			Covers(keel.Int64Of(keel.Scalar(pickup))),
	)
}

func reeferCondition() keel.Pred {
	sp := shipmentdb.Shipments
	mpf := modeprofiledb.Profiles.As(quickModeProfile)

	profiles := mpf.Where(
		mpf.Status.Eq(modeprofile.ProfileStatusActive),
		mpf.EquipmentClass.Eq(modeprofile.EquipmentClassRefrigerated),
		mpf.EquipmentTypeIDs.Contains(keel.JSONBuildArray(sp.TrailerTypeID)),
	)

	return keel.Or(
		sp.TemperatureMin.IsNotNull(),
		sp.TemperatureMax.IsNotNull(),
		keel.And(sp.TrailerTypeID.IsNotNull(), keel.Exists(profiles)),
	)
}

func costMiles(sm *shipmentdb.ShipmentMovesTable, includeDeadhead bool) keel.Float64Expr {
	if includeDeadhead {
		return keel.Float64Of(sm.Distance.Sum()).Coalesce(0)
	}

	return keel.Float64Of(sm.Distance.Sum().Filter(keel.BoolOf(sm.Loaded))).Coalesce(0)
}

func lowMarginCondition(margin *shipmentports.ShipmentMarginBasis) keel.Pred {
	sp := shipmentdb.Shipments
	sm := shipmentdb.ShipmentMoves.As(quickMilesAlias)
	revenue := sp.TotalChargeAmount

	cost := keel.Numeric(margin.CostPerMile).
		Mul(keel.CastNumeric(costMiles(sm, margin.IncludeDeadheadMiles))).
		RoundTo(2)

	miles := sm.
		Where(sm.ShipmentID.EqCol(sp.ID), sm.Status.Ne(shipment.MoveStatusCanceled)).
		Having(
			keel.Float64Of(sm.Distance.Sum()).Coalesce(0).Gt(0),
			keel.Or(
				revenue.AnyCol.Lte(0),
				revenue.Sub(cost).Mul(keel.Numeric(100)).Div(revenue).
					RoundTo(2).
					Lt(margin.TargetMarginPercent),
			),
		)

	return keel.Exists(miles)
}

func DetentionCondition(now int64, basis *shipmentports.ShipmentDetentionBasis) keel.Pred {
	sp := shipmentdb.Shipments

	if basis.UsePolicyEngine {
		dto := detentiondb.DetentionOccurrences.As(quickDetention)
		return keel.Exists(dto.Where(
			dto.ShipmentID.EqCol(sp.ID),
			dto.IsOpen.IsTrue(),
			dto.Status.Ne(detention.OccurrenceStatusBilled),
			dto.FreeTimeExpiresAt.Lte(now),
		))
	}

	return keel.Exists(DwellingStops(
		shipmentdb.Stops.As(quickDwellAlias),
		shipmentdb.ShipmentMoves.As(quickDwellMove),
		now,
		basis.ThresholdMinutes,
	))
}

func DwellingStops(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
	now, thresholdMinutes int64,
) *keel.Query[*shipment.Stop] {
	sp := shipmentdb.Shipments

	return st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).Where(
		sm.ShipmentID.EqCol(sp.ID),
		sm.Status.Ne(shipment.MoveStatusCanceled),
		st.Status.Ne(shipment.StopStatusCanceled),
		st.ActualArrival.Coalesce(0).Gt(0),
		st.ActualDeparture.Coalesce(0).Eq(0),
		st.CountDetentionOverride.Coalesce(true),
		st.ActualArrival.Lte(now-thresholdMinutes*secondsPerMinute),
	)
}

func AggregateScope(
	filter *pagination.QueryOptions,
	opts *shipmentports.ShipmentOptions,
) (*keel.Query[*shipment.Shipment], error) {
	if filter == nil {
		return nil, ErrShipmentScopeFilterMissing
	}

	quick, err := QuickFilterConditions(opts)
	if err != nil {
		return nil, err
	}

	return shipmentScope(filter, opts, quick), nil
}
