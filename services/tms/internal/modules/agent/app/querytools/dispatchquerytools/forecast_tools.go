package dispatchquerytools

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramHoursAhead      = "hoursAhead"
	defaultForecastHours = 24
	maxForecastHours     = 48
	defaultForecastLimit = 8
	maxForecastLimit     = 20

	forecastStatusReady        = "Ready"
	forecastStatusNoSkill      = "NoSkill"
	forecastStatusInsufficient = "InsufficientHistory"
	forecastStatusNotFitted    = "NotFitted"

	notFittedYet     = "not fitted yet"
	refittingMeaning = " The history is being read again right now, so these figures may " +
		"change within a minute; say so, and read the outlook again before relying on them."
	unnamedArea = "an area with no named origin"
)

type freightOutlookReader interface {
	Outlook(
		ctx context.Context,
		req *freightoutlookports.Request,
	) (*freightoutlookports.Outlook, error)
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
		"many loads usually appear and which regions will need more trucks. Learned from " +
		"this organization's own shipment history, not market data: for each area and " +
		"four-hour window it gives loads likely, what is booked, the drivers expected and the " +
		"loads with no truck nearby, then ranked moves (send an empty driver, book a load " +
		"that ends there, keep a driver, call a customer behind its usual pace) with what " +
		"each is worth in dollars. Plans use it only once quality.status is Ready; until then " +
		"say it is not proven yet and why, from quality.meaning, rather than guessing."
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
			toolschema.KeyMinimum: 4,
			toolschema.KeyMaximum: maxForecastHours,
			toolschema.KeyDescription: fmt.Sprintf(
				"How many hours ahead to look, from now: %d unless you ask, at most %d.",
				defaultForecastHours, maxForecastHours,
			),
		}),
		querytool.ParamLimit: querytool.LimitSchema(defaultForecastLimit, maxForecastLimit),
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

type forecastAreaView struct {
	Area               string   `json:"area"`
	WindowStart        string   `json:"windowStart"`
	WindowEnd          string   `json:"windowEnd"`
	LikelyLow          int      `json:"likelyLow"`
	LikelyHigh         int      `json:"likelyHigh"`
	Booked             int      `json:"booked"`
	DriversExpected    int      `json:"driversExpected"`
	LoadsWithNoTruck   float64  `json:"loadsWithNoTruck,omitempty"`
	DriversToSpare     float64  `json:"driversToSpare,omitempty"`
	OneMoreDriverWorth string   `json:"oneMoreDriverWorth"`
	Customers          []string `json:"customers"`
}

type forecastMoveView struct {
	Kind        string  `json:"kind"`
	Driver      string  `json:"driver,omitempty"`
	From        string  `json:"from,omitempty"`
	To          string  `json:"to"`
	Window      string  `json:"window"`
	Miles       float64 `json:"emptyMiles,omitempty"`
	Worth       string  `json:"worth,omitempty"`
	ProNumber   string  `json:"proNumber,omitempty"`
	HandTo      string  `json:"handTo,omitempty"`
	Customer    string  `json:"customer,omitempty"`
	LoadsMissed float64 `json:"loadsNotTendered,omitempty"`
}

type freightOutlookView struct {
	Quality     forecastQualityView `json:"quality"`
	Hours       int                 `json:"hoursAhead"`
	GeneratedAt string              `json:"generatedAt"`
	Short       []forecastAreaView  `json:"shortOfTrucks"`
	Spare       []forecastAreaView  `json:"trucksToSpare"`
	Moves       []forecastMoveView  `json:"moves"`
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
	limit := querytool.ReadLimit(params.Params, defaultForecastLimit, maxForecastLimit)
	outlook, err := t.outlook.Outlook(ctx, &freightoutlookports.Request{
		TenantInfo: querytool.TenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	return freightOutlookViewOf(outlook, hours, limit, timeutils.LoadLocation(params.Timezone)), nil
}

func forecastHours(params map[string]any) int {
	hours := querytool.OptionalInt(params, paramHoursAhead, defaultForecastHours)
	if hours <= 0 {
		return defaultForecastHours
	}

	return min(hours, maxForecastHours)
}

func freightOutlookViewOf(
	outlook *freightoutlookports.Outlook,
	hours, limit int,
	location *time.Location,
) *freightOutlookView {
	if outlook == nil {
		outlook = &freightoutlookports.Outlook{}
	}
	view := &freightOutlookView{
		Quality: forecastQualityViewOf(outlook.Quality, outlook.Refitting, location),
		Hours:   hours,
		Short:   []forecastAreaView{},
		Spare:   []forecastAreaView{},
		Moves:   []forecastMoveView{},
	}
	if outlook.GeneratedAt > 0 {
		view.GeneratedAt = timeutils.FormatStamp(outlook.GeneratedAt, location)
	}
	ready := view.Quality.Status == forecastStatusReady
	if ready {
		slots := min(int(math.Ceil(float64(hours)/4)), demandforecast.OutlookSlots)
		for _, area := range outlook.Areas {
			if short, ok := forecastAreaViewOf(area, slots, true, location); ok &&
				len(view.Short) < limit {
				view.Short = append(view.Short, short)
			}
			if spare, ok := forecastAreaViewOf(area, slots, false, location); ok &&
				len(view.Spare) < limit {
				view.Spare = append(view.Spare, spare)
			}
		}
		cutoff := outlook.GeneratedAt + int64(hours)*3600
		for _, step := range outlook.Steps {
			if step.SlotStart >= cutoff || len(view.Moves) >= limit {
				continue
			}
			view.Moves = append(view.Moves, forecastMoveViewOf(&step, location))
		}
	}
	view.Note = outlookNote(ready, view.Quality.Status, len(view.Short), hours)

	return view
}

func forecastAreaViewOf(
	area *freightoutlookports.Area,
	slots int,
	short bool,
	location *time.Location,
) (forecastAreaView, bool) {
	best, at := 0.0, -1
	for i, slot := range area.Balance.Slots[:min(slots, len(area.Balance.Slots))] {
		value := slot.Spare
		if short {
			value = slot.Short
		}
		if value > best {
			best, at = value, i
		}
	}
	if at < 0 || best < freightmove.MinShort {
		return forecastAreaView{}, false
	}

	slot := area.Balance.Slots[at]
	view := forecastAreaView{
		Area:               cmp.Or(area.Balance.Label, unnamedArea),
		WindowStart:        timeutils.FormatStamp(slot.Start, location),
		WindowEnd:          timeutils.FormatStamp(slot.End, location),
		LikelyLow:          slot.Low,
		LikelyHigh:         slot.High,
		Booked:             slot.Booked,
		DriversExpected:    slot.Drivers,
		OneMoreDriverWorth: dispatcheconomics.Dollars(area.ValueCents),
		Customers:          make([]string, 0, len(area.Customers)),
	}
	if short {
		view.LoadsWithNoTruck = tenths(best)
	} else {
		view.DriversToSpare = tenths(best)
	}
	for _, customer := range area.Customers {
		view.Customers = append(view.Customers, customer.Name)
	}

	return view, true
}

func forecastMoveViewOf(step *freightmove.Step, location *time.Location) forecastMoveView {
	view := forecastMoveView{
		Kind:      string(step.Kind),
		Driver:    step.WorkerName,
		From:      step.FromLabel,
		To:        cmp.Or(step.ToLabel, unnamedArea),
		Window:    timeutils.FormatStamp(step.SlotStart, location),
		Miles:     math.Round(step.Miles),
		ProNumber: step.ProNumber,
		Customer:  step.CustomerName,
	}
	switch step.Kind {
	case freightmove.KindCall:
		view.Driver, view.From = "", ""
		view.LoadsMissed = tenths(step.Loads)
	case freightmove.KindKeep:
		view.HandTo = step.PartnerName
		view.Worth = dispatcheconomics.Dollars(step.NetCents)
	case freightmove.KindSend, freightmove.KindBook:
		view.Worth = dispatcheconomics.Dollars(step.NetCents)
	}

	return view
}

func forecastQualityViewOf(
	quality *demandforecast.Quality,
	refitting bool,
	location *time.Location,
) forecastQualityView {
	view := fittedQualityViewOf(quality, location)
	if refitting {
		view.Meaning += refittingMeaning
	}

	return view
}

func fittedQualityViewOf(
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
			"Tested on the last %d weeks of this organization's own history: %d%% of the "+
				"tested days' load totals fell inside the forecast range, and its misses were "+
				"%d%% smaller than "+
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
			"The forecast is not Ready (status %s), so plans do not use it and nothing is "+
				"listed: it has not yet shown it predicts this organization's freight better "+
				"than repeating last week. Say so and why, from quality.meaning; do not guess "+
				"where freight will appear.",
			status,
		)
	}
	if count == 0 {
		return fmt.Sprintf(
			"No area is short of trucks in the next %d hours. trucksToSpare lists where "+
				"drivers will wait with little to haul.",
			hours,
		)
	}

	return fmt.Sprintf(
		"%d areas run short of trucks in the next %d hours, the worst window of each. moves "+
			"are ranked by what each is worth: Send an empty driver, Book a load ending there, "+
			"Keep a driver whose next load would leave a short area (handTo takes it), or Call "+
			"a customer behind its usual pace. They act through the freight outlook page.",
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
		demandfit.MinHistoryWeeks, demandfit.MinHistoryLoads,
	)
}
