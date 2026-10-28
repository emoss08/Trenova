package boardcardservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
)

const (
	minorExponent  = -2
	percentDivisor = 100
)

var receivableMetrics = []string{"arTotalOpen", "arOverdue", "arDaysSalesOutstanding"}

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

func (s *Service) metricAllowed(
	ctx context.Context,
	actor *permissionports.RequestActor,
	key string,
) (bool, error) {
	definition, ok := homelayout.MetricDefinitionFor(key)
	if !ok {
		return false, nil
	}

	return s.allowed(ctx, actor, definition.Resource, definition.Operation)
}

func (s *Service) readMetrics(
	ctx context.Context,
	actor *permissionports.RequestActor,
	keys []string,
	reads *board.Reads,
) error {
	wanted := make([]string, 0, len(keys))
	for _, key := range keys {
		allowed, err := s.metricAllowed(ctx, actor, key)
		if err != nil {
			return err
		}
		if allowed {
			wanted = append(wanted, key)
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	if slices.ContainsFunc(wanted, func(key string) bool {
		return !slices.Contains(receivableMetrics, key)
	}) {
		cards, err := s.shipmentCards(ctx, actor)
		if err != nil {
			return err
		}
		for _, key := range wanted {
			if value := shipmentMetric(key, cards); value != nil {
				reads.Metrics[key] = value
			}
		}
	}

	if slices.ContainsFunc(wanted, func(key string) bool {
		return slices.Contains(receivableMetrics, key)
	}) {
		kpis, err := s.receivables.GetDashboardKPIs(ctx, customerledgerports.GetARAnalyticsRequest{
			TenantInfo: actor.TenantInfo(),
		})
		if err != nil {
			return fmt.Errorf("read the receivables figures: %w", err)
		}
		for _, key := range wanted {
			if value := receivableMetric(key, kpis); value != nil {
				reads.Metrics[key] = value
			}
		}
	}

	return nil
}

func (s *Service) shipmentCards(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (*shipmentCards, error) {
	tenant := actor.TenantInfo()
	data, err := s.analytics.GetAnalytics(ctx, &kernelports.AnalyticsRequestOptions{
		OrgID:  tenant.OrgID,
		BuID:   tenant.BuID,
		UserID: tenant.UserID,
		Principal: kernelports.PrincipalInfo{
			Type:     actor.PrincipalType,
			ID:       actor.PrincipalID,
			UserID:   actor.UserID,
			APIKeyID: actor.APIKeyID,
		},
		Page:     kernelports.ShipmentAnalyticsPage,
		Timezone: s.Location(ctx, tenant).String(),
		Include:  kernelports.AnalyticsIncludeMetrics,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment figures: %w", err)
	}
	cards := &shipmentCards{}
	if err = jsonutils.Convert(data, cards); err != nil {
		return nil, fmt.Errorf("read the shipment figures: %w", err)
	}

	return cards, nil
}

func count(n int64) *board.Value {
	return &board.Value{Kind: board.ValueCount, Number: decimal.NewFromInt(n)}
}

func dollars(amount float64) *board.Value {
	return &board.Value{Kind: board.ValueMoney, Number: decimal.NewFromFloat(amount).Round(2)}
}

func minor(amount int64) *board.Value {
	return &board.Value{Kind: board.ValueMoney, Number: decimal.New(amount, minorExponent)}
}

func percent(points float64) *board.Value {
	return &board.Value{
		Kind:   board.ValuePercent,
		Number: decimal.NewFromFloat(points).Div(decimal.NewFromInt(percentDivisor)),
	}
}

func shipmentMetric(key string, cards *shipmentCards) *board.Value {
	switch key {
	case "activeShipments":
		return count(cards.ActiveShipments.Count)
	case "revenueToday":
		return dollars(cards.RevenueToday.Total)
	case "onTimePercent":
		return percent(cards.OnTimePercent.Percent)
	case "emptyMilePercent":
		return percent(cards.EmptyMilePercent.Percent)
	case "atRisk":
		return count(cards.AtRisk.Count)
	case "unassigned":
		return count(cards.Unassigned.Count)
	case "readyToDispatch":
		return count(cards.ReadyToDispatch.Count)
	case "marginPercent":
		if cards.Profitability.AvgMarginPct == nil {
			return nil
		}
		return percent(*cards.Profitability.AvgMarginPct)
	case "revenuePerMile":
		return dollars(cards.RevenueToday.RPM)
	case "costPerMile":
		return dollars(cards.Profitability.AvgCPM)
	case "movesLate":
		return count(cards.Dispatch.Late)
	case "movesDueNow":
		return count(cards.Dispatch.DueNow)
	case "movesDueToday":
		return count(cards.Dispatch.DueToday)
	case "movesUncovered":
		return count(cards.Dispatch.Uncovered)
	default:
		return nil
	}
}

func receivableMetric(key string, kpis *customerledgerports.ARDashboardKPIs) *board.Value {
	if kpis == nil {
		return nil
	}
	switch key {
	case "arTotalOpen":
		if kpis.Overview == nil {
			return nil
		}
		return minor(kpis.Overview.TotalOpenMinor)
	case "arOverdue":
		if kpis.Overview == nil {
			return nil
		}
		return minor(kpis.Overview.OverdueMinor)
	case "arDaysSalesOutstanding":
		return &board.Value{
			Kind:   board.ValueNumber,
			Number: decimal.NewFromFloat(kpis.CurrentDSODays).Round(1),
		}
	default:
		return nil
	}
}
