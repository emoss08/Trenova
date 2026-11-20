package ediinboundas2

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/metricstest"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAnAS2MessageForAnOrganizationWithoutEDIGetsTheUnknownPairReceipt(t *testing.T) {
	t.Parallel()

	profile := &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("edicp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Method:         edi.ConnectionMethodAS2,
	}
	profileRepo := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileForInboundURL(mock.Anything, mock.Anything).
		Return(profile, nil).
		Once()

	service := New(Params{
		Features: plantest.SharedServer().
			Without(profile.OrganizationID, platformcatalog.FeatureEDIIntegration),
		Metrics:     metricstest.Disabled(),
		Logger:      zap.NewNop(),
		ProfileRepo: profileRepo,
		Delivery:    ediServices().Delivery,
		Secrets:     ediServices().Secrets,
	})

	result, err := service.ReceiveAS2Message(t.Context(), &ReceiveAS2MessageRequest{
		From:        "PARTNER-AS2",
		To:          "TRENOVA-AS2",
		ContentType: "application/edi-x12",
		Body:        []byte(as2ReceiverPayload),
	})

	require.NoError(t, err)
	require.True(t, result.Rejected, "nothing is staged for an organization without EDI")
	mdn, err := as2.ParseMDN(result.MDNContentType, result.MDNBody, nil)
	require.NoError(t, err)
	require.False(t, mdn.Signed, "answered exactly like an address that names no profile")
	require.Equal(t, as2.MDNErrorAuthenticationFailed, mdn.FailureText())
}
