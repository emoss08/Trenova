package fixturetest

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

func OrgStub(orgID, buID pulid.ID) *tenant.Organization {
	return &tenant.Organization{ID: orgID, BusinessUnitID: buID}
}
