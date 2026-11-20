package briefingjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestListDueOrganizationsReadsOnlyTheOrganizationsWithTheDesk(t *testing.T) {
	t.Parallel()

	tenants := tenantportsmock.NewMockTenantSyncRepository(t)
	tenants.EXPECT().
		ListTenantsPage(mock.Anything, entitlement.For(platformcatalog.FeatureAIDesk), mock.Anything, mock.Anything).
		Return([]pagination.TenantInfo{}, nil).
		Once()

	a := &Activities{tenants: tenants, l: zap.NewNop()}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a.ListDueOrganizationsActivity)

	value, err := env.ExecuteActivity(a.ListDueOrganizationsActivity, &DueOrganizationsInput{
		Now: time.Now().Unix(),
	})
	require.NoError(t, err)

	var due DueOrganizations
	require.NoError(t, value.Get(&due))
	assert.Empty(t, due.Due)
}
