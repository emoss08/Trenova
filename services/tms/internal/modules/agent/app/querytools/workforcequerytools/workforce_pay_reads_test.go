package workforcequerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/timesheetports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type payrollRuns struct {
	listed bool
}

func (p *payrollRuns) ListExports(
	context.Context,
	*timesheetports.ListPayrollExportsRequest,
) ([]*timesheet.PayrollExport, error) {
	p.listed = true

	return []*timesheet.PayrollExport{}, nil
}

type exportScoped struct {
	querytooltest.FakePermissions

	scope permission.DataScope
}

func (e *exportScoped) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	allowed := req.Resource == permission.ResourceTimesheet.String() &&
		req.Operation == permission.OpExport

	return &permissionports.PermissionCheckResult{Allowed: allowed, DataScope: e.scope}, nil
}

func TestListPayrollExports_NeedsExportAcrossTheOrganization(t *testing.T) {
	t.Parallel()

	runs := &payrollRuns{}
	tool := newListPayrollExportsTool(runs, &exportScoped{scope: permission.DataScopeTeam})

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), err.Error())
	assert.False(t, runs.listed)
}

func TestListPayrollExports_ListsForAnOrganizationGrant(t *testing.T) {
	t.Parallel()

	runs := &payrollRuns{}
	tool := newListPayrollExportsTool(runs, &exportScoped{scope: permission.DataScopeOrganization})

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))

	require.NoError(t, err)
	assert.True(t, runs.listed)
}
