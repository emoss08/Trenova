package agentproposalrepository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func moneyTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func moneyScope(tenant pagination.TenantInfo) string {
	return `SELECT COUNT(*) AS calls, COALESCE(SUM(ABS(ap.money_minor)), 0) AS amount_minor ` +
		`FROM "agent_proposals" AS "ap" WHERE ((ap.organization_id = '` + tenant.OrgID.String() +
		`') AND (ap.business_unit_id = '` + tenant.BuID.String() +
		`') AND (ap.autonomy_tier = 'AutoExecute') `
}

func TestSumAutomaticMoney_CountsOnlyWritesThatRanInTheFunctionalCurrency(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenant := moneyTenant()

	sqlMock.ExpectQuery(regexp.QuoteMeta(moneyScope(tenant)+
		`AND (ap.status = 'Executed') AND (ap.money_minor IS NOT NULL) `+
		`AND (ap.money_currency = 'USD') AND (ap.executed_at IS NOT NULL) `+
		`AND (ap.executed_at >= 100))`)).
		WillReturnRows(sqlmock.NewRows([]string{"calls", "amount_minor"}).AddRow(3, 45_000))

	used, err := repo.SumAutomaticMoney(t.Context(), &agentports.SumAutomaticMoneyRequest{
		TenantInfo: tenant,
		Currency:   " usd ",
		Since:      100,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MoneyUse{Calls: 3, AmountMinor: 45_000}, used)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestSumAutomaticMoney_ASimulatedRunCountsItsSimulatedWrites(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenant := moneyTenant()
	runID := pulid.MustNew("arun_")

	sqlMock.ExpectQuery(regexp.QuoteMeta(moneyScope(tenant)+
		`AND (ap.status = 'Simulated') AND (ap.money_minor IS NOT NULL) `+
		`AND (ap.money_currency = 'CAD') AND (ap.simulated_at IS NOT NULL) `+
		`AND (ap.run_id = '`+runID.String()+`'))`)).
		WillReturnRows(sqlmock.NewRows([]string{"calls", "amount_minor"}).AddRow(1, 2_000))

	used, err := repo.SumAutomaticMoney(t.Context(), &agentports.SumAutomaticMoneyRequest{
		TenantInfo: tenant,
		Currency:   "CAD",
		RunID:      runID,
		Simulated:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MoneyUse{Calls: 1, AmountMinor: 2_000}, used)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestSumAutomaticMoney_RefusesToCountWithoutACurrency(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}

	_, err := repo.SumAutomaticMoney(t.Context(), &agentports.SumAutomaticMoneyRequest{
		TenantInfo: moneyTenant(),
		Currency:   " ",
	})
	require.ErrorIs(t, err, errMoneyCurrencyRequired)
	require.NoError(t, sqlMock.ExpectationsWereMet(), "nothing is read")
}
