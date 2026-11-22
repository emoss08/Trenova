package tablechangealertservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type askedSubscriptions struct {
	tablechangealertports.TCASubscriptionRepository

	asked []pulid.ID
}

func (a *askedSubscriptions) FindMatchingSubscriptions(
	_ context.Context,
	req tablechangealertports.FindMatchingTCASubscriptionsRequest,
) ([]*tablechangealert.TCASubscription, error) {
	a.asked = append(a.asked, req.OrganizationID)
	return nil, nil
}

func changeOf(t *testing.T, orgID, buID pulid.ID) redis.XMessage {
	t.Helper()
	payload, err := sonic.MarshalString(map[string]any{
		"operation": "INSERT",
		"table":     "shipments",
		"new_data": map[string]any{
			"id":               "shp_1",
			"organization_id":  orgID.String(),
			"business_unit_id": buID.String(),
		},
	})
	require.NoError(t, err)
	return redis.XMessage{ID: "1-0", Values: map[string]any{"payload": payload}}
}

func TestAChangeIsMatchedOnlyForAnOrganizationWithChangeAlerts(t *testing.T) {
	t.Parallel()

	with, without := plantest.Tenant(), plantest.Tenant()
	subs := &askedSubscriptions{}
	consumer := &Consumer{
		l:       zap.NewNop(),
		subRepo: subs,
		features: plantest.SharedServer().
			Planned(with.OrgID).
			Without(without.OrgID, platformcatalog.FeatureTableChangeAlerts),
	}

	require.NoError(
		t,
		consumer.processMessage(t.Context(), changeOf(t, without.OrgID, without.BuID)),
	)
	assert.Empty(t, subs.asked, "nothing is matched for an organization without change alerts")

	require.NoError(t, consumer.processMessage(t.Context(), changeOf(t, with.OrgID, with.BuID)))
	assert.Equal(t, []pulid.ID{with.OrgID}, subs.asked)
}
