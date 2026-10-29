package smime

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

func TestCompressRoundTrip(t *testing.T) {
	t.Parallel()

	compressed, err := Compress([]byte(smimetest.SamplePayload))
	require.NoError(t, err)
	require.True(t, IsCompressedData(compressed))

	decompressed, err := Decompress(compressed)
	require.NoError(t, err)
	require.Equal(t, smimetest.SamplePayload, string(decompressed))
}

func TestParsePEMHelpers(t *testing.T) {
	t.Parallel()

	identity := smimetest.NewIdentity(t, "pem.example")
	certificatePEM := encodeCertificatePEM(t, identity.Certificate)
	keyPEM := encodePKCS8PEM(t, identity.Key)

	certificate, err := ParseCertificate(certificatePEM)
	require.NoError(t, err)
	require.True(t, certificate.Equal(identity.Certificate))

	key, err := ParsePrivateKey(keyPEM)
	require.NoError(t, err)
	require.IsType(t, &rsa.PrivateKey{}, key)

	_, err = ParseCertificate([]byte("not pem"))
	require.ErrorIs(t, err, ErrNoPEMBlock)
	_, err = ParsePrivateKey([]byte("not pem"))
	require.ErrorIs(t, err, ErrUnsupportedKeyType)
}

func encodeCertificatePEM(t *testing.T, certificate *x509.Certificate) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
}

func encodePKCS8PEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}
