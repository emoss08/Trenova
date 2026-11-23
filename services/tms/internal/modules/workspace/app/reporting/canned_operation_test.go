package reporting

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var everyCaseReports = []string{
	"cases-handled-by-user",
	"watchtower-open-items-by-user",
	"watchtower-backlog-by-kind",
	"watchtower-resolution-reasons",
	"watchtower-response-times-by-week",
	"open-case-load-by-owner",
}

func ownCasesService(t *testing.T) *Service {
	t.Helper()
	svc, _ := engineService(t, engineOptions{
		withheld: map[permission.Resource]permission.Operation{
			permission.ResourceWatchtower: permission.OpManage,
		},
	})
	library, err := canned.Default()
	require.NoError(t, err)
	svc.canned = library
	svc.runRepo = idleRuns{}
	svc.cfg = &reportingconfig.Config{PreviewRowLimit: 40}
	return svc
}

func TestCannedLibrary_LeavesOutTheCaseReportsForSomeoneWhoSeesOnlyTheirOwnCases(t *testing.T) {
	t.Parallel()

	svc := ownCasesService(t)
	request := planRequest(testUserID())
	library, err := svc.CannedLibrary(t.Context(), &request)
	require.NoError(t, err)

	listed := make(map[string]bool, len(library))
	for i := range library {
		listed[library[i].Entry.Key] = true
	}
	for _, key := range everyCaseReports {
		assert.Falsef(t, listed[key], "%s is offered without See every case", key)
	}
	assert.True(t, listed["desk-token-usage-by-user"], "the usage report needs no case permission")
	assert.Len(t, library, len(svc.canned.All())-len(everyCaseReports))
}

func TestCannedLibrary_SeeEveryCaseListsTheCaseReports(t *testing.T) {
	t.Parallel()

	svc := cannedService(t, plantest.AllFeatures())
	request := planRequest(testUserID())
	library, err := svc.CannedLibrary(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, library, len(svc.canned.All()))
}

func TestCannedReport_CustomizingACaseReportWithoutSeeEveryCaseIsRefused(t *testing.T) {
	t.Parallel()

	svc := ownCasesService(t)
	request := planRequest(testUserID())

	_, err := svc.ForkCanned(t.Context(), &ForkCannedRequest{
		Request:   request,
		CannedKey: "cases-handled-by-user",
	})
	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Contains(t, err.Error(), "See every case")
}

func TestCannedReport_PreviewingACaseReportWithoutSeeEveryCaseIsRefused(t *testing.T) {
	t.Parallel()

	svc := ownCasesService(t)
	request := planRequest(testUserID())
	entry, err := svc.GetCanned("watchtower-open-items-by-user")
	require.NoError(t, err)

	err = svc.Check(t.Context(), &PrepareParams{
		Tenant:     request.TenantInfo,
		Definition: entry.Definition,
		Mode:       reportaccess.ModeRead,
	})
	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Contains(t, err.Error(), "See every case")
}

func TestCatalog_LeavesOutTheCaseDatasetsWithoutSeeEveryCase(t *testing.T) {
	t.Parallel()

	svc := ownCasesService(t)
	request := planRequest(testUserID())
	catalog, err := svc.Catalog(t.Context(), &request)
	require.NoError(t, err)

	keys := make(map[string]bool, len(catalog.Entities))
	for i := range catalog.Entities {
		keys[catalog.Entities[i].View.Key] = true
	}
	for _, key := range []string{"item", "resolution", "owner"} {
		assert.Falsef(t, keys[key], "%s is offered without See every case", key)
	}
	assert.True(t, keys["daily_usage"])
	assert.True(t, keys["person"])
	assert.Empty(t, catalog.WithheldByPlan, "a missing permission is not a plan restriction")
}
