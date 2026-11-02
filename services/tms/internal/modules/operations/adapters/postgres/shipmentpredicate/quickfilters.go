package shipmentpredicate

import (
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
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
) keel.Query[*shipment.Stop] {
	return shipmentStop(st, sm, pickFirstPickup)
}

func LastDeliveryStop(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
) keel.Query[*shipment.Stop] {
	return shipmentStop(st, sm, pickLastDelivery)
}

func shipmentStop(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
	pick stopPick,
) keel.Query[*shipment.Stop] {
	sp := shipmentdb.Shipments
	q := st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			st.Status.Ne(shipmentvocab.StopStatusCanceled),
		)

	if pick == pickFirstPickup {
		return q.Where(st.Type.In(shipmentvocab.PickupStopTypes()...)).
			Order(sm.Sequence.Asc(), st.Sequence.Asc()).
			Limit(1)
	}

	return q.Where(st.Type.In(shipmentvocab.DeliveryStopTypes()...)).
		Order(sm.Sequence.Desc(), st.Sequence.Desc()).
		Limit(1)
}

func BillingTransferCandidateCondition() keel.Pred {
	sp := shipmentdb.Shipments

	return keel.And(
		sp.Status.In(shipmentvocab.BillingTransferCandidateStatuses()...),
		sp.BillingTransferStatus.In(
			shipmentvocab.BillingTransferNone,
			shipmentvocab.BillingTransferSentBackToOps,
		),
	)
}

func StageCondition(stage shipmentvocab.Stage) keel.Pred {
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
	spec quickfilter.QuickFilterSpec,
) (keel.Pred, error) {
	if basis == nil || basis.Location == nil {
		return nil, ErrQuickFilterBasisMissing
	}

	switch spec.Filter {
	case quickfilter.QuickFilterLate:
		return StageCondition(shipmentvocab.StageLate), nil
	case quickfilter.QuickFilterUncovered:
		return StageCondition(shipmentvocab.StageNeedsCoverage), nil
	case quickfilter.QuickFilterMoving:
		return StageCondition(shipmentvocab.StageMoving), nil
	case quickfilter.QuickFilterDeliveringToday:
		day := LocalDayOf(basis.Now, basis.Location)
		return deliveryWithin(day.Start, day.End), nil
	case quickfilter.QuickFilterDeliveryHour:
		if spec.Hour == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		start := LocalHourStart(basis.Now, basis.Location, *spec.Hour)
		return deliveryWithin(start, start+secondsPerHour), nil
	case quickfilter.QuickFilterPickupWindow:
		if spec.WindowStartMinutes == nil {
			return nil, ErrQuickFilterParameterMissing
		}
		return pickupWindowCondition(basis.Now.Unix(), spec), nil
	case quickfilter.QuickFilterReefer:
		return reeferCondition(), nil
	case quickfilter.QuickFilterLowMargin:
		if basis.Margin == nil {
			return nil, ErrQuickFilterMarginMissing
		}
		return lowMarginCondition(basis.Margin), nil
	case quickfilter.QuickFilterDetention:
		if basis.Detention == nil {
			return nil, ErrQuickFilterDetentionMissing
		}
		return DetentionCondition(basis.Now.Unix(), basis.Detention), nil
	case quickfilter.QuickFilterReadyToBill:
		return BillingTransferCandidateCondition(), nil
	default:
		return nil, ErrQuickFilterUnknown
	}
}

// deliveryWithin matches shipments that deliver in [start, end). It reads the
// delivery time shipmentprojection stores on the shipment, so a count over
// every shipment in scope is a column comparison per row, not a search for
// each shipment's last delivery stop.
func deliveryWithin(start, end int64) keel.Pred {
	sp := shipmentdb.Shipments

	return keel.And(
		sp.Status.Ne(shipmentvocab.StatusCanceled),
		sp.DeliveryAt.Gte(start),
		sp.DeliveryAt.Lt(end),
	)
}

// pickupWindowCondition matches uncovered shipments whose first pickup is
// scheduled in [now+start, now+end): an unset or zero start leaves the window
// open behind, an unset end open ahead. It reads the pickup time
// shipmentprojection stores on the shipment.
func pickupWindowCondition(now int64, spec quickfilter.QuickFilterSpec) keel.Pred {
	sp := shipmentdb.Shipments
	preds := []keel.Pred{
		StageCondition(shipmentvocab.StageNeedsCoverage),
		sp.PickupAt.IsNotNull(),
	}
	if start := int64(*spec.WindowStartMinutes); start > 0 {
		preds = append(preds, sp.PickupAt.Gte(now+start*secondsPerMinute))
	}
	if spec.WindowEndMinutes != nil {
		preds = append(preds, sp.PickupAt.Lt(now+int64(*spec.WindowEndMinutes)*secondsPerMinute))
	}

	return keel.And(preds...)
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
		Where(sm.ShipmentID.EqCol(sp.ID), sm.Status.Ne(shipmentvocab.MoveStatusCanceled)).
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

// DetentionCondition matches shipments in detention: those with an open,
// unbilled occurrence past its free time under the policy engine, otherwise
// those with a stop dwelling past the threshold. Both are a membership test
// against one uncorrelated set (the open occurrences, the dwelling stops), so a
// count over every shipment in scope builds the set once and probes it per
// row, and a filtered list reads it as a semi-join.
func DetentionCondition(now int64, basis *shipmentports.ShipmentDetentionBasis) keel.Pred {
	sp := shipmentdb.Shipments

	if basis.UsePolicyEngine {
		dto := detentiondb.DetentionOccurrences.As(quickDetention)
		return sp.ID.InQuery(dto.Pick(dto.ShipmentID).Where(
			dto.IsOpen.IsTrue(),
			dto.Status.Ne(detention.OccurrenceStatusBilled),
			dto.FreeTimeExpiresAt.Lte(now),
		))
	}

	sm := shipmentdb.ShipmentMoves.As(quickDwellMove)

	return sp.ID.InQuery(DwellingStops(
		shipmentdb.Stops.As(quickDwellAlias),
		sm,
		now,
		basis.ThresholdMinutes,
	).Pick(sm.ShipmentID))
}

// DwellingStops are the stops sitting at a facility past the detention
// threshold: arrived, not departed, not excluded from detention, on a live
// move. The query is not tied to a shipment; callers join or match
// sm.ShipmentID. The arrival and departure tests are written as
// idx_stops_arrived_open's predicate, with inline constants, so the planner
// reads the dwelling stops from that index.
func DwellingStops(
	st *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
	now, thresholdMinutes int64,
) keel.Query[*shipment.Stop] {
	return st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).Where(
		sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
		st.Status.Ne(shipmentvocab.StopStatusCanceled),
		keel.Int64Of(st.ActualArrival).GtExpr(keel.ConstInt64(0)),
		keel.Or(
			st.ActualDeparture.IsNull(),
			keel.Int64Of(st.ActualDeparture).EqExpr(keel.ConstInt64(0)),
		),
		st.CountDetentionOverride.Coalesce(true),
		st.ActualArrival.Lte(now-thresholdMinutes*secondsPerMinute),
	)
}

func AggregateScope(
	filter *pagination.QueryOptions,
	opts *shipmentports.ShipmentOptions,
) (keel.Query[*shipment.Shipment], error) {
	if filter == nil {
		return nil, ErrShipmentScopeFilterMissing
	}

	quick, err := QuickFilterConditions(opts)
	if err != nil {
		return nil, err
	}

	return Scope(filter, opts, quick), nil
}
