package dispatchquerytools

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramHoursAhead = "hoursAhead"

	forecastStatusReady        = "Ready"
	forecastStatusNoSkill      = "NoSkill"
	forecastStatusInsufficient = "InsufficientHistory"
	forecastStatusNotFitted    = "NotFitted"

	notFittedYet = "not fitted yet"
	unnamedArea  = "an area with no named origin"
)

type freightOutlookReader interface {
	Outlook(
		ctx context.Context,
		req *demandforecastports.OutlookRequest,
	) (*demandforecastports.Outlook, error)
}

type forecastDemandTool struct {
	outlook freightOutlookReader
}

func NewForecastDemandTool(outlook freightOutlookReader) agenttoolports.AgentQueryTool {
	return &forecastDemandTool{outlook: outlook}
}

func (t *forecastDemandTool) Name() string { return "dispatch_forecast_demand" }

func (t *forecastDemandTool) Description() string {
	return "Say which areas will be busy in the coming days and where to send trucks, " +
		"weighing freight customers have not booked yet against the drivers expected. Answers whether freight is coming into a city, how " +
		"many loads usually appear on a given day and which regions will need more trucks. " +
		"Learned from this organization's own shipment history, not market data: each " +
		"twelve-hour window gives the usual range of loads, what is booked, the loads still " +
		"expected, the drivers expected free there, the shortfall and what one more driver " +
		"ending there is worth in dollars. Plans use it only once quality.status is Ready; " +
		"until then it lists no windows, so say it is not proven yet and why, from " +
		"quality.meaning, rather than guessing. Answer by naming the areas, the windows and " +
		"the shortfall; act on it through dispatch_plan."
}

func (t *forecastDemandTool) SearchTerms() []string {
	return []string{
		"outlook", "anticipate", "demand", "upcoming", "coming", "busy", "usually", "regions",
		"short of drivers", "need more trucks", "position trucks",
	}
}

func (t *forecastDemandTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(map[string]any{
		paramHoursAhead: toolschema.Clamped(map[string]any{
			toolschema.KeyType:    toolschema.TypeInteger,
			toolschema.KeyMinimum: 1,
			toolschema.KeyMaximum: demandforecastports.MaxOutlookHours,
			toolschema.KeyDescription: fmt.Sprintf(
				"How many hours ahead to look, from now: %d unless you ask, at most %d. Ask "+
					"for enough to reach the day in question; a weekend can need all %d.",
				demandforecastports.DefaultOutlookHours,
				demandforecastports.MaxOutlookHours,
				demandforecastports.MaxOutlookHours,
			),
		}),
		querytool.ParamLimit: querytool.LimitSchema(
			demandforecastports.DefaultOutlookLimit,
			demandforecastports.MaxOutlookLimit,
		),
	})
}

func (t *forecastDemandTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipmentMove,
	})
}

type forecastQualityView struct {
	Status                  string `json:"status"`
	Meaning                 string `json:"meaning"`
	HistoryWeeks            int32  `json:"historyWeeks"`
	HistoryLoads            int64  `json:"historyLoads"`
	WeeksTested             int32  `json:"weeksTested"`
	IntervalCoveragePercent int    `json:"intervalCoveragePercent"`
	SkillPercent            int    `json:"skillPercent"`
	FittedAt                string `json:"fittedAt"`
}

type forecastCustomerView struct {
	CustomerID    string  `json:"customerId"`
	Name          string  `json:"name"`
	ExpectedLoads float64 `json:"expectedLoads"`
}

type freightWindowView struct {
	Area               string                 `json:"area"`
	WindowStart        string                 `json:"windowStart"`
	WindowEnd          string                 `json:"windowEnd"`
	TypicalLow         int                    `json:"typicalLow"`
	TypicalHigh        int                    `json:"typicalHigh"`
	LikelyLow          int                    `json:"likelyLow"`
	LikelyHigh         int                    `json:"likelyHigh"`
	Booked             int                    `json:"booked"`
	ExpectedUnbooked   float64                `json:"expectedUnbooked"`
	DriversExpected    int                    `json:"driversExpected"`
	Shortfall          float64                `json:"shortfall"`
	OneMoreDriverWorth string                 `json:"oneMoreDriverWorth"`
	Customers          []forecastCustomerView `json:"customers"`
	Explanation        string                 `json:"explanation"`
}

type freightOutlookView struct {
	Quality     forecastQualityView `json:"quality"`
	Hours       int                 `json:"hoursAhead"`
	GeneratedAt string              `json:"generatedAt"`
	Count       int                 `json:"count"`
	Windows     []freightWindowView `json:"windows"`
	Note        string              `json:"note"`
}

func (t *forecastDemandTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	hours := forecastHours(params.Params)
	outlook, err := t.outlook.Outlook(ctx, &demandforecastports.OutlookRequest{
		TenantInfo: querytool.TenantOf(params),
		Hours:      hours,
		Limit: querytool.ReadLimit(
			params.Params,
			demandforecastports.DefaultOutlookLimit,
			demandforecastports.MaxOutlookLimit,
		),
	})
	if err != nil {
		return nil, err
	}

	return freightOutlookViewOf(outlook, hours, timeutils.LoadLocation(params.Timezone)), nil
}

func forecastHours(params map[string]any) int {
	hours := querytool.OptionalInt(
		params,
		paramHoursAhead,
		demandforecastports.DefaultOutlookHours,
	)
	if hours <= 0 {
		return demandforecastports.DefaultOutlookHours
	}

	return min(hours, demandforecastports.MaxOutlookHours)
}

func freightOutlookViewOf(
	outlook *demandforecastports.Outlook,
	hours int,
	location *time.Location,
) *freightOutlookView {
	if outlook == nil {
		outlook = &demandforecastports.Outlook{}
	}
	view := &freightOutlookView{
		Quality: forecastQualityViewOf(outlook.Quality, location),
		Hours:   hours,
		Windows: make([]freightWindowView, 0, len(outlook.Windows)),
	}
	if outlook.GeneratedAt > 0 {
		view.GeneratedAt = timeutils.FormatStamp(outlook.GeneratedAt, location)
	}
	ready := view.Quality.Status == forecastStatusReady
	for _, window := range outlook.Windows {
		if window == nil || !ready {
			continue
		}
		view.Windows = append(view.Windows, freightWindowViewOf(window, location))
	}
	view.Count = len(view.Windows)
	view.Note = outlookNote(ready, view.Quality.Status, view.Count, hours)

	return view
}

func freightWindowViewOf(
	window *demandforecastports.OutlookWindow,
	location *time.Location,
) freightWindowView {
	out := freightWindowView{
		Area:               window.AreaLabel,
		WindowStart:        timeutils.FormatStamp(window.WindowStart, location),
		WindowEnd:          timeutils.FormatStamp(window.WindowEnd, location),
		TypicalLow:         window.TypicalLow,
		TypicalHigh:        window.TypicalHigh,
		LikelyLow:          window.LikelyLow,
		LikelyHigh:         window.LikelyHigh,
		Booked:             window.Booked,
		ExpectedUnbooked:   tenths(window.ExpectedUnbooked),
		DriversExpected:    window.DriversExpected,
		Shortfall:          tenths(window.Shortfall),
		OneMoreDriverWorth: dispatcheconomics.Dollars(window.ValueCents),
		Customers:          make([]forecastCustomerView, 0, len(window.Customers)),
		Explanation:        window.Explanation,
	}
	if out.Area == "" {
		out.Area = unnamedArea
	}
	for _, customer := range window.Customers {
		out.Customers = append(out.Customers, forecastCustomerView{
			CustomerID:    customer.CustomerID.String(),
			Name:          customer.Name,
			ExpectedLoads: tenths(customer.Expected),
		})
	}

	return out
}

func forecastQualityViewOf(
	quality *demandforecast.Quality,
	location *time.Location,
) forecastQualityView {
	if quality == nil || quality.FittedAt <= 0 {
		return forecastQualityView{
			Status: forecastStatusNotFitted,
			Meaning: "The forecast has not been fitted for this organization yet, so plans " +
				"do not use it. Say so rather than guessing where freight will appear.",
			FittedAt: notFittedYet,
		}
	}

	view := forecastQualityView{
		HistoryWeeks:            quality.HistoryWeeks,
		HistoryLoads:            quality.HistoryLoads,
		WeeksTested:             quality.WeeksTested,
		IntervalCoveragePercent: percent(quality.IntervalCoverage),
		SkillPercent:            percent(quality.Skill),
		FittedAt:                timeutils.FormatStamp(quality.FittedAt, location),
	}
	switch quality.Status {
	case demandforecast.StatusReady:
		view.Status = forecastStatusReady
		view.Meaning = fmt.Sprintf(
			"Tested on the last %d weeks of this organization's own history: %d%% of actual "+
				"counts fell inside the forecast range, and its misses were %d%% smaller than "+
				"repeating the same slot last week. Plans use it when Anticipate freight is on.",
			view.WeeksTested, view.IntervalCoveragePercent, view.SkillPercent,
		)
	case demandforecast.StatusNoSkill:
		view.Status = forecastStatusNoSkill
		view.Meaning = "Tested, but not yet better than repeating the same slot last week, " +
			"so plans ignore it. Say the forecast is not proven yet."
	case demandforecast.StatusInsufficientHistory:
		view.Status, view.Meaning = forecastStatusInsufficient, insufficientMeaning(&view)
	default:
		view.Status, view.Meaning = forecastStatusInsufficient, insufficientMeaning(&view)
	}

	return view
}

func outlookNote(ready bool, status string, count, hours int) string {
	if !ready {
		return fmt.Sprintf(
			"The forecast is not Ready (status %s), so plans do not use it and no windows "+
				"are listed: it has not yet shown it predicts this organization's freight "+
				"better than repeating last week. Say so and why, from quality.meaning; do "+
				"not guess where freight will appear.",
			status,
		)
	}
	if count == 0 {
		return fmt.Sprintf(
			"The forecast lists no area with freight beyond the drivers expected there in the "+
				"next %d hours.",
			hours,
		)
	}

	return fmt.Sprintf(
		"%d windows in the next %d hours, the most short of drivers first. Name the area, "+
			"the window and the shortfall; oneMoreDriverWorth is what a plan credits a driver "+
			"who ends there, and explanation says it in a sentence.",
		count, hours,
	)
}

func tenths(value float64) float64 {
	return math.Round(value*10) / 10
}

func percent(share float64) int {
	return int(math.Round(share * 100))
}

func insufficientMeaning(view *forecastQualityView) string {
	return fmt.Sprintf(
		"Not enough history to test yet: %d weeks and %d loads, where it needs at least "+
			"%d weeks and %d loads. Plans ignore it until then; say so.",
		view.HistoryWeeks, view.HistoryLoads,
		demandforecast.MinHistoryWeeks, demandforecast.MinHistoryLoads,
	)
}
