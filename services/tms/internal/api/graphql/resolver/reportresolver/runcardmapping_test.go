package reportresolver

import (
	"testing"

	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReportRunDeskCardRefusesSomeoneWhoMayNotReadReports(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			L: zap.NewNop(),
			PermissionEngine: &mocks.DenyResourcesPermissionEngine{
				Denied: []string{permission.ResourceReport.String()},
			},
		},
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	_, err := resolver.ReportRunDeskCard(ctx, pulid.MustNew("rrun_").String())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestReportRunDeskCardCarriesTheSampleAndComparesTotalsByColumn(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("rdef_")
	previousID := pulid.MustNew("rrun_")
	card := &reportingservice.RunCard{
		Run: &report.ReportRun{
			ID:     pulid.MustNew("rrun_"),
			Status: report.RunStatusSucceeded,
			Format: report.FormatXLSX,
		},
		Report: reportingservice.RunCardReport{
			Name:         "Revenue by customer",
			Category:     "Billing",
			DefinitionID: definitionID,
		},
		Parameters: []reportingservice.RunCardParameter{{
			Name:        "windowDays",
			Label:       "Window (days)",
			Type:        "int",
			Value:       30.0,
			WindowStart: 1_788_000_000,
			WindowEnd:   1_790_678_400,
		}},
		Timezone: "America/Chicago",
		Sample: &reportingservice.PreviewResult{
			Columns: []table.Column{
				{ID: "customer", Label: "Customer", Type: value.String},
				{
					ID:      "revenue",
					Label:   "Revenue",
					Type:    value.Decimal,
					Display: format.Resolve(value.Decimal, format.Hint{Kind: format.Money}, nil),
				},
			},
			Rows:      [][]any{{"ACME", "1180.5"}, {nil, "1180.5"}},
			Levels:    []table.Level{table.Detail, table.Subtotal(1)},
			Totals:    []any{nil, "1180.5"},
			Truncated: true,
			Timezone:  "America/Chicago",
		},
		Previous: &reportingservice.RunCardPrevious{
			RunID:       previousID,
			CompletedAt: 1_790_000_000,
			RowCount:    208,
			ColumnIDs:   []string{"revenue", "customer"},
			Totals:      []any{"990", nil},
		},
		Chart: &ir.Chart{ID: "revenue", Type: ir.ChartBar, X: "customer"},
		Sort:  []ir.Sort{{Column: "revenue", Desc: true}},
	}

	model := reportRunDeskCardToModel(card)

	assert.Equal(t, "Revenue by customer", model.Report.Name)
	require.NotNil(t, model.Report.DefinitionID)
	assert.Equal(t, definitionID.String(), *model.Report.DefinitionID)
	assert.Nil(t, model.Report.CannedKey)
	assert.Nil(t, model.Withheld)

	require.Len(t, model.Parameters, 1)
	require.NotNil(t, model.Parameters[0].WindowStart)
	assert.Equal(t, 1_788_000_000, *model.Parameters[0].WindowStart)

	require.NotNil(t, model.Sample)
	require.Len(t, model.Sample.Columns, 2)
	assert.Equal(t, "decimal", model.Sample.Columns[1].Type)
	assert.Equal(t, "currency", model.Sample.Columns[1].Display.Style)
	assert.Equal(t, []any{[]any{"ACME", "1180.5"}, []any{nil, "1180.5"}}, model.Sample.Rows)
	assert.Equal(t, []int{0, 1}, model.Sample.Levels)
	assert.Equal(t, []any{nil, "1180.5"}, model.Sample.Totals)
	assert.True(t, model.Sample.Truncated)
	assert.Equal(t, "America/Chicago", model.Sample.Timezone)

	require.NotNil(t, model.Previous)
	assert.Equal(t, previousID.String(), model.Previous.RunID)
	assert.Equal(t, 208, model.Previous.RowCount)
	assert.Equal(t, []string{"revenue", "customer"}, model.Previous.ColumnIds)
	assert.Equal(t, []any{"990", nil}, model.Previous.Totals)

	require.NotNil(t, model.Chart)
	assert.Equal(t, "bar", model.Chart.Type)
	require.NotNil(t, model.Chart.X)
	assert.Equal(t, "customer", *model.Chart.X)
	assert.Equal(t, []string{}, model.Chart.Series)
	assert.Equal(t, []*gqlmodel.ReportRunDeskSort{{Column: "revenue", Desc: true}}, model.Sort)
}

func TestReportRunDeskCardWithoutASampleSaysWhyItIsWithheld(t *testing.T) {
	t.Parallel()

	model := reportRunDeskCardToModel(&reportingservice.RunCard{
		Run:        &report.ReportRun{ID: pulid.MustNew("rrun_")},
		Report:     reportingservice.RunCardReport{CannedKey: "revenue-by-customer"},
		Parameters: []reportingservice.RunCardParameter{},
		Withheld:   reportingservice.RunCardWithheldAccess,
	})

	assert.Nil(t, model.Sample)
	assert.Nil(t, model.Previous)
	assert.Nil(t, model.Chart)
	require.NotNil(t, model.Withheld)
	assert.Equal(t, "access", *model.Withheld)
	require.NotNil(t, model.Report.CannedKey)
	assert.Equal(t, "revenue-by-customer", *model.Report.CannedKey)
	assert.Empty(t, model.Sort)
}
