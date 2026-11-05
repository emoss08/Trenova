package editransport

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func as2ProfileWith(config map[string]any) *edi.EDICommunicationProfile {
	return &edi.EDICommunicationProfile{Method: edi.ConnectionMethodAS2, Config: config}
}

func TestAS2ConfigDecryptsWithTheSigningKeyWhenNoOtherIsSet(t *testing.T) {
	t.Parallel()

	local := newAS2TestIdentity(t, "trenova.example")

	cfg, err := AS2ConfigFromProfile(
		as2ProfileWith(map[string]any{ConfigKeyLocalCertificate: local.certificatePEM}),
		map[string]string{SecretKeyAS2PrivateKey: local.keyPEM},
	)

	require.NoError(t, err)
	assert.False(t, cfg.SeparateDecryptionKey)
	assert.Equal(t, cfg.LocalCertificate, cfg.DecryptionCertificate)
	assert.True(t, keyMatchesCertificate(cfg.DecryptionKey, cfg.LocalCertificate))
	assert.Equal(t,
		editransportports.EDIConnectionCheckWarning,
		testAS2DecryptionIdentity(cfg).Status,
	)
}

func TestAS2ConfigKeepsASeparateDecryptionKeyApart(t *testing.T) {
	t.Parallel()

	signing := newAS2TestIdentity(t, "signing.example")
	decryption := newAS2TestIdentity(t, "decryption.example")

	cfg, err := AS2ConfigFromProfile(
		as2ProfileWith(map[string]any{
			ConfigKeyLocalCertificate:           signing.certificatePEM,
			ConfigKeyLocalEncryptionCertificate: decryption.certificatePEM,
		}),
		map[string]string{
			SecretKeyAS2PrivateKey:           signing.keyPEM,
			SecretKeyAS2DecryptionPrivateKey: decryption.keyPEM,
		},
	)

	require.NoError(t, err)
	assert.True(t, cfg.SeparateDecryptionKey)
	assert.True(t, keyMatchesCertificate(cfg.PrivateKey, cfg.LocalCertificate))
	assert.True(t, keyMatchesCertificate(cfg.DecryptionKey, cfg.DecryptionCertificate))
	assert.False(t, keyMatchesCertificate(cfg.DecryptionKey, cfg.LocalCertificate))
	assert.NotEqual(t,
		editransportports.EDIConnectionCheckFailed,
		testAS2DecryptionIdentity(cfg).Status,
	)
}

func TestAS2DecryptionIdentityFailsAPairThatDoesNotMatch(t *testing.T) {
	t.Parallel()

	signing := newAS2TestIdentity(t, "signing.example")
	decryption := newAS2TestIdentity(t, "decryption.example")

	cfg, err := AS2ConfigFromProfile(
		as2ProfileWith(map[string]any{
			ConfigKeyLocalEncryptionCertificate: decryption.certificatePEM,
		}),
		map[string]string{SecretKeyAS2DecryptionPrivateKey: signing.keyPEM},
	)

	require.NoError(t, err)
	assert.Equal(t,
		editransportports.EDIConnectionCheckFailed,
		testAS2DecryptionIdentity(cfg).Status,
	)
}

func TestAS2DeliveryConfigRefusesHalfADecryptionPair(t *testing.T) {
	t.Parallel()

	decryption := newAS2TestIdentity(t, "decryption.example")
	base := map[string]any{
		ConfigKeyLocalAS2ID:   "TRENOVA",
		ConfigKeyPartnerAS2ID: "PARTNER",
		ConfigKeyEndpointURL:  "https://as2.partner.example/as2",
	}

	certificateOnly := map[string]any{
		ConfigKeyLocalEncryptionCertificate: decryption.certificatePEM,
	}
	for key, value := range base {
		certificateOnly[key] = value
	}
	cfg, err := AS2ConfigFromProfile(as2ProfileWith(certificateOnly), nil)
	require.NoError(t, err)
	require.ErrorContains(t, validateAS2DeliveryConfig(cfg), "decryption private key")

	cfg, err = AS2ConfigFromProfile(
		as2ProfileWith(base),
		map[string]string{SecretKeyAS2DecryptionPrivateKey: decryption.keyPEM},
	)
	require.NoError(t, err)
	require.ErrorContains(t, validateAS2DeliveryConfig(cfg), "encryption certificate")
}
