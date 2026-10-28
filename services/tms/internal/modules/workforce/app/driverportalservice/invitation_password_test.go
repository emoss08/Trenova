package driverportalservice

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func requirePasswordFieldError(t *testing.T, err error) {
	t.Helper()
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "password", multiErr.Errors[0].Field)
}

func TestAcceptInvitationRefusesAWeakPasswordBeforeLookingUpTheToken(t *testing.T) {
	t.Parallel()

	for name, password := range map[string]string{
		"eight characters": "abcd-123",
		"common":           "password1234",
		"over 72 bytes":    strings.Repeat("ab1-", 19),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			portalRepo := workerportsmock.NewMockPortalAccessRepository(t)
			svc := &Service{l: zap.NewNop(), portalRepo: portalRepo}

			result, err := svc.AcceptInvitation(t.Context(), &AcceptInvitationRequest{
				Token:    "invite-token",
				Password: password,
			})

			assert.Nil(t, result)
			requirePasswordFieldError(t, err)
		})
	}
}

func TestAcceptInvitationRefusesAPasswordContainingTheInvitedEmail(t *testing.T) {
	t.Parallel()

	portalRepo := workerportsmock.NewMockPortalAccessRepository(t)
	invitation := &portalinvitation.PortalInvitation{
		ID:        pulid.MustNew("wpi_"),
		Email:     "marisol.ortega@carrier.example",
		Status:    portalinvitation.PortalInvitationStatusPending,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Worker:    &worker.Worker{FirstName: "Marisol", LastName: "Ortega"},
	}
	portalRepo.EXPECT().
		GetInvitationByTokenHash(mock.Anything, tokenutils.Hash("invite-token")).
		Return(invitation, nil).
		Once()
	svc := &Service{l: zap.NewNop(), portalRepo: portalRepo}

	result, err := svc.AcceptInvitation(t.Context(), &AcceptInvitationRequest{
		Token:    "invite-token",
		Password: "Marisol.Ortega#2026",
	})

	assert.Nil(t, result)
	requirePasswordFieldError(t, err)
	portalRepo.AssertNotCalled(t, "ActivatePortalAccess", mock.Anything, mock.Anything)
}
