package homeboardreader

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const percentDivisor = 100

type Params struct {
	fx.In

	Analytics kernelports.AnalyticsService
}

type Dispatch struct {
	analytics kernelports.AnalyticsService
}

func NewDispatch(p Params) *Dispatch {
	return &Dispatch{analytics: p.Analytics}
}

func (*Dispatch) Metric() string { return homeboard.MetricDispatchToday }

type shipmentCards struct {
	ActiveShipments struct {
		Count int64 `json:"count"`
	} `json:"activeShipments"`
	RevenueToday struct {
		Total float64 `json:"total"`
		RPM   float64 `json:"rpm"`
	} `json:"revenueToday"`
	OnTimePercent struct {
		Percent float64 `json:"percent"`
	} `json:"onTimePercent"`
	EmptyMilePercent struct {
		Percent float64 `json:"percent"`
	} `json:"emptyMilePercent"`
	AtRisk struct {
		Count int64 `json:"count"`
	} `json:"atRisk"`
	Unassigned struct {
		Count int64 `json:"count"`
	} `json:"unassigned"`
	ReadyToDispatch struct {
		Count int64 `json:"count"`
	} `json:"readyToDispatch"`
	Profitability struct {
		AvgCPM       float64  `json:"avgCpm"`
		AvgMarginPct *float64 `json:"avgMarginPct"`
	} `json:"profitability"`
	Dispatch struct {
		Late      int64 `json:"late"`
		DueNow    int64 `json:"dueNow"`
		DueToday  int64 `json:"dueToday"`
		Uncovered int64 `json:"uncovered"`
	} `json:"dispatch"`
}

func (r *Dispatch) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	options := &kernelports.AnalyticsRequestOptions{
		OrgID:   req.Tenant.OrgID,
		BuID:    req.Tenant.BuID,
		UserID:  req.Tenant.UserID,
		Page:    kernelports.ShipmentAnalyticsPage,
		Include: kernelports.AnalyticsIncludeMetrics,
	}
	if req.Location != nil {
		options.Timezone = req.Location.String()
	}
	if actor := req.Actor; actor != nil {
		options.Principal = kernelports.PrincipalInfo{
			Type:     actor.PrincipalType,
			ID:       actor.PrincipalID,
			UserID:   actor.UserID,
			APIKeyID: actor.APIKeyID,
		}
	}
	data, err := r.analytics.GetAnalytics(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("read the shipment figures: %w", err)
	}
	cards := &shipmentCards{}
	if err = jsonutils.Convert(data, cards); err != nil {
		return nil, fmt.Errorf("read the shipment figures: %w", err)
	}

	return &boardcatalog.MetricRead{
		Title:   i18n.T(ctx, "Today's dispatch"),
		Source:  i18n.T(ctx, "the home page's dispatch figures"),
		Figures: figuresOf(cards),
	}, nil
}

func count(n int64) *boardcatalog.Value {
	value := boardcatalog.Count(n)

	return &value
}

func dollars(amount float64) *boardcatalog.Value {
	value := boardcatalog.Amount(decimal.NewFromFloat(amount).Round(2), "")

	return &value
}

func percent(points float64) *boardcatalog.Value {
	value := boardcatalog.Percent(
		decimal.NewFromFloat(points).Div(decimal.NewFromInt(percentDivisor)),
	)

	return &value
}

func figuresOf(cards *shipmentCards) map[string]*boardcatalog.Value {
	figures := map[string]*boardcatalog.Value{
		"activeShipments":  count(cards.ActiveShipments.Count),
		"revenueToday":     dollars(cards.RevenueToday.Total),
		"onTimePercent":    percent(cards.OnTimePercent.Percent),
		"emptyMilePercent": percent(cards.EmptyMilePercent.Percent),
		"atRisk":           count(cards.AtRisk.Count),
		"unassigned":       count(cards.Unassigned.Count),
		"readyToDispatch":  count(cards.ReadyToDispatch.Count),
		"marginPercent":    nil,
		"revenuePerMile":   dollars(cards.RevenueToday.RPM),
		"costPerMile":      dollars(cards.Profitability.AvgCPM),
		"movesLate":        count(cards.Dispatch.Late),
		"movesDueNow":      count(cards.Dispatch.DueNow),
		"movesDueToday":    count(cards.Dispatch.DueToday),
		"movesUncovered":   count(cards.Dispatch.Uncovered),
	}
	if cards.Profitability.AvgMarginPct != nil {
		figures["marginPercent"] = percent(*cards.Profitability.AvgMarginPct)
	}

	return figures
}
