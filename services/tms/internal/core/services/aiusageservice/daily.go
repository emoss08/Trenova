package aiusageservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	defaultDailyDays = 7
	maxDailyDays     = 90
	dayLayout        = "2006-01-02"
)

func (s *Service) Daily(
	ctx context.Context,
	req *services.AIUsageDailyRequest,
) ([]services.AIUsageDay, error) {
	days := req.Days
	switch {
	case days <= 0:
		days = defaultDailyDays
	case days > maxDailyDays:
		days = maxDailyDays
	}

	zone := req.Timezone
	if zone == "" {
		zone = time.UTC.String()
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"timezone", errortypes.ErrInvalid, "Unknown timezone",
		)
	}

	today := s.clock().In(location)
	first := time.Date(today.Year(), today.Month(), today.Day()-(days-1), 0, 0, 0, 0, location)

	rows, err := s.usage.Daily(ctx, repositories.AIUsageDailyRequest{
		TenantInfo: req.TenantInfo,
		Since:      first.Unix(),
		Timezone:   location.String(),
		ProviderID: req.ProviderID,
	})
	if err != nil {
		return nil, err
	}

	return fillDays(first, days, rows), nil
}

// fillDays lays the days that had usage over every day of the window, so a
// quiet day is a zero rather than a gap.
func fillDays(first time.Time, days int, rows []repositories.AIUsageDayTotals) []services.AIUsageDay {
	byDay := make(map[string]*repositories.AIUsageDayTotals, len(rows))
	for idx := range rows {
		byDay[rows[idx].Day] = &rows[idx]
	}

	out := make([]services.AIUsageDay, 0, days)
	for offset := range days {
		day := first.AddDate(0, 0, offset).Format(dayLayout)
		entry := services.AIUsageDay{Day: day, CostUSD: "0"}
		if row, ok := byDay[day]; ok {
			entry.Calls = row.Calls
			entry.Failed = row.Failed
			entry.InputTokens = row.InputTokens
			entry.OutputTokens = row.OutputTokens
			entry.CostUSD = row.CostUSD
			entry.PricedCalls = row.PricedCalls
			entry.LatencyP50Ms = row.LatencyP50
			entry.LatencyP95Ms = row.LatencyP95
		}
		out = append(out, entry)
	}
	return out
}
