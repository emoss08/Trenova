package aiproviderspendservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usageRepo struct {
	repositories.AIUsageRepository
	spend    map[pulid.ID]decimal.Decimal
	requests []repositories.AIUsageProviderSpendRequest
	fail     error
}

func (u *usageRepo) SpendByProvider(
	_ context.Context,
	req repositories.AIUsageProviderSpendRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	u.requests = append(u.requests, req)
	if u.fail != nil {
		return nil, u.fail
	}
	out := make(map[pulid.ID]decimal.Decimal, len(req.ProviderIDs))
	for _, id := range req.ProviderIDs {
		if spend, ok := u.spend[id]; ok {
			out[id] = spend
		}
	}

	return out, nil
}

func TestMonthSpend_SumsFromTheStartOfTheUTCMonthForEveryProviderAtOnce(t *testing.T) {
	t.Parallel()

	spent := pulid.MustNew("aiprv_")
	idle := pulid.MustNew("aiprv_")
	repo := &usageRepo{spend: map[pulid.ID]decimal.Decimal{spent: decimal.RequireFromString("12.34")}}
	now := time.Date(2026, time.October, 8, 15, 0, 0, 0, time.UTC)
	svc := newService(repo, func() time.Time { return now })

	got, err := svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{spent, idle, spent})
	require.NoError(t, err)

	assert.True(t, got[spent].Equal(decimal.RequireFromString("12.34")))
	assert.True(t, got[idle].IsZero())
	require.Len(t, repo.requests, 1)
	assert.ElementsMatch(t, []pulid.ID{spent, idle}, repo.requests[0].ProviderIDs)
	assert.Equal(t, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC).Unix(), repo.requests[0].Since)
}

func TestMonthSpend_ServesFromCacheForAMinute(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aiprv_")
	repo := &usageRepo{spend: map[pulid.ID]decimal.Decimal{id: decimal.NewFromInt(5)}}
	now := time.Date(2026, time.October, 8, 15, 0, 0, 0, time.UTC)
	svc := newService(repo, func() time.Time { return now })

	_, err := svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{id})
	require.NoError(t, err)
	repo.spend[id] = decimal.NewFromInt(9)

	now = now.Add(59 * time.Second)
	got, err := svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{id})
	require.NoError(t, err)
	assert.True(t, got[id].Equal(decimal.NewFromInt(5)))
	assert.Len(t, repo.requests, 1)

	now = now.Add(2 * time.Second)
	got, err = svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{id})
	require.NoError(t, err)
	assert.True(t, got[id].Equal(decimal.NewFromInt(9)))
	assert.Len(t, repo.requests, 2)
}

func TestMonthSpend_StartsAgainWhenTheMonthTurns(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aiprv_")
	repo := &usageRepo{spend: map[pulid.ID]decimal.Decimal{id: decimal.NewFromInt(5)}}
	now := time.Date(2026, time.October, 31, 23, 59, 50, 0, time.UTC)
	svc := newService(repo, func() time.Time { return now })

	_, err := svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{id})
	require.NoError(t, err)

	now = now.Add(20 * time.Second)
	_, err = svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{id})
	require.NoError(t, err)
	require.Len(t, repo.requests, 2)
	assert.Equal(t, time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC).Unix(), repo.requests[1].Since)
}

func TestMonthSpend_ReturnsTheRepositoryError(t *testing.T) {
	t.Parallel()

	repo := &usageRepo{fail: errors.New("database down")}
	svc := newService(repo, time.Now)

	_, err := svc.MonthSpend(t.Context(), pagination.TenantInfo{}, []pulid.ID{pulid.MustNew("aiprv_")})
	require.Error(t, err)
}
