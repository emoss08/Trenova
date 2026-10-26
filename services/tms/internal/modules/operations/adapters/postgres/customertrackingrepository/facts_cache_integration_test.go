//go:build integration

package customertrackingrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/customertrackingrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotifyingCustomersAreCachedPerTenantUntilAProfileChanges(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "NC-1", shipment.StatusInTransit)
	other := seedtest.SeedAdditionalTenant(t, t.Context(), h.db, "NC")

	cached := cachetest.Open(t, t.Context(), h.dsn)
	params := customertrackingrepository.Params{
		DB:     postgres.NewTestConnection(cached),
		Logger: zap.NewNop(),
	}
	facts := customertrackingrepository.NewFacts(params)
	profiles := customertrackingrepository.NewProfiles(params)

	var notifying bool
	readFor := func(tenant pagination.TenantInfo) func() error {
		return func() error {
			var readErr error
			notifying, readErr = facts.HasNotifyingCustomers(context.Background(), tenant)
			return readErr
		}
	}

	t.Run("an organization without rules is cached as quiet", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(h.tenant))
		assert.False(t, notifying)
		cachetest.RequireHit(t, cached, readFor(h.tenant))
		assert.False(t, notifying)
	})

	t.Run("turning a customer's emails on invalidates the answer", func(t *testing.T) {
		profile := customertracking.DefaultProfile(
			h.tenant.OrgID,
			h.tenant.BuID,
			graph.Shipment.CustomerID,
		)
		profile.ID = customertracking.NewProfileID()
		profile.CreatedAt, profile.UpdatedAt = h.now, h.now
		profile.NotifyDelivered = true
		enabled := h.now - 3600
		profile.NotificationsEnabledAt = &enabled
		require.NoError(t, profiles.Create(h.ctx, profile))

		cachetest.RequireMiss(t, cached, readFor(h.tenant))
		assert.True(t, notifying)
		cachetest.RequireHit(t, cached, readFor(h.tenant))
		assert.True(t, notifying)
	})

	t.Run("another organization keeps its own answer", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(other.Info()))
		assert.False(t, notifying)
	})
}
