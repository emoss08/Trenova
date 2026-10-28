package permission

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var kindsGrantedOnTheirOwn = []permission.Resource{
	permission.ResourceWorkerInjury,
	permission.ResourceWorkerDOTTest,
	permission.ResourceBenefitPlan,
	permission.ResourceWorkerEmploymentEvent,
	permission.ResourceWorkerDisciplinaryAction,
	permission.ResourcePerformanceReview,
	permission.ResourceTimesheet,
	permission.ResourceProfileChangeRequest,
	permission.ResourceDriverPortal,
}

func TestCheck_AWorkerGrantReachesNoKindGrantedOnItsOwn(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	worker := permission.ResourceWorker.String()
	for _, kind := range kindsGrantedOnTheirOwn {
		t.Run(kind.String(), func(t *testing.T) {
			t.Parallel()

			eng, userID, orgID := cachedEngine(
				t,
				map[string]*permissionports.CachedResourcePermission{
					worker: cachedGrant(permission.DataScopeOrganization,
						registry.GetOperationsForResource(worker)...),
				},
			)
			check := func(resource string, op permission.Operation) bool {
				result, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
					UserID:         userID,
					OrganizationID: orgID,
					Resource:       resource,
					Operation:      op,
				})
				require.NoError(t, err)
				return result.Allowed
			}

			for _, op := range registry.GetOperationsForResource(kind.String()) {
				assert.False(t, check(kind.String(), op),
					"every worker operation, read and update among them, says nothing about %s:%s",
					kind, op)
			}
			assert.True(t, check(permission.ResourceWorkerCredential.String(), permission.OpRead),
				"the rest of the worker's family still follows the worker grant")
		})
	}
}

func TestCheck_AKindGrantedOnItsOwnAnswersToItsOwnGrant(t *testing.T) {
	t.Parallel()

	for _, kind := range kindsGrantedOnTheirOwn {
		t.Run(kind.String(), func(t *testing.T) {
			t.Parallel()

			eng, userID, orgID := cachedEngine(
				t,
				map[string]*permissionports.CachedResourcePermission{
					kind.String(): cachedGrant(permission.DataScopeTeam, permission.OpRead),
				},
			)
			check := func(resource permission.Resource, op permission.Operation) *permissionports.PermissionCheckResult {
				result, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
					UserID:         userID,
					OrganizationID: orgID,
					Resource:       resource.String(),
					Operation:      op,
				})
				require.NoError(t, err)
				return result
			}

			read := check(kind, permission.OpRead)
			assert.True(t, read.Allowed)
			assert.Equal(t, permission.DataScopeTeam, read.DataScope)
			assert.False(t, check(permission.ResourceWorker, permission.OpRead).Allowed,
				"a grant on %s says nothing about the worker it belongs to", kind)

			detail, err := eng.GetResourcePermissions(t.Context(), userID, orgID, kind.String())
			require.NoError(t, err)
			assert.Equal(t, []permission.Operation{permission.OpRead}, detail.Operations)
		})
	}
}
