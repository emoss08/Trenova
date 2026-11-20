package driverportalinvitationservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/portalaccessportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAnInvitationOpensOnlyForAnOrganizationWithTheDriverPortal(t *testing.T) {
	t.Parallel()

	invitation := &portalinvitation.PortalInvitation{
		ID:             pulid.MustNew("wpi_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Email:          "marisol.ortega@carrier.example",
		Status:         portalinvitation.PortalInvitationStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
		Worker:         &worker.Worker{FirstName: "Marisol", LastName: "Ortega"},
	}
	portalRepo := portalaccessportsmock.NewMockPortalAccessRepository(t)
	portalRepo.EXPECT().
		GetInvitationByTokenHash(mock.Anything, tokenutils.Hash("invite-token")).
		Return(invitation, nil).
		Times(2)

	suspended := &Service{
		l:          zap.NewNop(),
		portalRepo: portalRepo,
		features: plantest.SharedServer().
			Without(invitation.OrganizationID, platformcatalog.FeatureDriverPortal),
	}
	_, err := suspended.GetInvitationPreview(t.Context(), "invite-token")
	require.True(t, errortypes.IsPlanRestrictionError(err), "%v", err)

	open := &Service{
		l:          zap.NewNop(),
		portalRepo: portalRepo,
		features: plantest.SharedServer().
			Planned(invitation.OrganizationID).
			Without(pulid.MustNew("org_"), platformcatalog.FeatureDriverPortal),
	}
	preview, err := open.GetInvitationPreview(t.Context(), "invite-token")
	require.NoError(t, err, "another organization's plan does not count")
	assert.Equal(t, invitation.Email, preview.Email)
}
