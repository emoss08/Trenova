package carrierresolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCarrierByIDRequest_LoadsOnlyTheRelationsTheQuerySelects(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("car_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	requested := func(paths ...string) func(string) bool {
		return func(path string) bool {
			for _, candidate := range paths {
				if candidate == path {
					return true
				}
			}
			return false
		}
	}

	bare := carrierByIDRequest(requested(), id, tenant)
	assert.Equal(t, id, bare.ID)
	assert.Equal(t, tenant, bare.TenantInfo)
	assert.False(t, bare.IncludeState)
	assert.False(t, bare.IncludeContacts)
	assert.False(t, bare.IncludeInsurancePolicies)

	card := carrierByIDRequest(requested("contacts", "insurancePolicies"), id, tenant)
	assert.True(t, card.IncludeContacts)
	assert.True(t, card.IncludeInsurancePolicies)
	assert.False(t, card.IncludeState)

	assert.True(t, carrierByIDRequest(requested("remitState"), id, tenant).IncludeState)
	assert.True(t, carrierByIDRequest(requested("state"), id, tenant).IncludeState)
}
