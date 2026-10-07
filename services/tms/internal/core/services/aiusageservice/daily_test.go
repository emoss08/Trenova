package aiusageservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dailyUsage struct {
	repositories.AIUsageRepository
	asked repositories.AIUsageDailyRequest
	rows  []repositories.AIUsageDayTotals
}

func (d *dailyUsage) Daily(
	_ context.Context,
	req repositories.AIUsageDailyRequest,
) ([]repositories.AIUsageDayTotals, error) {
	d.asked = req
	return d.rows, nil
}

func TestDailyFillsQuietDaysInTheAskedTimezone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	usage := &dailyUsage{rows: []repositories.AIUsageDayTotals{
		{Day: "2026-10-05", AIUsageTotals: repositories.AIUsageTotals{Calls: 4, Failed: 1, CostUSD: "0.12"}},
	}}
	svc := &Service{usage: usage, clock: func() time.Time {
		return time.Date(2026, 10, 7, 3, 0, 0, 0, time.UTC)
	}}

	days, err := svc.Daily(t.Context(), &services.AIUsageDailyRequest{Days: 3, Timezone: "America/Chicago"})

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 4, 0, 0, 0, 0, chicago).Unix(), usage.asked.Since,
		"at 3am UTC it is still the 6th in Chicago")
	require.Len(t, days, 3)
	assert.Equal(t, []string{"2026-10-04", "2026-10-05", "2026-10-06"},
		[]string{days[0].Day, days[1].Day, days[2].Day})
	assert.Equal(t, 0, days[0].Calls)
	assert.Equal(t, "0", days[0].CostUSD)
	assert.Equal(t, 4, days[1].Calls)
	assert.Equal(t, "0.12", days[1].CostUSD)
}

func TestDailyRefusesAnUnknownTimezone(t *testing.T) {
	t.Parallel()

	svc := &Service{usage: &dailyUsage{}, clock: time.Now}

	_, err := svc.Daily(t.Context(), &services.AIUsageDailyRequest{Timezone: "Mars/Olympus"})

	require.Error(t, err)
}
