//go:build integration

package factoringrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (e *factoringEnv) overviewRepository() factoringports.OverviewRepository {
	return NewOverviewRepository(Params{DB: postgres.WrapDB(e.db), Logger: zap.NewNop()})
}

func (e *factoringEnv) factorCustomers() factoringports.FactorCustomerRepository {
	return NewFactorCustomerRepository(Params{DB: postgres.WrapDB(e.db), Logger: zap.NewNop()})
}

func (e *factoringEnv) readOverview(
	t *testing.T,
	factorID pulid.ID,
	now int64,
) *factoringports.OverviewRead {
	t.Helper()
	read, err := e.overviewRepository().Get(e.ctx, &factoringports.GetOverviewRequest{
		TenantInfo:         e.tenantInfo,
		FactoringCompanyID: factorID,
		At:                 now,
		ReadyBefore:        now - 86400,
		ChargebackBefore:   now + 7*86400,
		MovedSince:         now - 30*86400,
		PaceLimit:          20,
		AttentionLimit:     20,
	})
	require.NoError(t, err)
	return read
}

func (e *factoringEnv) listFactorCustomers(
	t *testing.T,
	factorID pulid.ID,
	stage factoringports.CustomerNoticeStage,
	now int64,
) *factoringports.FactorCustomerPage {
	t.Helper()
	page, err := e.factorCustomers().List(e.ctx, &factoringports.ListFactorCustomersRequest{
		Filter:             &pagination.QueryOptions{TenantInfo: e.tenantInfo},
		Cursor:             pagination.CursorInfo{Limit: 50, IncludeTotalCount: true},
		FactoringCompanyID: factorID,
		Stage:              stage,
		At:                 now,
	})
	require.NoError(t, err)
	return page
}

func customerIDs(page *factoringports.FactorCustomerPage) []pulid.ID {
	ids := make([]pulid.ID, len(page.Items))
	for i, item := range page.Items {
		ids[i] = item.CustomerID
	}
	return ids
}

func TestOverviewReadsOneFactorAndItsCustomersNotices(t *testing.T) {
	env := setupFactoringEnv(t)
	factor := env.createFactor(t, "OVRV")
	other := env.createFactor(t, "OTHR")
	now := time.Now().Unix()

	read := env.readOverview(t, factor.ID, now)
	assert.Equal(t, factor.Name, read.Factor.Name)
	assert.Equal(t, domaintypes.StatusActive, read.Factor.Status)
	assert.Equal(t, factoringports.NoticeCounts{}, read.Notices,
		"a factor nobody is assigned to has no notices to count")
	assert.Nil(t, read.Agreement)
	assert.Nil(t, read.Transport)
	assert.Empty(t, read.Funded)
	assert.Empty(
		t,
		env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeAll, now).Items,
	)

	env.nameFactorOnProfile(t, &factor.ID)

	read = env.readOverview(t, factor.ID, now)
	assert.Equal(t, factoringports.NoticeCounts{Assigned: 1, NeedsNotice: 1}, read.Notices)
	all := env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeAll, now)
	assert.Equal(t, []pulid.ID{env.customerID}, customerIDs(all))
	assert.Nil(t, all.Items[0].Notice)
	require.NotNil(t, all.TotalCount)
	assert.Equal(t, read.Notices.Assigned, *all.TotalCount, "the table counts what the overview counts")
	assert.Len(
		t,
		env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeNeeded, now).Items,
		1,
	)
	assert.Empty(
		t,
		env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeAwaiting, now).Items,
	)
	assert.Empty(
		t,
		env.listFactorCustomers(t, other.ID, factoringports.CustomerNoticeAll, now).Items,
		"a customer assigned to one factor is not another factor's customer",
	)

	noa, err := env.noas.Create(env.ctx, &factoring.Noa{
		OrganizationID:     env.tenantInfo.OrgID,
		BusinessUnitID:     env.tenantInfo.BuID,
		CustomerID:         env.customerID,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)

	read = env.readOverview(t, factor.ID, now)
	assert.Equal(t, factoringports.NoticeCounts{Assigned: 1, Awaiting: 1}, read.Notices)
	awaiting := env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeAwaiting, now)
	require.Len(t, awaiting.Items, 1)
	require.NotNil(t, awaiting.Items[0].Notice)
	assert.Equal(t, noa.ID, awaiting.Items[0].Notice.ID)
	assert.Empty(
		t,
		env.listFactorCustomers(t, factor.ID, factoringports.CustomerNoticeNeeded, now).Items,
	)

	_, err = env.agreements.Create(env.ctx, env.agreement(other.ID, now-86400, nil))
	require.NoError(t, err)
	assert.Nil(t, env.readOverview(t, factor.ID, now).Agreement,
		"another factor's agreement is not this factor's")

	created, err := env.agreements.Create(env.ctx, env.agreement(factor.ID, now-86400, nil))
	require.NoError(t, err)
	read = env.readOverview(t, factor.ID, now)
	require.NotNil(t, read.Agreement)
	assert.Equal(t, created.ID, read.Agreement.ID)
	assert.Len(t, read.Agreement.FeeTiers, 2, "the agreement comes with its fee tiers")
}

func TestOverviewOfAFactorThatDoesNotExistIsNotFound(t *testing.T) {
	env := setupFactoringEnv(t)
	now := time.Now().Unix()

	_, err := env.overviewRepository().Get(env.ctx, &factoringports.GetOverviewRequest{
		TenantInfo:         env.tenantInfo,
		FactoringCompanyID: pulid.MustNew("fac_"),
		At:                 now,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
