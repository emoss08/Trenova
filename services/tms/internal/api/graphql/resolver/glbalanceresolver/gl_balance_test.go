package glbalanceresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/accounting/app/glbalanceservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glbalanceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks/glbalanceportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGLBalanceQueries_CheckTheAccountingReportPermission(t *testing.T) {
	t.Parallel()

	periodID := pulid.MustNew("fp_").String()

	calls := []struct {
		name string
		run  func(ctx context.Context, deps *Deps) error
	}{
		{"trialBalance", func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TrialBalance(ctx, periodID)
			return err
		}},
		{"incomeStatement", func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).IncomeStatement(ctx, periodID)
			return err
		}},
		{"balanceSheet", func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).BalanceSheet(ctx, periodID)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceAccountingReport,
				permission.OpRead,
			)
		})
	}
}

func TestIncomeStatement_ReadsThePeriodInTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	periodID := pulid.MustNew("fp_")
	repo := glbalanceportsmock.NewMockGLBalanceRepository(t)
	repo.EXPECT().
		ListTrialBalanceByPeriod(mock.Anything, glbalanceports.ListTrialBalanceByPeriodRequest{
			TenantInfo:     resolvercore.TenantInfo(authCtx),
			FiscalPeriodID: periodID,
		}).
		Return([]*glbalanceports.GLPeriodAccountBalance{
			{
				AccountCode:       "4000",
				AccountName:       "Freight revenue",
				AccountCategory:   accounttype.CategoryRevenue,
				PeriodCreditMinor: 900000,
			},
			{
				AccountCode:      "6000",
				AccountName:      "Fuel",
				AccountCategory:  accounttype.CategoryExpense,
				PeriodDebitMinor: 250000,
			},
		}, nil)

	engine := &resolvertest.RecordingPermissionEngine{}
	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		GLBalanceService: glbalanceservice.New(glbalanceservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}

	statement, err := resolver.IncomeStatement(
		resolvertest.AuthedContext(t, authCtx),
		periodID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, permission.ResourceAccountingReport.String(), engine.Request.Resource)
	assert.Equal(t, periodID, statement.FiscalPeriodID)
	assert.Equal(t, int64(900000), statement.Revenue.TotalMinor)
	assert.Equal(t, int64(250000), statement.OperatingExpense.TotalMinor)
	assert.Equal(t, int64(650000), statement.NetIncomeMinor)
}
