package edisetup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAnUpdateKeepsTheProfilesInboundAddress(t *testing.T) {
	t.Parallel()

	existing := as2Profile()
	existing.ID = pulid.MustNew("edicp_")
	existing.AS2InboundToken = "PROFILEADDRESS"

	service := &Service{l: zap.NewNop()}
	entity, err := service.buildCommunicationProfile(
		t.Context(),
		&UpsertEDICommunicationProfileRequest{
			ProfileID: existing.ID,
			Method:    edi.ConnectionMethodAS2,
			Name:      "Renamed",
			Config:    existing.Config,
		},
		existing,
	)

	require.NoError(t, err)
	assert.Equal(t, "PROFILEADDRESS", entity.AS2InboundToken)
}

func TestOnlyAnAS2ProfileHasAnInboundURLToRotate(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("edicp_")
	profiles := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	profiles.EXPECT().
		GetProfileByID(mock.Anything, mock.Anything).
		Return(sftpProfile(), nil).
		Once()

	_, err := (&Service{profileRepo: profiles, l: zap.NewNop()}).RotateAS2InboundURL(
		t.Context(),
		&RotateAS2InboundURLRequest{ProfileID: profileID, TenantInfo: pagination.TenantInfo{}},
		&permissionports.RequestActor{},
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	profiles.AssertNotCalled(t, "RotateAS2InboundToken", mock.Anything, mock.Anything)
}

func TestRotatingTheInboundURLSavesANewAddress(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("edicp_")
	existing := as2Profile()
	existing.ID = profileID
	existing.AS2InboundToken = "OLDADDRESS"
	rotated := as2Profile()
	rotated.ID = profileID
	rotated.AS2InboundToken = "NEWADDRESS"

	profiles := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	profiles.EXPECT().GetProfileByID(mock.Anything, mock.Anything).Return(existing, nil).Once()
	profiles.EXPECT().
		RotateAS2InboundToken(
			mock.Anything,
			mock.MatchedBy(func(req edipartnerports.RotateAS2InboundTokenRequest) bool {
				return req.ProfileID == profileID &&
					req.InboundToken != "" &&
					req.InboundToken != "OLDADDRESS"
			}),
		).
		Return(nil).
		Once()
	profiles.EXPECT().GetProfileByID(mock.Anything, mock.Anything).Return(rotated, nil).Once()

	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().
		LogAction(
			mock.MatchedBy(func(params *auditports.LogActionParams) bool {
				states := fmt.Sprint(params.PreviousState, params.CurrentState)
				return !strings.Contains(states, "OLDADDRESS") &&
					!strings.Contains(states, "NEWADDRESS")
			}),
			mock.Anything,
		).
		Return(nil).
		Once()

	result, err := (&Service{
		profileRepo:  profiles,
		auditService: audit,
		l:            zap.NewNop(),
	}).RotateAS2InboundURL(
		t.Context(),
		&RotateAS2InboundURLRequest{ProfileID: profileID},
		&permissionports.RequestActor{},
	)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/edi/as2/inbound/NEWADDRESS/", result.AS2InboundPath)
	assert.Empty(t, result.AS2InboundToken)
}
