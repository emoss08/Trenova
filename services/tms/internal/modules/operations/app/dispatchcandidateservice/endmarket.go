package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
)

func endMarketOf(
	snapshot *FleetSnapshot,
	workerID pulid.ID,
	move *dispatchboardports.BoardMove,
	freeAt int64,
) *dispatchplanports.EndMarket {
	if move.DestinationLatitude == nil || move.DestinationLongitude == nil {
		return nil
	}
	point := dispatchmarket.Point{
		Latitude:  *move.DestinationLatitude,
		Longitude: *move.DestinationLongitude,
	}

	if drop, ok := snapshot.Forecast.Value(demandforecast.Query{
		Point:    point,
		FreeAt:   freeAt,
		WorkerID: workerID,
	}); ok {
		market := &dispatchplanports.EndMarket{
			Source:    dispatchplanports.EndMarketForecast,
			DropCents: drop.Cents,
			Window:    drop.Key,
			Reason:    drop.Reason,
		}
		if start, availableAt, known := snapshot.DriverStart(workerID); known {
			if stay, stayed := snapshot.Forecast.Value(demandforecast.Query{
				Point:    start,
				FreeAt:   availableAt,
				WorkerID: workerID,
			}); stayed {
				market.StayCents = stay.Cents
				market.StayWindow = stay.Key
				market.StayReason = stay.Reason
			}
		}
		market.Cents = market.DropCents - market.StayCents
		if market.Cents == 0 {
			return nil
		}

		return market
	}

	cents := snapshot.EndMarket.EndMarketCents(point.Latitude, point.Longitude, freeAt)
	if cents <= 0 {
		return nil
	}

	return &dispatchplanports.EndMarket{
		Source:    dispatchplanports.EndMarketHistory,
		Cents:     cents,
		DropCents: cents,
	}
}

func Reprice(
	market *dispatchplanports.EndMarket,
	forecaster *demandforecast.Forecaster,
	workerID pulid.ID,
) {
	if market == nil || market.Source != dispatchplanports.EndMarketForecast {
		return
	}
	drop := forecaster.ValueAt(market.Window, workerID)
	market.DropCents, market.Reason = drop.Cents, drop.Reason
	if market.StayReason != nil {
		stay := forecaster.ValueAt(market.StayWindow, workerID)
		market.StayCents, market.StayReason = stay.Cents, stay.Reason
	}
	market.Cents = market.DropCents - market.StayCents
}
