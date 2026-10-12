package orderresolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestOrderByIDRequest_LoadsOnlyTheRelationsTheQuerySelects(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ord_")
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

	card := orderByIDRequest(requested("customer", "charges"), id, tenant)
	assert.Equal(t, id, card.ID)
	assert.Equal(t, tenant, card.TenantInfo)
	assert.False(t, card.IncludeShipment, "a card that pages the legs never loads them all")
	assert.True(t, card.IncludeCustomer)
	assert.True(t, card.IncludeCharges)

	panel := orderByIDRequest(requested("legs"), id, tenant)
	assert.True(t, panel.IncludeShipment)
	assert.False(t, panel.IncludeCustomer)
	assert.False(t, panel.IncludeCharges)
}
