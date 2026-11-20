package inboundjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundupkeep"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type listedTenants struct {
	tenantports.TenantSyncRepository

	organizations []tenant.SyncOrganization
	needs         []entitlement.Need
}

func (f *listedTenants) ListTenantsPage(
	_ context.Context,
	need entitlement.Need,
	after pulid.ID,
	limit int,
) ([]pagination.TenantInfo, error) {
	f.needs = append(f.needs, need)
	page := make([]pagination.TenantInfo, 0, limit)
	passed := after.IsNil()
	for _, org := range f.organizations {
		if !passed {
			passed = org.ID == after
			continue
		}
		if len(page) == limit {
			break
		}
		page = append(page, pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID})
	}
	return page, nil
}

// scriptedPurger answers each tenant's passes from a queue of batch sizes.
type scriptedPurger struct {
	mu      sync.Mutex
	batches map[pulid.ID][]int
	fail    map[pulid.ID]bool
	asked   []inboundupkeep.PurgeSettledRequest
}

func (p *scriptedPurger) PurgeSettled(
	_ context.Context,
	req inboundupkeep.PurgeSettledRequest,
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
	assert.Equal(t,
		[]entitlement.Need{entitlement.For(platformcatalog.FeatureAITriage)},
		activities.tenants.(*listedTenants).needs,
		"retention pages only through the organizations with AI triage")

	perTenant := map[pulid.ID]int{}
	cutoff := timeutils.NowUnix() - inboundupkeep.RetentionDays*24*60*60
	for _, req := range purger.asked {
		perTenant[req.TenantInfo.OrgID]++
		assert.Equal(t, inboundRetentionBatch, req.Limit)
		assert.InDelta(t, cutoff, req.Before, 60, "the window is six months back")
	}
	assert.Equal(t, 3, perTenant[busy], "a full batch means there may be more")
	assert.Equal(t, 1, perTenant[broken])
	assert.Equal(t, 1, perTenant[quiet])
}
