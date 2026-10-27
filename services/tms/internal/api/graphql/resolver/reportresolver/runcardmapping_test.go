package reportresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReportRunDeskCardRefusesSomeoneWhoMayNotReadReports(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core: &base.Core{
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
			Type:        reportcatalog.FieldInt,
			Value:       30.0,
			WindowStart: 1_788_000_000,
			WindowEnd:   1_790_678_400,
		}},
		Timezone: "America/Chicago",
		Sample: &reportrows.Sample{
			Schema: []reportrows.Column{
				{ID: "customer", Label: "Customer", Type: reportcatalog.FieldString},
				{
					ID:      "revenue",
					Label:   "Revenue",
					Type:    reportcatalog.FieldDecimal,
					Format:  reportcatalog.FormatMoney,
					Display: reportfmt.Resolved{Style: reportfmt.StyleCurrency, Currency: "USD"},
				},
			},
			Rows:   [][]any{{"ACME", "1180.5"}},
			Totals: []any{nil, "1180.5"},
			More:   true,
		},
		Previous: &reportingservice.RunCardPrevious{
			RunID:       previousID,
			CompletedAt: 1_790_000_000,
			RowCount:    208,
			Schema:      []reportrows.Column{{ID: "revenue"}, {ID: "customer"}},
			Totals:      []any{"990", nil},
		},
		Chart: &report.ChartSpec{Type: report.ChartBar, XColumnID: "customer"},
		Sort:  []report.SortSpec{{ColumnID: "revenue", Direction: dbtype.SortDirectionDesc}},
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
	assert.Equal(t, "currency", model.Sample.Columns[1].Display.Style)
	require.NotNil(t, model.Sample.Columns[1].Format)
	assert.Equal(t, "money", *model.Sample.Columns[1].Format)
	assert.Equal(t, [][]any{{"ACME", "1180.5"}}, model.Sample.Rows)
	assert.Equal(t, []any{nil, "1180.5"}, model.Sample.Totals)
	assert.True(t, model.Sample.More)

	require.NotNil(t, model.Previous)
	assert.Equal(t, previousID.String(), model.Previous.RunID)
	assert.Equal(t, 208, model.Previous.RowCount)
	assert.Equal(t, []string{"revenue", "customer"}, model.Previous.ColumnIds)
	assert.Equal(t, []any{"990", nil}, model.Previous.Totals)

	require.NotNil(t, model.Chart)
	assert.Equal(t, "bar", model.Chart.Type)
	assert.Equal(t, []string{}, model.Chart.SeriesIds)
	assert.Equal(t, "desc", model.Sort[0].Direction)
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
