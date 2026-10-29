package reportdashboardservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type planDashboards struct {
	reportports.ReportDashboardRepository

	dashboard *report.Dashboard
}

func (r *planDashboards) GetByID(
	context.Context,
	*reportports.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	copied := *r.dashboard

	return &copied, nil
}

func TestPlanDeleteDashboard_OnlyTheOwner(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	dashboard := &report.Dashboard{ID: pulid.MustNew("rdb_"), Name: "Ops", OwnerID: owner}
	svc := &Service{l: zap.NewNop(), dashboardRepo: &planDashboards{dashboard: dashboard}}

	planned, err := svc.PlanDeleteDashboard(t.Context(), &GetDashboardRequest{
		Request: reporting.Request{
			TenantInfo: reportingtest.Tenant(owner),
		}, DashboardID: dashboard.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, dashboard.ID, planned.ID)

	_, err = svc.PlanDeleteDashboard(t.Context(), &GetDashboardRequest{
		Request: reporting.Request{
			TenantInfo: reportingtest.Tenant(pulid.MustNew("usr_")),
		}, DashboardID: dashboard.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)
}
