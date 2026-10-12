package workerresolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestWorkerByIDRequest_LoadsOnlyTheRelationsSelected(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	selected := map[string]bool{
		"profile":              true,
		"profile.licenseState": true,
		"fleetCode":            true,
	}

	req := workerByIDRequest(func(path string) bool { return selected[path] }, id, tenant)

	assert.Equal(t, id, req.ID)
	assert.Equal(t, tenant, req.TenantInfo)
	assert.True(t, req.IncludeProfile)
	assert.True(t, req.IncludeLicenseState)
	assert.True(t, req.IncludeFleetCode)
	assert.False(t, req.IncludeManager)
	assert.False(t, req.IncludeState)
}

func TestWorkerByIDRequest_NeverReadsALicenseStateWithoutTheProfile(t *testing.T) {
	t.Parallel()

	req := workerByIDRequest(
		func(path string) bool { return path == "profile.licenseState" },
		pulid.MustNew("wrk_"),
		pagination.TenantInfo{},
	)

	assert.False(t, req.IncludeProfile)
	assert.False(t, req.IncludeLicenseState)
}
