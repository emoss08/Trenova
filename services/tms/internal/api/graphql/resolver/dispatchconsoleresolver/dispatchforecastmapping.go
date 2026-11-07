package dispatchconsoleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
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

func mapDispatchForecastQuality(quality *demandforecast.Quality) *gqlmodel.DispatchForecastQuality {
	if quality == nil {
		return &gqlmodel.DispatchForecastQuality{
			Status: gqlmodel.DispatchForecastStatusInsufficientHistory,
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
	outlook *demandforecastports.Outlook,
) *gqlmodel.DispatchFreightOutlook {
	if outlook == nil {
		return &gqlmodel.DispatchFreightOutlook{
			Quality: mapDispatchForecastQuality(nil),
			Windows: []*gqlmodel.DispatchFreightWindow{},
		}
	}

	windows := make([]*gqlmodel.DispatchFreightWindow, 0, len(outlook.Windows))
	for _, window := range outlook.Windows {
		if window == nil {
			continue
		}
		windows = append(windows, mapDispatchFreightWindow(window))
	}

	return &gqlmodel.DispatchFreightOutlook{
		Quality:     mapDispatchForecastQuality(outlook.Quality),
		GeneratedAt: int(outlook.GeneratedAt),
		Windows:     windows,
	}
}

func mapDispatchFreightWindow(
	window *demandforecastports.OutlookWindow,
) *gqlmodel.DispatchFreightWindow {
	return &gqlmodel.DispatchFreightWindow{
		AreaLabel:        window.AreaLabel,
		WindowStart:      int(window.WindowStart),
		WindowEnd:        int(window.WindowEnd),
		TypicalLow:       window.TypicalLow,
		TypicalHigh:      window.TypicalHigh,
		LikelyLow:        window.LikelyLow,
		LikelyHigh:       window.LikelyHigh,
		Booked:           window.Booked,
		ExpectedUnbooked: window.ExpectedUnbooked,
		DriversExpected:  window.DriversExpected,
		Shortfall:        window.Shortfall,
		ValueCents:       int(window.ValueCents),
		Customers:        mapDispatchForecastCustomers(window.Customers),
		Explanation:      window.Explanation,
	}
}
