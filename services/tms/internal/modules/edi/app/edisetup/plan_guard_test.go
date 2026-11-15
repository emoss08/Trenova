package edisetup

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestEDIConnectionsAndPartnersRefuseWhenTheEDIAddOnIsOff(t *testing.T) {
	t.Parallel()

	svc := &Service{
		features: plantest.Without(platformcatalog.FeatureEDIIntegration),
		secret:   ediprofilesecret.New(ediprofilesecret.Params{}),
	}
	tenant := plantest.Tenant()

	partner, err := svc.CreatePartner(t.Context(), &edi.EDIPartner{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}, nil)
	require.Nil(t, partner)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	connection, err := svc.CreateConnection(
		t.Context(),
		&CreateEDIConnectionRequest{TenantInfo: tenant},
		nil,
	)
	require.Nil(t, connection)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
