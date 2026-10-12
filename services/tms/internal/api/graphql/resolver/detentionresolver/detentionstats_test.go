package detentionresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetentionStatsRequest_NarrowsToTheLocationAsked(t *testing.T) {
	t.Parallel()

	locationID := pulid.MustNew("loc_")
	raw := locationID.String()
	limit := 1
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	req, err := detentionStatsRequest(&gqlmodel.DetentionStatsInput{
		From:       100,
		To:         200,
		Limit:      &limit,
		LocationID: &raw,
	}, tenant)
	require.NoError(t, err)

	assert.Equal(t, locationID, req.LocationID)
	assert.Equal(t, int64(100), req.From)
	assert.Equal(t, int64(200), req.To)
	assert.Equal(t, 1, req.Limit)
	assert.Equal(t, tenant, req.TenantInfo)
}

func TestDetentionStatsRequest_LeavesEveryLocationInWhenNoneIsAsked(t *testing.T) {
	t.Parallel()

	empty := ""
	for _, locationID := range []*string{nil, &empty} {
		req, err := detentionStatsRequest(&gqlmodel.DetentionStatsInput{
			From:       100,
			To:         200,
			LocationID: locationID,
		}, pagination.TenantInfo{})
		require.NoError(t, err)
		assert.True(t, req.LocationID.IsNil())
	}
}

func TestDetentionStatsRequest_RejectsAMalformedLocationID(t *testing.T) {
	t.Parallel()

	raw := "dallas-yard"
	_, err := detentionStatsRequest(&gqlmodel.DetentionStatsInput{LocationID: &raw}, pagination.TenantInfo{})
	require.Error(t, err)
}
