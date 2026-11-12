package dispatchconsoleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
)

func mapDispatchEndMarket(market *dispatchplanports.EndMarket) *gqlmodel.DispatchEndMarket {
	if market == nil {
		return nil
	}

	mapped := &gqlmodel.DispatchEndMarket{
		Source:         mapDispatchEndMarketSource(market.Source),
		ValueCents:     int(market.Cents),
		DropValueCents: int(market.DropCents),
		StayValueCents: int(market.StayCents),
		Customers:      []*gqlmodel.DispatchForecastCustomer{},
	}
	if market.StayCents > 0 && market.StayReason != nil {
		mapped.StayAreaLabel = gqlscalar.NonEmptyPtr(market.StayReason.AreaLabel)
	}

	reason := market.Reason
	if reason == nil {
		return mapped
	}
	mapped.AreaLabel = gqlscalar.NonEmptyPtr(reason.AreaLabel)
	mapped.WindowStart = new(int(reason.WindowStart))
	mapped.WindowEnd = new(int(reason.WindowEnd))
	mapped.TypicalLow = new(reason.TypicalLow)
	mapped.TypicalHigh = new(reason.TypicalHigh)
	mapped.Booked = new(reason.Booked)
	mapped.DriversExpected = new(reason.DriversExpected)
	mapped.Customers = mapDispatchForecastCustomers(reason.Customers)

	return mapped
}

func mapDispatchEndMarketSource(
	source dispatchplanports.EndMarketSource,
) gqlmodel.DispatchEndMarketSource {
	if source == dispatchplanports.EndMarketForecast {
		return gqlmodel.DispatchEndMarketSourceForecast
	}

	return gqlmodel.DispatchEndMarketSourceHistory
}

func mapDispatchForecastCustomers(
	customers []demandforecast.CustomerShare,
) []*gqlmodel.DispatchForecastCustomer {
	mapped := make([]*gqlmodel.DispatchForecastCustomer, 0, len(customers))
	for _, customer := range customers {
		mapped = append(mapped, &gqlmodel.DispatchForecastCustomer{
			CustomerID:    customer.CustomerID.String(),
			Name:          customer.Name,
			ExpectedLoads: customer.Expected,
		})
	}

	return mapped
}

func mapDispatchForecastQuality(
	quality *demandforecast.Quality,
	refitting bool,
) *gqlmodel.DispatchForecastQuality {
	if quality == nil {
		return &gqlmodel.DispatchForecastQuality{
			Status:    gqlmodel.DispatchForecastStatusInsufficientHistory,
			Refitting: refitting,
		}
	}

	return &gqlmodel.DispatchForecastQuality{
		Status:                 mapDispatchForecastStatus(quality.Status),
		HistoryWeeks:           int(quality.HistoryWeeks),
		HistoryLoads:           int(quality.HistoryLoads),
		WeeksTested:            int(quality.WeeksTested),
		LoadsTested:            int(quality.LoadsTested),
		IntervalCoverage:       quality.IntervalCoverage,
		MeanAbsoluteError:      quality.MAE,
		NaiveMeanAbsoluteError: quality.NaiveMAE,
		Skill:                  quality.Skill,
		FittedAt:               gqlscalar.NonZeroIntPtr(quality.FittedAt),
		Refitting:              refitting,
	}
}

func mapDispatchForecastStatus(status demandforecast.Status) gqlmodel.DispatchForecastStatus {
	switch status {
	case demandforecast.StatusReady:
		return gqlmodel.DispatchForecastStatusReady
	case demandforecast.StatusNoSkill:
		return gqlmodel.DispatchForecastStatusNoSkill
	case demandforecast.StatusInsufficientHistory:
		return gqlmodel.DispatchForecastStatusInsufficientHistory
	default:
		return gqlmodel.DispatchForecastStatusInsufficientHistory
	}
}

func mapDispatchFreightOutlook(
	outlook *freightoutlookports.Outlook,
) *gqlmodel.DispatchFreightOutlook {
	mapped := &gqlmodel.DispatchFreightOutlook{
		Quality:      mapDispatchForecastQuality(outlook.Quality, outlook.Refitting),
		GeneratedAt:  int(outlook.GeneratedAt),
		SlotStarts:   make([]int, 0, len(outlook.Starts)),
		Network:      make([]*gqlmodel.DispatchFreightNetworkSlot, 0, len(outlook.Network)),
		Areas:        make([]*gqlmodel.DispatchFreightArea, 0, len(outlook.Areas)),
		AreaCount:    outlook.AreaCount,
		Moves:        mapDispatchFreightMoves(outlook.Steps),
		Pace:         mapDispatchFreightPaces(outlook.Pace),
		Record:       make([]*gqlmodel.DispatchForecastDay, 0, len(outlook.Days)),
		Positionings: mapDispatchPositionings(outlook.Positionings),
	}
	for _, start := range outlook.Starts {
		mapped.SlotStarts = append(mapped.SlotStarts, int(start))
	}
	for _, slot := range outlook.Network {
		mapped.Network = append(mapped.Network, &gqlmodel.DispatchFreightNetworkSlot{
			Start:      int(slot.Start),
			LikelyLow:  slot.Low,
			LikelyHigh: slot.High,
			NeedLow:    slot.NeedLow,
			NeedHigh:   slot.NeedHigh,
			Need:       slot.Need,
			Drivers:    slot.Drivers,
			Short:      slot.Short,
			Uncovered:  slot.Uncovered,
			Idle:       slot.Idle,
		})
	}
	for _, area := range outlook.Areas {
		mapped.Areas = append(mapped.Areas, mapDispatchFreightArea(area))
	}
	for _, day := range outlook.Days {
		mapped.Record = append(mapped.Record, &gqlmodel.DispatchForecastDay{
			DayStart: int(day.DayStart),
			Low:      int(day.Low),
			High:     int(day.High),
			Mean:     day.Mean,
			Actual:   int(day.Actual),
		})
	}

	return mapped
}

func mapDispatchFreightAreaDetail(
	detail *freightoutlookports.AreaDetail,
) *gqlmodel.DispatchFreightAreaDetail {
	mapped := &gqlmodel.DispatchFreightAreaDetail{
		Quality:     mapDispatchForecastQuality(detail.Quality, detail.Refitting),
		GeneratedAt: int(detail.GeneratedAt),
		Area:        mapDispatchFreightArea(detail.Area),
		Bounds: &gqlmodel.DispatchFreightAreaBounds{
			South: detail.Bounds.South,
			West:  detail.Bounds.West,
			North: detail.Bounds.North,
			East:  detail.Bounds.East,
		},
		SlotStart:           int(detail.SlotStart),
		Reach:               make([]*gqlmodel.DispatchFreightReach, 0, len(detail.Reach)),
		Moves:               mapDispatchFreightMoves(detail.Steps),
		Pace:                mapDispatchFreightPaces(detail.Pace),
		Recent:              make([]int, 0, len(detail.Recent)),
		RecentLow:           detail.RecentLow,
		RecentHigh:          detail.RecentHigh,
		Positionings:        mapDispatchPositionings(detail.Positionings),
		DeliveryLocationIds: make([]string, 0, len(detail.DeliveryLocationIDs)),
	}
	for _, id := range detail.DeliveryLocationIDs {
		mapped.DeliveryLocationIds = append(mapped.DeliveryLocationIds, id.String())
	}
	for _, loads := range detail.Recent {
		mapped.Recent = append(mapped.Recent, int(loads))
	}
	for _, row := range detail.Reach {
		mapped.Reach = append(mapped.Reach, &gqlmodel.DispatchFreightReach{
			WorkerID:   row.WorkerID.String(),
			WorkerName: row.WorkerName,
			Label:      row.Label,
			Latitude:   row.Point.Latitude,
			Longitude:  row.Point.Longitude,
			FreeAt:     int(row.FreeAt),
			ArriveAt:   int(row.ArriveAt),
			Miles:      row.Miles,
			NetCents:   int(row.NetCents),
			Reach:      gqlmodel.DispatchFreightReachKind(row.Reach),
		})
	}

	return mapped
}

func mapDispatchFreightArea(area *freightoutlookports.Area) *gqlmodel.DispatchFreightArea {
	mapped := &gqlmodel.DispatchFreightArea{
		Cell:       int(area.Balance.Cell),
		Label:      area.Balance.Label,
		Latitude:   area.Point.Latitude,
		Longitude:  area.Point.Longitude,
		ValueCents: int(area.ValueCents),
		Slots:      make([]*gqlmodel.DispatchFreightSlot, 0, len(area.Balance.Slots)),
		Customers:  make([]*gqlmodel.DispatchFreightAreaCustomer, 0, len(area.Customers)),
	}
	for _, slot := range area.Balance.Slots {
		mapped.Slots = append(mapped.Slots, &gqlmodel.DispatchFreightSlot{
			Start:      int(slot.Start),
			LikelyLow:  slot.Low,
			LikelyHigh: slot.High,
			Booked:     slot.Booked,
			Need:       slot.Need,
			Drivers:    slot.Drivers,
			Short:      slot.Short,
			Spare:      slot.Spare,
		})
	}
	for _, customer := range area.Customers {
		mapped.Customers = append(mapped.Customers, &gqlmodel.DispatchFreightAreaCustomer{
			CustomerID: customer.CustomerID.String(),
			Name:       customer.Name,
			Share:      customer.Share,
		})
	}

	return mapped
}

func mapDispatchFreightMoves(steps []freightmove.Step) []*gqlmodel.DispatchFreightMove {
	mapped := make([]*gqlmodel.DispatchFreightMove, 0, len(steps))
	for i := range steps {
		step := &steps[i]
		laneIDs := make([]string, 0, len(step.LaneMoveIDs))
		for _, id := range step.LaneMoveIDs {
			laneIDs = append(laneIDs, id.String())
		}
		mapped = append(mapped, &gqlmodel.DispatchFreightMove{
			Kind:           gqlmodel.DispatchFreightMoveKind(step.Kind),
			WorkerID:       gqlscalar.NonEmptyPtr(step.WorkerID.String()),
			WorkerName:     step.WorkerName,
			FromCell:       int(step.FromCell),
			FromLabel:      step.FromLabel,
			ToCell:         int(step.ToCell),
			ToLabel:        step.ToLabel,
			SlotStart:      int(step.SlotStart),
			Short:          step.Short,
			FreeAt:         gqlscalar.NonZeroIntPtr(step.FreeAt),
			ArriveAt:       gqlscalar.NonZeroIntPtr(step.ArriveAt),
			Miles:          step.Miles,
			EmptyCostCents: int(step.EmptyCostCents),
			DropCents:      int(step.DropCents),
			StayCents:      int(step.StayCents),
			NetCents:       int(step.NetCents),
			SavedCents:     int(step.SavedCents),
			MoveID:         gqlscalar.NonEmptyPtr(step.MoveID.String()),
			ProNumber:      step.ProNumber,
			LaneMoveIds:    laneIDs,
			PartnerID:      gqlscalar.NonEmptyPtr(step.PartnerID.String()),
			PartnerName:    step.PartnerName,
			PartnerCell:    int(step.PartnerCell),
			PartnerLabel:   step.PartnerLabel,
			PartnerFreeAt:  gqlscalar.NonZeroIntPtr(step.PartnerFreeAt),
			PartnerMiles:   step.PartnerMiles,
			CustomerID:     gqlscalar.NonEmptyPtr(step.CustomerID.String()),
			CustomerName:   step.CustomerName,
			Usual:          step.Usual,
			ByNow:          step.ByNow,
			Tendered:       step.Tendered,
			Loads:          step.Loads,
		})
	}

	return mapped
}

func mapDispatchFreightPaces(paces []freightmove.Pace) []*gqlmodel.DispatchFreightPace {
	mapped := make([]*gqlmodel.DispatchFreightPace, 0, len(paces))
	for _, pace := range paces {
		slots := make([]*gqlmodel.DispatchFreightPaceSlot, 0, len(pace.Slots))
		for _, slot := range pace.Slots {
			slots = append(slots, &gqlmodel.DispatchFreightPaceSlot{
				Start:    int(slot.Start),
				Usual:    slot.Usual,
				ByNow:    slot.ByNow,
				Tendered: slot.Tendered,
			})
		}
		mapped = append(mapped, &gqlmodel.DispatchFreightPace{
			CustomerID:   pace.CustomerID.String(),
			CustomerName: pace.CustomerName,
			Cell:         int(pace.Cell),
			Label:        pace.Label,
			Slots:        slots,
		})
	}

	return mapped
}

func mapDispatchPositionings(
	rows []*freightoutlookports.Positioning,
) []*gqlmodel.DispatchPositioning {
	mapped := make([]*gqlmodel.DispatchPositioning, 0, len(rows))
	for _, row := range rows {
		mapped = append(mapped, mapDispatchPositioning(row.Positioning, row.WorkerName))
	}

	return mapped
}

func mapDispatchPositioning(
	positioning *driverpositioning.Positioning,
	workerName string,
) *gqlmodel.DispatchPositioning {
	return &gqlmodel.DispatchPositioning{
		ID:         positioning.ID.String(),
		WorkerID:   positioning.WorkerID.String(),
		WorkerName: workerName,
		FromLabel:  positioning.FromLabel,
		ToCell:     int(positioning.ToCell),
		ToLabel:    positioning.ToLabel,
		SlotStart:  int(positioning.SlotStart),
		FreeAt:     int(positioning.FreeAt),
		ArriveAt:   int(positioning.ArriveAt),
		Miles:      positioning.Miles,
		ValueCents: int(positioning.ValueCents),
		Status:     gqlmodel.DispatchPositioningStatus(positioning.Status),
		CreatedAt:  int(positioning.CreatedAt),
	}
}
