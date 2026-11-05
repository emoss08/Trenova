package ediprofilesecret

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAReaderGetsTheInboundPathOfAnAS2ProfileAndNeverTheStoredToken(t *testing.T) {
	t.Parallel()

	profile := WithSecretState(&edi.EDICommunicationProfile{
		Method:          edi.ConnectionMethodAS2,
		AS2InboundToken: "PROFILEADDRESS",
	})

	assert.Equal(t, "/api/v1/edi/as2/inbound/PROFILEADDRESS/", profile.AS2InboundPath)
	assert.Empty(t, profile.AS2InboundToken)
}

func TestOnlyAnAS2ProfileHasAnInboundPath(t *testing.T) {
	t.Parallel()

	for _, method := range []edi.ConnectionMethod{
		edi.ConnectionMethodSFTP,
		edi.ConnectionMethodVAN,
		edi.ConnectionMethodInternal,
	} {
		profile := WithSecretState(&edi.EDICommunicationProfile{
			Method:          method,
			AS2InboundToken: "PROFILEADDRESS",
		})

		assert.Empty(t, profile.AS2InboundPath)
	}
}

func TestAProfileReadWithoutItsTokenHasNoInboundPath(t *testing.T) {
	t.Parallel()

	profile := WithSecretState(&edi.EDICommunicationProfile{Method: edi.ConnectionMethodAS2})

	assert.Empty(t, profile.AS2InboundPath)
}

func TestTheStoredInboundTokenStaysOutOfAProfilesJSON(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(&edi.EDICommunicationProfile{
		Method:          edi.ConnectionMethodAS2,
		AS2InboundToken: "PROFILEADDRESS",
	})

	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "PROFILEADDRESS")
}

func TestInboundTokensAreLongAndDistinct(t *testing.T) {
	t.Parallel()

	first, second := edi.NewAS2InboundToken(), edi.NewAS2InboundToken()

	assert.NotEqual(t, first, second)
	assert.Len(t, first, 52)
}
