package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestEDIConnectionsAndPartnersRefuseWhenThePlanRestrictsIntegrations(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityIntegrations)}
	tenant := plantest.Tenant()

	partner, err := svc.CreatePartner(t.Context(), &edi.EDIPartner{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}, nil)
	require.Nil(t, partner)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	connection, err := svc.CreateConnection(t.Context(), &CreateEDIConnectionRequest{TenantInfo: tenant}, nil)
	require.Nil(t, connection)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
