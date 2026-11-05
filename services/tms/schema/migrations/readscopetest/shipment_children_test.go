package readscopetest

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentChildrenFollowTheirShipment(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../20261231023154_read_scope_tracking_children.sql")
	require.NoError(t, err)

	children := []string{
		"tracking_events",
		"check_calls",
		"carrier_tracking_links",
		"customer_tracking_links",
		"customer_status_notifications",
		"stop_etas",
		"shipment_holds",
		"shipment_commodities",
		"additional_charges",
		"shipment_comment_mentions",
		"shipment_comment_acknowledgments",
		"permits",
		"permit_requirements",
	}

	policies := strings.Split(string(raw), "CREATE POLICY read_scope ON ")
	require.Len(t, policies, len(children)+1)

	for i, table := range children {
		policy := policies[i+1]
		assert.True(
			t,
			strings.HasPrefix(policy, "public."+table+" AS RESTRICTIVE FOR SELECT"),
			table,
		)
		assert.Contains(t, policy, "trenova_rls.read_scope('shipment')) <> 'own'", table)
		for _, column := range []string{"shipment_id", "organization_id", "business_unit_id"} {
			assert.Contains(t, policy, table+"."+column, table)
		}
		assert.Contains(t, policy, "s.owner_id = (SELECT trenova_rls.user_id())", table)
	}
}
