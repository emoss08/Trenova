package inboundjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type listedTenants struct {
	repositories.TenantSyncRepository

	organizations []tenant.SyncOrganization
}

func (t *listedTenants) ListOrganizations(context.Context) ([]tenant.SyncOrganization, error) {
	return t.organizations, nil
}

// scriptedPurger answers each tenant's passes from a queue of batch sizes.
type scriptedPurger struct {
	mu      sync.Mutex
	batches map[pulid.ID][]int
	fail    map[pulid.ID]bool
	asked   []inboundmessageservice.PurgeSettledRequest
}

func (p *scriptedPurger) PurgeSettled(
	_ context.Context,
	req inboundmessageservice.PurgeSettledRequest,
) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.asked = append(p.asked, req)
	if p.fail[req.TenantInfo.OrgID] {
		return 0, errors.New("database unavailable")
	}
	queue := p.batches[req.TenantInfo.OrgID]
	if len(queue) == 0 {
		return 0, nil
	}
	p.batches[req.TenantInfo.OrgID] = queue[1:]

	return queue[0], nil
}

func TestInboundMessageRetention_PagesEachTenantAndStepsOverAFailure(t *testing.T) {
	t.Parallel()

	busy, broken, quiet := pulid.MustNew("org_"), pulid.MustNew("org_"), pulid.MustNew("org_")
	purger := &scriptedPurger{
		batches: map[pulid.ID][]int{
			busy:  {inboundRetentionBatch, inboundRetentionBatch, 7},
			quiet: {0},
		},
		fail: map[pulid.ID]bool{broken: true},
	}
	activities := &Activities{
		purger: purger,
		tenants: &listedTenants{organizations: []tenant.SyncOrganization{
			{ID: busy, BusinessUnitID: pulid.MustNew("bu_")},
			{ID: broken, BusinessUnitID: pulid.MustNew("bu_")},
			{ID: quiet, BusinessUnitID: pulid.MustNew("bu_")},
		}},
		l: zap.NewNop(),
	}

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(activities.InboundMessageRetentionActivity)
	encoded, err := env.ExecuteActivity(activities.InboundMessageRetentionActivity)
	require.NoError(t, err)

	var result InboundMessageRetentionResult
	require.NoError(t, encoded.Get(&result))

	assert.Equal(t, 2*inboundRetentionBatch+7, result.Deleted)
	assert.Equal(t, []string{broken.String()}, result.Failed)

	perTenant := map[pulid.ID]int{}
	cutoff := timeutils.NowUnix() - inboundmessageservice.RetentionDays*24*60*60
	for _, req := range purger.asked {
		perTenant[req.TenantInfo.OrgID]++
		assert.Equal(t, inboundRetentionBatch, req.Limit)
		assert.InDelta(t, cutoff, req.Before, 60, "the window is six months back")
	}
	assert.Equal(t, 3, perTenant[busy], "a full batch means there may be more")
	assert.Equal(t, 1, perTenant[broken])
	assert.Equal(t, 1, perTenant[quiet])
}
