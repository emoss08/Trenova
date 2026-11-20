package accountsreceivableservice

import (
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerledgerportsmock"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCustomerLedgerReportCompilesAgainstTheCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	limits := reportengine.DefinitionLimits(&reportingconfig.Config{})
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: pulid.ID("org_test"),
		BusinessUnitID: pulid.ID("bu_test"),
	})

	for name, opening := range map[string]int64{"owed": 430000, "nothing": 0, "in credit": -2500} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			definition := ledgerReport(&ledgerReportWindow{
				req: customerledgerports.ARLedgerPageRequest{
					CustomerID: pulid.ID("cus_1"),
					StartDate:  1767225600,
					EndDate:    1769903999,
				},
				openingMinor: opening,
			})
			compiled, compileErr := analytics.Compile(
				ctx,
				catalog,
				nil,
				definition,
				nil,
				analytics.CompileOptions{Limits: limits},
			)
			require.NoError(t, compileErr)

			labels := make([]string, 0, len(compiled.Columns))
			for _, column := range compiled.Columns {
				labels = append(labels, column.Label)
			}
			assert.Equal(
				t,
				[]string{"Date", "Document", "Event", "Debit", "Credit", "Balance"},
				labels,
				"the ledger's position and running total stay out of the file",
			)
		})
	}
}

func TestCustomerLedgerReportStartsFromTheOpeningBalance(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		GetCustomerLedgerBalance(mock.Anything, customerledgerports.GetCustomerLedgerBalanceRequest{
			TenantInfo: tenant,
			CustomerID: customerID,
			Before:     100,
		}).
		Return(int64(-750), nil).
		Once()
	svc := &Service{repo: repo}

	definition, err := svc.CustomerLedgerReport(
		t.Context(),
		tenant,
		customerledgerports.ARLedgerPageRequest{CustomerID: customerID, StartDate: 100, EndDate: 200},
	)

	require.NoError(t, err)
	byID := make(map[string]ir.Column, len(definition.Columns))
	for _, column := range definition.Columns {
		byID[column.ID] = column
	}
	require.NotNil(t, byID["balance"].Calc)
	assert.Equal(t, "[running_minor] - 750", byID["balance"].Calc.Text)
	assert.Equal(t, "position", byID["running_minor"].Window.Over)
	assert.Equal(t, []ir.Sort{{Column: "position"}}, definition.Sort)
	require.NotNil(t, definition.Filters)
	assert.Equal(t, []ir.Filter{
		{Field: "customer.id", Op: ir.OpEq, Value: customerID.String()},
		{Field: "transactionDate", Op: ir.OpLte, Value: int64(200)},
		{Field: "transactionDate", Op: ir.OpGte, Value: int64(100)},
	}, definition.Filters.And)
}

func TestCustomerLedgerReportFromTheFirstEntryReadsNoBalance(t *testing.T) {
	t.Parallel()

	svc := &Service{repo: customerledgerportsmock.NewMockAccountsReceivableRepository(t)}

	definition, err := svc.CustomerLedgerReport(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		customerledgerports.ARLedgerPageRequest{CustomerID: pulid.MustNew("cus_"), EndDate: 200},
	)

	require.NoError(t, err)
	require.Len(t, definition.Filters.And, 2)
	for _, column := range definition.Columns {
		if column.ID == "balance" {
			assert.Equal(t, "[running_minor]", column.Calc.Text)
		}
	}
}

func TestCustomerLedgerReportRefusesABackwardWindow(t *testing.T) {
	t.Parallel()

	svc := &Service{repo: customerledgerportsmock.NewMockAccountsReceivableRepository(t)}

	_, err := svc.CustomerLedgerReport(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		customerledgerports.ARLedgerPageRequest{
			CustomerID: pulid.MustNew("cus_"),
			StartDate:  300,
			EndDate:    200,
		},
	)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "startDate", validation.Field())
}
