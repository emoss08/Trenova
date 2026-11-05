package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
)

// StubOrganizations answers with one organization, an empty one when unset.
type StubOrganizations struct {
	tenantports.OrganizationRepository

	Organization *tenant.Organization
	Err          error
}

func (r *StubOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.Organization == nil {
		return &tenant.Organization{}, nil
	}

	return r.Organization, nil
}
